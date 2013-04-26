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
	Disconnect() error

	// Returns the output channel for the device. If the
	// device has not been connected, the value of the
	// channel is nil. If the device has been disconnected
	// the channel will be closed.
	Out() <-chan *DataFrame

	// Starts recording the streaming data to a file.
	Record(file string) (err error)

	// Stops recording the streaming data.
	Stop()
}

// ConnectFunc performs the low-level operation to connect
// to the device
type ConnectFunc func() (chan *DataFrame, error)

// DisconnectFunc perfoms the low-level operation to disconnect
// from the device
type DisconnectFunc func() error

// StreamFunc performs the operation of reading the stream and
// writing data frames to the output channel, while also listening
// for recording signals.
type StreamFunc func(<-chan bool, chan<- *DataFrame)

// baseDevice
type baseDevice struct {
	offSignal chan bool
	out       chan *DataFrame
	lock      sync.Mutex
	connected bool

	// low-level ops
	connFunc    ConnectFunc
	disconnFunc DisconnectFunc
	streamFunc  StreamFunc
}

func (d *baseDevice) Connect() (out <-chan *DataFrame, err error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	// check connection
	if d.connected {
		return nil, fmt.Errorf("already connected to the device")
	}

	// perform connect
	d.out, err = d.connFunc()
	if err != nil {
		return nil, fmt.Errorf("could not connect: %v", err)
	}

	// begin to stream
	go func() {
		d.streamFunc(d.offSignal, d.out)
	}()

	// mark connected
	d.connected = true
	return d.out, nil
}

func (d *baseDevice) Disconnect() (err error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	// check for idempotency
	if !d.connected {
		return
	}

	// send the off signal; will block until the
	// offSignal is processed on the output thread
	d.offSignal <- true
	close(d.out)

	// disconnect
	err = d.disconnFunc()
	d.connected = false

	return err
}

func (d *baseDevice) Out() <-chan *DataFrame {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.out
}

func (d *baseDevice) Record(file string) (err error) {
	return
}

func (d *baseDevice) Stop() {

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

func (d *AvatarDevice) Disconnect() (err error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if !d.connected {
		return
	}

	// send the off signal; will block until the
	// offSignal is processed on the output thread
	d.offSignal <- true
	err = d.reader.Close() // best-effort
	close(d.output)
	d.connected = false
	return
}

func (d *AvatarDevice) Out() <-chan *DataFrame {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.output
}

func (d *AvatarDevice) Record(file string) (err error) {
	return
}

func (d *AvatarDevice) Stop() {

}
