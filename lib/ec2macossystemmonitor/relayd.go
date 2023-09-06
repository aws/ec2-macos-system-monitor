package ec2macossystemmonitor

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"hash/adler32"
	"net"
	"os"
	"sync/atomic"
	"time"
)

const SocketTimeout = 5 * time.Second

// BuildMessage takes a tag along with data for the tag and builds a byte slice to be sent to the relay.
//
// The tag is used as a way to namespace various payloads that are supported. Data is the payload and its format is
// specific to each tag. Each payload has the option to be compressed and this flag is part of the envelope created for
// sending data. The slice of bytes is passed back to the caller to allow flexibility to log the bytes if desired before
// passing to the relay via PassToRelayd
func BuildMessage(tag string, data string, compress bool) ([]byte, error) {
	var payload SerialPayload
	// This determines if the data will be passed in as provided or zlib compressed and then base64 encoded
	// Some payload will exceed the limit of what can be sent on the serial device, so compression allows more data
	// to be sent. base64 encoding allows safe characters only to be passed on the device
	if compress {
		var b bytes.Buffer
		w, err := zlib.NewWriterLevel(&b, 9)
		if err != nil {
			return []byte{}, fmt.Errorf("ec2macossystemmonitor: couldn't get compression writer: %s", err)
		}
		_, err = w.Write([]byte(data))
		if err != nil {
			return []byte{}, fmt.Errorf("ec2macossystemmonitor: couldn't copy compressed data: %s", err)
		}
		err = w.Close()
		if err != nil {
			return []byte{}, fmt.Errorf("ec2macossystemmonitor: couldn't close compressor: %s", err)
		}

		encodedData := base64.StdEncoding.EncodeToString(b.Bytes())
		payload = SerialPayload{
			Tag:      tag,
			Compress: compress,
			Data:     encodedData,
		}
	} else {
		// No compression needed, simply create the SerialPayload
		payload = SerialPayload{
			Tag:      tag,
			Compress: compress,
			Data:     data,
		}
	}
	// Once the payload is created, it's converted to json
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return []byte{}, fmt.Errorf("ec2macossystemmonitor: couldn't get %s into json", err)
	}
	// A checksum is computed on the json payload for the serial message
	checkSum := adler32.Checksum(payloadBytes)
	message := SerialMessage{
		Csum:    checkSum,
		Payload: string(payloadBytes),
	}
	// Once the message is created, it's converted to json
	messageBytes, err := json.Marshal(message)
	if err != nil {
		return []byte{}, fmt.Errorf("ec2macossystemmonitor: couldn't convert %s into json", err)
	}
	messageBytes = append(messageBytes, "\n"...)

	return messageBytes, nil
}

// PassToRelayd takes a byte slice and writes it to a UNIX socket to send for relaying.
func PassToRelayd(messageBytes []byte) (n int, err error) {
	// The socket file needs to be created to write, the server creates this file.
	if !fileExists(SocketPath) {
		return 0, fmt.Errorf("ec2macossystemmonitor: %s does not exist, cannot send message: %s", SocketPath, string(messageBytes))
	}
	// Finally write the serial message to the domain socket
	sock, err := net.Dial("unix", SocketPath)
	if err != nil {
		return 0, fmt.Errorf("cec2macossystemmonitor: could not connect to %s: %s", SocketPath, err)
	}
	defer sock.Close()

	_, err = sock.Write(messageBytes)
	if err != nil {
		return 0, fmt.Errorf("ec2macossystemmonitor: error while writing to socket: %s", err)
	}

	// Return the length of the bytes written to the socket
	return len(messageBytes), nil
}

// SendMessage takes a tag along with data for the tag and writes to a UNIX socket to send for relaying. This is provided
// for convenience to allow quick sending of data to the relay. It calls BuildMessage and then PassToRelayd in order.
func SendMessage(tag string, data string, compress bool) (n int, err error) {
	msgBytes, err := BuildMessage(tag, data, compress)
	if err != nil {
		return 0, fmt.Errorf("ec2macossystemmonitor: error while building message bytes: %s", err)
	}

	return PassToRelayd(msgBytes)
}

// SerialRelay contains the serial connection and UNIX domain socket listener as well as the channel that communicates
// that the resources can be closed.
type SerialRelay struct {
	serialConnection SerialConnection // serialConnection is the managed serial device connection for writing
	listener         net.UnixListener // listener is the UNIX domain socket UnixzzListener for reading
	ReadyToClose     chan bool        // ReadyToClose is the channel for communicating the need to close connections
}

// NewRelay creates an instance of the relay server and returns a SerialRelay for manual closing.
//
// The SerialRelay returned from NewRelay is designed to be used in a go routine by using StartRelay. This allows the
// caller to handle OS Signals and other events for clean shutdown rather than relying upon defer calls.
func NewRelay(serialDevice string) (relay SerialRelay, err error) {
	// Create a serial connection
	serCon, err := NewSerialConnection(serialDevice)
	if err != nil {
		return SerialRelay{}, fmt.Errorf("relayd: failed to build a connection to serial interface: %s", err)
	}

	// Clean the socket in case its stale
	if err = os.RemoveAll(SocketPath); err != nil {
		if _, ok := err.(*os.PathError); ok {
			// Help guide that the SocketPath is invalid
			return SerialRelay{}, fmt.Errorf("relayd: unable to clean %s: %s", SocketPath, err)
		} else {
			// Unknown issue, return the error directly
			return SerialRelay{}, err
		}

	}

	// Create a listener on the socket by getting the address and then creating a Unix Listener
	addr, err := net.ResolveUnixAddr("unix", SocketPath)
	if err != nil {
		return SerialRelay{}, fmt.Errorf("relayd: unable to resolve address: %s", err)
	}
	listener, err := net.ListenUnix("unix", addr)
	if err != nil {
		return SerialRelay{}, fmt.Errorf("relayd: unable to listen on socket: %s", err)
	}
	// Create the SerialRelay to return
	relay.listener = *listener
	relay.serialConnection = *serCon
	// Create the channel for sending an exit
	relay.ReadyToClose = make(chan bool)
	return relay, nil
}

// StartRelay takes the connections for the serial relay and begins listening.
//
// This is a server implementation of the SerialRelay so it logs to a provided logger, and empty logger can be provided
// to stop logging if desired. This function is designed to be used in a go routine so logging may be the only way to
// get data about behavior while it is running. The resources can be shut down by sending true to the ReadyToClose
// channel. This invokes CleanUp() which is exported in case the caller desires to call it instead.
func (relay *SerialRelay) StartRelay(logger *Logger, relayStatus *StatusLogBuffer) {
	for {

		// Accept new connections, dispatching them to relayServer in a goroutine.
		err := relay.listener.SetDeadline(time.Now().Add(SocketTimeout))
		if err != nil {
			logger.Fatal("Unable to set deadline on socket:", err)
		}
		socCon, err := relay.listener.Accept()
		// Look for signal to exit, otherwise keep going, check the error only if we aren't supposed to shutdown
		select {
		case <-relay.ReadyToClose:
			logger.Info("[relayd] requested to shutdown")
			// Clean up resources manually
			relay.CleanUp()
			// Return to stop the connections from continuing
			return
		default:
			// If ReadyToClose has not been sent, then check for errors, handle timeouts, otherwise process
			if err != nil {
				if er, ok := err.(net.Error); ok && er.Timeout() {
					// This is just a timeout, break the loop and go to the top to start listening again
					continue
				} else {
					// This is some other error, for Accept(), its a fatal error if we can't Accept()
					logger.Fatal("Unable to start accepting on socket:", err)
				}
			}

		}
		// Write the date to the relay
		written, err := relay.serialConnection.RelayData(socCon)
		if err != nil {
			logger.Errorf("Failed to send data: %s\n", err)
		}
		// Increment the counter
		atomic.AddInt64(&relayStatus.Written, int64(written))
	}
}

// CleanUp manually closes the connections for a Serial Relay. This is called from StartRelay when true is sent on
// ReadyToClose so it should only be called separately if closing outside of that mechanism.
func (relay *SerialRelay) CleanUp() {
	_ = relay.listener.Close()
	_ = relay.serialConnection.Close()
	_ = os.RemoveAll(SocketPath)
}
