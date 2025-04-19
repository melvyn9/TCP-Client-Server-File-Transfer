package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
)

var serverAddr string

func main() {
	// Defined flags port and host. These are changeable at the command line.
	// Flags must be before the mode or filename arguments other this will not work.
	// Don't ask me how close I got to ripping my hair out because of this.
	port := flag.Int("port", 8080, "Port to accept connections on")

	//localhost also OK
	//TODO 1: set up a string flag named "host" with a default value of 127.0.0.1 (localhost is also OK).
	// Include a brief description of its usage in the usage argument.
	host := flag.String("host", "127.0.0.1", "Host to bind to")

	// Parse flags first
	flag.Parse()

	// Reconstruct the server address
	serverAddr = *host + ":" + strconv.Itoa(*port)

	// Grab non-flag arguments (i.e. mode and/or filename)
	args := flag.Args()

	// check we have at least a mode argument
	// note the numbers are a bit different compared to using os.args as "main.go" doesn't count as an argument.
	if len(args) < 1 {
		fmt.Println("Usage:")
		fmt.Println("  Server mode: go run main.go [-host <host>] [-port <port>] server")
		fmt.Println("  Client mode: go run main.go [-host <host>] [-port <port>] client <filename>")
		os.Exit(1)
	}

	// gets mode (server or client) and determined which mode to run in
	mode := args[0]
	switch mode {
	case "server":
		runServer()
	case "client":
		// check at this point that you have some kind of filename (we will check later if the filename is actually valid or not)
		if len(args) < 2 {
			fmt.Println("Error: Missing filename for client mode")
			fmt.Println("  Client mode: go run main.go [-host <host>] [-port <port>] client <filename>")
			os.Exit(1)
		}
		//extract the filename (recall args[0] was the mode)
		filename := args[1]
		runClient(filename)
	default:
		// Handle unknown mode
		fmt.Println("Usage:")
		fmt.Println("  Server mode: go run main.go [-host <host>] [-port <port>] server")
		fmt.Println("  Client mode: go run main.go [-host <host>] [-port <port>] client <filename>")
		os.Exit(1)
	}
}

func runServer() {
	// Starts a TCP server listening on serverAddr
	// TODO 2: as written above. Make sure both listener, and err variables are correctly defined so the rest of the code doesn't throw errors.
	// serverAddr was already defined in main, and we will use TCP for this.
	listener, err := net.Listen("tcp", serverAddr)
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Printf("Server listening on %s\n", serverAddr)

	for {
		// Accept incoming connections
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Failed to accept connection: %v\n", err)
			continue
		}

		fmt.Printf("New connection from %s\n", conn.RemoteAddr())

		// TODO 9: uncomment this before writing handleConnection
		go handleConnection(conn) // Handle each connection in a separate goroutine
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Reads the length of the incoming filename (4 bytes, big-endian)
	var filenameLen uint32
	err := binary.Read(conn, binary.BigEndian, &filenameLen)
	if err != nil {
		fmt.Printf("Failed to read filename length: %v\n", err)
		return
	}

	// Reads the filename based on the length
	filenameBytes := make([]byte, filenameLen)
	_, err = io.ReadFull(conn, filenameBytes)
	if err != nil {
		fmt.Printf("Failed to read filename: %v\n", err)
		return
	}
	filename := string(filenameBytes)

	// Ensure only the base name is used (avoid writing files to unexpected directories)
	filename = filepath.Base(filename)

	// Read the length of the incoming content (4 bytes, big-endian)
	var contentLen uint32
	// TODO 10: use binary.Read to proprely define contentLen.
	// We are using binary.BigEndian for this, as well as the connection defined as "conn"
	// make sure err is defined as well.
	err = binary.Read(conn, binary.BigEndian, &contentLen)
	if err != nil {
		fmt.Printf("Failed to read content length: %v\n", err)
		return
	}

	fmt.Printf("Receiving file: %s (%d bytes)\n", filename, contentLen)

	// Ensures a "server-storage" directory exists, or creates one if missing.
	sStorageDir := "server-storage"
	err = os.MkdirAll(sStorageDir, 0755)
	if err != nil {
		fmt.Printf("Failed to create server storage directory: %v\n", err)
		return
	}

	// Creates a new file in the "downloads" directory with the received name
	filename = filepath.Join(sStorageDir, filename)
	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Failed to create file: %v\n", err)
		return
	}
	defer file.Close()

	// Reads the file content in chunks and write it to the file
	buffer := make([]byte, 4096)
	var totalRead uint32 = 0

	// TODO 11: what should this for loop be?
	for totalRead < contentLen {
		readSize := uint32(len(buffer))

		//adjusts buffer size if the remaining # of bytes is less than 4096
		if contentLen-totalRead < readSize {
			readSize = contentLen - totalRead
		}

		//TODO 12: use io.ReadFull to read into our buffer (note readSize gives us an idea of up to how many bytes we can read)
		bytesRead, err := io.ReadFull(conn, buffer[:readSize])
		if err != nil {
			fmt.Printf("Failed to read content: %v\n", err)
			return
		}

		// TODO 13: use Golang file io to write contents from the buffer into the resulting file (variable name is "file")
		// we will not use the int return value for anything in this sample code, you may blank it out if you wish.
		_, err = file.Write(buffer[:bytesRead])
		if err != nil {
			fmt.Printf("Failed to write to file: %v\n", err)
			return
		}

		totalRead += uint32(bytesRead)
		fmt.Printf("\rProgress: %.2f%%", float64(totalRead)/float64(contentLen)*100)
	}

	fmt.Println("\nFile received successfully")
}

func runClient(filename string) {
	// TODO 3: read through the commments to understand whats happening
	// Opens the file to be sent (note your client's files MUST be in a folder called client-storage)
	filename = "client-storage/" + filename
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Failed to open file %s: %v\n", filename, err)
		os.Exit(1)
	}
	defer file.Close()

	// Extracts and prepare the base filename (ex: converts assets/car1.jpg to car1.jpg)
	baseFilename := filepath.Base(filename)
	filenameBytes := []byte(baseFilename)

	// Connect to the server
	// TODO 4: as written above. Again, we are using tcp and serverAddr was defined already.
	conn, err := net.Dial("tcp", serverAddr)
	if err != nil {
		fmt.Printf("Failed to connect to server: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	// Send the filename length (ex: car1.jpg is 8 bytes)
	err = binary.Write(conn, binary.BigEndian, uint32(len(filenameBytes)))
	if err != nil {
		fmt.Printf("Failed to send filename length: %v\n", err)
		os.Exit(1)
	}

	// Send the filename
	_, err = conn.Write(filenameBytes)
	if err != nil {
		fmt.Printf("Failed to send filename: %v\n", err)
		os.Exit(1)
	}

	// Retrieve file info (e.g., size)
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Printf("Failed to get file info: %v\n", err)
		os.Exit(1)
	}

	// Send the content length as a 4 byte integer
	contentLen := uint32(fileInfo.Size())
	// TODO 5: send the content length data over the connection, make sure err is defined.
	// we will use binary.BigEndian for everything
	err = binary.Write(conn, binary.BigEndian, contentLen)
	if err != nil {
		fmt.Printf("Failed to send content length: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Sending file: %s (%d bytes)\n", baseFilename, contentLen)

	// Send the file contents in chunks
	// TODO 6: Create a byte/uint8-based buffer with a large enough size (4096 recommended)
	buffer := make([]byte, 4096)
	var totalSent uint32 = 0

	for totalSent < contentLen {
		// TODO 7: read bytes from the file (variable name is also "file") into the buffer created in TODO 6.
		// make sure bytesRead and err are defined
		bytesRead, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			fmt.Printf("Failed to read from file: %v\n", err)
			os.Exit(1)
		}

		if bytesRead == 0 {
			break
		}

		// TODO 8: read bytes from the buffer (limit yourself to bytesRead # of bytes) and write them to the connection.
		// you must define bytesWritten and err for things to work.
		bytesWritten, err := conn.Write(buffer[:bytesRead])
		if err != nil {
			fmt.Printf("Failed to send data: %v\n", err)
			os.Exit(1)
		}

		totalSent += uint32(bytesWritten)
		fmt.Printf("\rProgress: %.2f%%", float64(totalSent)/float64(contentLen)*100)
	}

	fmt.Println("\nFile sent successfully")
}
