package goavatar

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// ----------------------------------------------------------------- //
// Device
// ----------------------------------------------------------------- //

// Device represents an AvatarEEG device (or a mock device).
type Device interface {

	// Connect to the device and return the output channel.
	// Connecting to a device that is already connected is
	// an error.
	Connect() (<-chan *DataFrame, error)

	// Connected returns true if and only if the device is
	// currently connected.
	Connected() bool

	// Disconnects from the device, closes the output channel,
	// and cleans relevant resources. Calls to disconnect are
	// idempotent.
	Disconnect()

	// Returns the output channel for the device. If the
	// device has not been connected, the value of the
	// channel is nil. If the device has been disconnected
	// the channel will be closed.
	Out() <-chan *DataFrame
}

// ----------------------------------------------------------------- //
// AvatarEEG Device
// ----------------------------------------------------------------- //

type AvatarDevice struct {
	serialPort string          // serial port like /dev/tty.AvatarEEG03009-SPPDev
	offSignal  chan bool       // send a value to disconnect the device
	reader     io.ReadCloser   // the reader of the serial port
	output     chan *DataFrame // channel that delivers raw Avatar output
	lock       sync.Mutex      // for synchronizing calls to control the device
	connected  bool            // connection status
}

// NewAvatarDevice creates a new AvatarEEG connection. The user 
// can then start streaming data by calling Connect() and reading the 
// output channel.
func NewAvatarDevice(serialPort string) *AvatarDevice {
	return &AvatarDevice{
		serialPort: serialPort,
		offSignal:  make(chan bool),
	}
}

func (d *AvatarDevice) Connected() bool {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.connected
}

func (d *AvatarDevice) Connect() (output <-chan *DataFrame, err error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.connected {
		return nil, fmt.Errorf("already connected to the device")
	}

	// connect to the reader for the port; this will
	// fail if we are already reading from this port
	reader, err := os.Open(d.serialPort)
	if err != nil {
		return nil, fmt.Errorf("cannot connect: %v", err)
	}
	d.reader = reader
	d.output = make(chan *DataFrame, DataBufferSize)

	go func() {
		parseByteStream(d.reader, d.offSignal, d.output)
	}()

	d.connected = true
	return d.output, nil
}

func (d *AvatarDevice) Disconnect() {
	d.lock.Lock()
	defer d.lock.Unlock()

	if !d.connected {
		return
	}

	// send the off signal; will block until the
	// offSignal is processed on the output thread
	d.offSignal <- true
	d.reader.Close() // best-effort
	close(d.output)
	d.connected = false
}

func (d *AvatarDevice) Out() <-chan *DataFrame {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.output
}
