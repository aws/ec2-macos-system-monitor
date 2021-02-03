package ec2macossystemmonitor

import (
	"bytes"
	"fmt"
	"io"
	"net"

	"go.bug.st/serial"
)

// SocketPath is the default socket for relayd.
const SocketPath = "/tmp/.ec2monitoring.sock"

// SerialConnection is the container for passing the ReadWriteCloser for serial connections.
type SerialConnection struct {
	port serial.Port
}

// SerialPayload is the container for a payload that is written to serial device.
type SerialPayload struct {
	// Tag is the namespace that separates different types of data on the device
	Tag string `json:"tag"`
	// Compress determines if the data is compressed and base64 encoded
	Compress bool `json:"compress"`
	// Data is the actual data payload to be consumed
	Data string `json:"data"`
}

// SerialMessage is the container to actually send on the serial connection, contains checksum of SerialPayload to
// provide additional assurance the entire payload has been written.
type SerialMessage struct {
	// Csum is the checksum used to ensure all data was received
	Csum uint32 `json:"csum"`
	// Payload is the SerialPayload in json format
	Payload string `json:"payload"`
}

// CheckSocketExists is a helper function to quickly check for the server.
func CheckSocketExists() (exists bool) {
	return fileExists(SocketPath)
}

// NewSerialConnection creates a serial device connection and returns a reference to the connection.
func NewSerialConnection(device string) (conn *SerialConnection, err error) {
	// Set up options for serial device, take defaults for now on everything else
	mode := &serial.Mode{
		BaudRate: 115200,
	}

	// Attempt to avoid opening a non-existent serial connection
	if !fileExists(device) {
		return nil, fmt.Errorf("ec2macossystemmonitor: serial device does not exist: %s", device)
	}
	// Open the serial port
	port, err := serial.Open(device, mode)
	if err != nil {
		return nil, fmt.Errorf("ec2macossystemmonitor: unable to get serial connection: %s", err)
	}
	// Put the port in a SerialConnection for handing it off
	s := SerialConnection{port}
	return &s, nil
}

// Close is simply a pass through to close the device so it remains open in the scope needed.
func (s *SerialConnection) Close() (err error) {
	err = s.port.Close()
	if err != nil {
		return err
	}
	return nil
}

// RelayData is the primary function for reading data from the socket provided and writing to the serial connection.
func (s *SerialConnection) RelayData(sock net.Conn) (n int, err error) {
	defer sock.Close()
	// Create a buffer for reading in from the socket, probably want to bound this
	var buf bytes.Buffer
	// Read in the socket data into the buffer
	_, err = io.Copy(&buf, sock)
	if err != nil {
		return 0, fmt.Errorf("ec2macossystemmonitor: failed to read socket to buffer: %s", err)
	}
	// Write out the buffer to the serial device
	written, err := s.port.Write(buf.Bytes())
	if err != nil {
		return 0, fmt.Errorf("ec2macossystemmonitor: failed to write buffer to serial: %s", err)
	}
	return written, nil
}
