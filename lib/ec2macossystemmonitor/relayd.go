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

// DefaultRelaydSocketPath is the default socket for relayd listener.
const DefaultRelaydSocketPath = "/tmp/.ec2monitoring.sock"

// CheckSocketExists is a helper function to quickly check if the service UDS
// exists.
func CheckSocketExists(socketPath string) (exists bool) {
	return fileExists(socketPath)
}

// BuildMessage takes a tag along with data for the tag and builds a byte slice to be sent to the relay.
//
// The tag is used as a way to namespace various payloads that are supported. Data is the payload and its format is
// specific to each tag. Each payload has the option to be compressed and this flag is part of the envelope created for
// sending data. The slice of bytes is passed back to the caller to allow flexibility to log the bytes if desired before
// passing to the relay via PassToRelayd
func BuildMessage(tag string, data string, compress bool) ([]byte, error) {
	payload := SerialPayload{
		Tag: tag,
		Compress: compress,
		Data: data,
	}

	// This determines if the data will be passed in as provided or zlib compressed and then base64 encoded
	// Some payload will exceed the limit of what can be sent on the serial device, so compression allows more data
	// to be sent. base64 encoding allows safe characters only to be passed on the device
	if compress {
		var b bytes.Buffer
		w, err := zlib.NewWriterLevel(&b, 9)
		if err != nil {
			return nil, fmt.Errorf("ec2macossystemmonitor: couldn't get compression writer: %w", err)
		}
		_, err = w.Write([]byte(data))
		if err != nil {
			return nil, fmt.Errorf("ec2macossystemmonitor: couldn't copy compressed data: %w", err)
		}
		err = w.Close()
		if err != nil {
			return nil, fmt.Errorf("ec2macossystemmonitor: couldn't close compressor: %w", err)
		}

		payload.Data = base64.StdEncoding.EncodeToString(b.Bytes())
	}

	// Marshal the payload to wrap in the relay output message.
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("ec2macossystemmonitor: %w", err)
	}

	messageBytes, err := json.Marshal(SerialMessage{
		Checksum: adler32.Checksum(payloadBytes),
		Payload:  string(payloadBytes),
	})
	if err != nil {
		return nil, fmt.Errorf("ec2macossystemmonitor: marshal: %w", err)
	}

	// FIXME: message shouldn't append the newline, that's up to clients to
	// decide (ie: flushing data as needed to clients/servers).
	messageBytes = append(messageBytes, "\n"...)

	return messageBytes, nil
}

// PassToRelayd takes a byte slice and writes it to a UNIX socket to send for relaying.
func PassToRelayd(messageBytes []byte) (n int, err error) {
	// Make sure we have socket to connect to.
	if !fileExists(DefaultRelaydSocketPath) {
		return 0, fmt.Errorf("ec2macossystemmonitor: %s does not exist, cannot send message: %s", DefaultRelaydSocketPath, string(messageBytes))
	}

	// Connect and relay!
	sock, err := net.Dial("unix", DefaultRelaydSocketPath)
	if err != nil {
		return 0, fmt.Errorf("cec2macossystemmonitor: could not connect to %s: %s", DefaultRelaydSocketPath, err)
	}
	defer sock.Close()

	n, err = sock.Write(messageBytes)
	if err != nil {
		return n, fmt.Errorf("ec2macossystemmonitor: error while writing to socket: %s", err)
	}

	return n, nil
}

// SendMessage takes a tag along with data for the tag and writes to a UNIX socket to send for relaying. This is provided
// for convenience to allow quick sending of data to the relay. It calls BuildMessage and then PassToRelayd in order.
func SendMessage(tag string, data string, compress bool) (n int, err error) {
	msgBytes, err := BuildMessage(tag, data, compress)
	if err != nil {
		return 0, fmt.Errorf("ec2macossystemmonitor: error while building message bytes: %w", err)
	}

	return PassToRelayd(msgBytes)
}

// SerialRelay manages client & listener to relay recieved messages to a serial
// connection.
type SerialRelay struct {
	// serialConnection is the managed serial device connection for writing
	// (ie: relayed output).
	serialConnection *SerialConnection
	// listener handles connections to relay received messages to the configured
	// serialConnection.
	listener net.Listener
	// ReadyToClose is the channel for communicating the need to close
	// connections.
	//
	// TODO: use context as replacement for cancellation
	ReadyToClose chan bool
}

// NewRelay creates an instance of the relay server and returns a SerialRelay for manual closing.
//
// The SerialRelay returned from NewRelay is designed to be used in a go routine by using StartRelay. This allows the
// caller to handle OS Signals and other events for clean shutdown rather than relying upon defer calls.
func NewRelay(serialDevice string) (relay SerialRelay, err error) {
	const socketPath = DefaultRelaydSocketPath

	// Create a serial connection
	serCon, err := NewSerialConnection(serialDevice)
	if err != nil {
		return SerialRelay{}, fmt.Errorf("relayd: failed to build a connection to serial interface: %w", err)
	}

	// Remove
	if err = os.RemoveAll(socketPath); err != nil {
		if _, ok := err.(*os.PathError); ok {
			// Help guide that the SocketPath is invalid
			return SerialRelay{}, fmt.Errorf("relayd: unable to clean %s: %w", socketPath, err)
		} else {
			// Unknown issue, return the error directly
			return SerialRelay{}, err
		}

	}

	// Create the UDS listener.
	addr, err := net.ResolveUnixAddr("unix", DefaultRelaydSocketPath)
	if err != nil {
		return SerialRelay{}, fmt.Errorf("relayd: unable to resolve address: %w", err)
	}
	listener, err := net.ListenUnix("unix", addr)
	if err != nil {
		return SerialRelay{}, fmt.Errorf("relayd: unable to listen on socket: %w", err)
	}

	return SerialRelay{
		listener:         listener,
		serialConnection: serCon,
		ReadyToClose:     make(chan bool),
	}, nil
}

// setListenerDeadline will set a deadline on the underlying net.Listener if
// supported, no-op otherwise.
func (relay *SerialRelay) setListenerDeadline(t time.Time) error {
	deadliner, ok := relay.listener.(interface{
		SetDeadline(time.Time) error
	})
	if ok {
		return deadliner.SetDeadline(t)
	}

	return nil
}

// StartRelay starts the listener ahdn handles connections for the serial relay.
//
// This is a server implementation of the SerialRelay so it logs to a provided
// logger, and empty logger can be provided to stop logging if desired. This
// function is designed to be used in a go routine so logging may be the only
// way to get data about behavior while it is running. The resources can be shut
// down by sending true to the ReadyToClose channel. This invokes CleanUp()
// which is exported in case the caller desires to call it instead.
func (relay *SerialRelay) StartRelay(logger *Logger, relayStatus *StatusLogBuffer) {
	// Accept new connections, dispatching them to relayServer in a goroutine.
	for {
		err := relay.setListenerDeadline(time.Now().Add(SocketTimeout))
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

	_ = os.RemoveAll(DefaultRelaydSocketPath)
}
