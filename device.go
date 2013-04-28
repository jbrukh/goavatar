package goavatar

import (
	"fmt"
	"sync"
)

// ----------------------------------------------------------------- //
// Device
// ----------------------------------------------------------------- //

// Device represents an AvatarEEG device (or a mock device).
type Device interface {

	// Name of the device.
	Name() string

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

	// Recording returns true if and only if the device is currently
	// recording.
	Recording() bool
}

// ConnectFunc performs the low-level operation to connect
// to the device
type ConnectFunc func() error

// DisconnectFunc perfoms the low-level operation to disconnect
// from the device
type DisconnectFunc func() error

// StreamFunc performs the operation of reading the stream and
// writing data frames to the output channel, while also listening
// for control codes that tell it to terminate or record.
type StreamFunc func(<-chan ControlCode, chan<- *DataFrame)

// ControlCode is used for interacting with the parser of the stream,
// which is operating on a separate thread through the control channel.
type ControlCode int

const (
	Terminate   ControlCode = iota // Terminate streaming
	RecordStart                    // Start recording
	RecordStop                     // Stop recording
)

// baseDevice provides the basic framework for devices, including
// the skeleton implementation that keeps track of connection and
// recording state and thread-safety. However, the baseDevice provides
// no logic for streaming data and expects this functionality to
// be parameterized.
//
// In particular, implementors should respect the control channel
// and should send output data on the output channel.
type baseDevice struct {
	name      string
	control   chan ControlCode
	out       chan *DataFrame
	lock      sync.Mutex
	connected bool
	recording bool

	// low-level ops
	connFunc    ConnectFunc
	disconnFunc DisconnectFunc
	streamFunc  StreamFunc
}

// Create a new base device that performs connectivity
// and streaming based on the given function.
func newBaseDevice(name string, connFunc ConnectFunc, disconnFunc DisconnectFunc, streamFunc StreamFunc) *baseDevice {
	return &baseDevice{
		name:        name,
		control:     make(chan ControlCode),
		connFunc:    connFunc,
		disconnFunc: disconnFunc,
		streamFunc:  streamFunc,
	}
}

func (d *baseDevice) Name() string {
	return d.name
}

func (d *baseDevice) Connect() (out <-chan *DataFrame, err error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	// check connection
	if d.connected {
		return nil, fmt.Errorf("already connected to the device")
	}

	// perform connect
	if err = d.connFunc(); err != nil {
		return nil, fmt.Errorf("could not connect to the device: %v", err)
	}

	// create the output channel
	d.out = make(chan *DataFrame, DataBufferSize)

	// begin to stream
	go func() {
		d.streamFunc(d.control, d.out)
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
	// control code is processed on the output thread
	d.control <- Terminate
	close(d.out)

	// disconnect
	err = d.disconnFunc()
	d.connected = false

	return err
}

func (d *baseDevice) Connected() bool {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.connected
}

func (d *baseDevice) Out() <-chan *DataFrame {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.out
}

func (d *baseDevice) Record(file string) (err error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	// TODO: set the file in the device
	d.control <- RecordStart
	d.recording = true
	return
}

func (d *baseDevice) Stop() {
	d.lock.Lock()
	defer d.lock.Unlock()

	// TODO: set the file in the device
	d.control <- RecordStop
	d.recording = false
	return
}

func (d *baseDevice) Recording() bool {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.recording
}
