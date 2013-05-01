package goavatar

import (
	"fmt"
	"log"
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
	Connect() error

	// Connected returns true if and only if the device is
	// currently connected.
	Connected() bool

	// Disconnects from the device, closes the output channel,
	// and cleans relevant resources. Calls to disconnect are
	// idempotent.
	Disconnect() error

	// Returns the output channel for the device. 
	Out() <-chan *DataFrame

	// Starts recording the streaming data to a file.
	Record() (err error)

	// Stops recording the streaming data.
	Stop() (outFile string, err error)

	// Recording returns true if and only if the device is currently
	// recording.
	Recording() bool
}

// ConnectFunc performs the low-level operation to connect
// to the device. This usually means opening the port of the
// device for reading.
type ConnectFunc func() error

// DisconnectFunc perfoms the low-level operation to disconnect
// from the device. This usually means closing the port of the
// device.
type DisconnectFunc func() error

// StreamFunc performs the operation of reading the stream and
// writing data frames to the output channel. This function is
// expected to obey the following contract:
//
// (1) It shalt not perform any resource cleanup, this is the
//     job of the DisconnectFunc. It shalt not call 
//     device.Disconnect().
// (2) It shalt obey c.ShouldTerminate() and exit without error.
// (3) Upon any error, it shall return that error.
//
type StreamFunc func(c *Control) error

// RecorderProvider produces a recorder for the given file
type RecorderProvider func() Recorder

type Control struct {
	done chan bool
	out  chan *DataFrame
	d    *baseDevice
}

func newControl(d *baseDevice) *Control {
	return &Control{
		done: make(chan bool),
		out:  make(chan *DataFrame, DataBufferSize),
		d:    d,
	}
}

func (control *Control) ShouldTerminate() bool {
	select {
	case <-control.done:
		return true
	default:
	}
	return false
}

func (control *Control) Send(df *DataFrame) {
	control.out <- df
	if !control.ShouldTerminate() {
		if control.d.Recording() {
			control.d.recorder.ProcessFrame(df)
		}
	}
}

func (control *Control) Close() {
	close(control.out)
}

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
	lock      sync.Mutex
	connected bool
	recording bool
	recorder  Recorder
	control   *Control

	// low-level ops
	connFunc     ConnectFunc
	disconnFunc  DisconnectFunc
	streamFunc   StreamFunc
	recorderFunc RecorderProvider
}

// Create a new base device that performs connectivity
// and streaming based on the given function.
func newBaseDevice(name string, connFunc ConnectFunc, disconnFunc DisconnectFunc,
	streamFunc StreamFunc, recorderFunc RecorderProvider) *baseDevice {
	return &baseDevice{
		name:         name,
		connFunc:     connFunc,
		disconnFunc:  disconnFunc,
		streamFunc:   streamFunc,
		recorderFunc: recorderFunc,
	}
}

func (d *baseDevice) Name() string {
	return d.name
}

func (d *baseDevice) Connect() (err error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	// check connection
	if d.connected {
		return fmt.Errorf("already connected to the device")
	}

	// perform connect
	if err = d.connFunc(); err != nil {
		return fmt.Errorf("could not connect to the device: %v", err)
	}

	// create the controller
	d.control = newControl(d)

	// begin to stream
	go func() {
		// run the streamer and listen for errors
		if err := d.streamFunc(d.control); err != nil {
			log.Printf("error in streamer: %v", err)
		}

		// on error or exit, we will disconnect the device;
		// since we know the streamer has exited we will
		// not send the done signal
		if err := d.disconnect(true); err != nil {
			log.Printf("error on disconnect: $v", err)
		}

	}()

	// mark connected
	d.connected = true
	return nil
}

func (d *baseDevice) Disconnect() (err error) {
	return d.disconnect(false)
}

func (d *baseDevice) disconnect(ignoreDone bool) (err error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	// check for idempotency
	if !d.connected {
		return
	}

	// when we know the streamer goroutine has
	// exited, we should skip this step
	if !ignoreDone {
		d.control.done <- true
	}

	// if we are in the process of recording, we
	// should stop
	if d.recording {
		d.recorder.Stop()
		d.recording = false
	}

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
	return d.control.out
}

func (d *baseDevice) Record() (err error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.recording {
		return fmt.Errorf("already recording")
	}

	if !d.connected {
		return fmt.Errorf("device is not connected")
	}

	if d.recorder = d.recorderFunc(); d.recorder == nil {
		return fmt.Errorf("no recorder was provided")
	}

	if err := d.recorder.Start(); err != nil {
		return fmt.Errorf("could not start the recorder: %v", err)
	}

	d.recording = true
	return
}

func (d *baseDevice) Stop() (outFile string, err error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if !d.recording {
		return
	}

	if outFile, err = d.recorder.Stop(); err != nil {
		log.Printf("could not shut down the recorder: %v", err)
	}
	d.recorder = nil

	d.recording = false
	return
}

func (d *baseDevice) Recording() bool {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.recording
}
