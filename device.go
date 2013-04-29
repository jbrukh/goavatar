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
//     device.Disconnect() or close the output channel.
// (2) It shalt obey the ControlCode channel: upon receiving
//     a Terminate code, it shall exit with no errors.
// (3) Upon any error, it shall return that error.
//
type StreamFunc func(<-chan ControlCode, chan<- *DataFrame) error

// RecorderFunc produces a recorder for the given file
type RecorderFunc func(file string) Recorder

// ControlCode is used for interacting with the parser of the stream,
// which is operating on a separate thread through the control channel.
type ControlCode int

const (
	Terminate ControlCode = iota // Terminate streaming
	Terminated
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
	publicOut chan *DataFrame
	lock      sync.Mutex
	connected bool
	recording bool
	recorder  Recorder
	wg        sync.WaitGroup

	// low-level ops
	connFunc     ConnectFunc
	disconnFunc  DisconnectFunc
	streamFunc   StreamFunc
	recorderFunc RecorderFunc
}

// Create a new base device that performs connectivity
// and streaming based on the given function.
func newBaseDevice(name string, connFunc ConnectFunc, disconnFunc DisconnectFunc,
	streamFunc StreamFunc, recorderFunc RecorderFunc) *baseDevice {
	return &baseDevice{
		name:         name,
		control:      make(chan ControlCode),
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

	// create the internal output channel
	d.out = make(chan *DataFrame, DataBufferSize)
	d.publicOut = make(chan *DataFrame, DataBufferSize)

	// run the interceptor worker
	go func() {
		d.wg.Add(1)
		d.interceptOut()
		d.wg.Done()
	}()

	// begin to stream
	go func() {
		d.wg.Add(1)
		// run the streamer and listen for errors
		if err := d.streamFunc(d.control, d.out); err != nil {
			log.Printf("error in streamer: %v", err)
		}
		d.wg.Done()

		// on error or exit, we will disconnect the device
		if err := d.Disconnect(); err != nil {
			log.Printf("error on disconnect: $v", err)
		}

	}()

	// mark connected
	d.connected = true
	return nil
}

func (d *baseDevice) Disconnect() (err error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	// check for idempotency
	if !d.connected {
		return
	}

	log.Printf("disconnecting the device")

	// send the Terminate signal; will block until the
	// control code is processed on the output thread
	d.control <- Terminate

	// close the internal output channel, as the streamer
	// is now guaranteed to exit, not pushing to d.out
	// any more
	close(d.out)

	// interceptor will react to the d.out channel closing
	// by itself exiting; we wait until that happens...
	d.wg.Wait()

	// then we close the publicOut channel so that consumers
	// are aware that we are disconnecting
	close(d.publicOut)

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
	return d.publicOut
}

func (d *baseDevice) Record(file string) (err error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	if d.recording {
		return fmt.Errorf("already recording")
	}

	// TODO: set the file in the device
	if d.recorder = d.recorderFunc(file); d.recorder == nil {
		return fmt.Errorf("no recorder was provided")
	}

	if err := d.recorder.Start(); err != nil {
		return fmt.Errorf("could not start the recorder: %v", err)
	}

	d.recording = true
	return
}

func (d *baseDevice) Stop() {
	d.lock.Lock()
	defer d.lock.Unlock()

	if !d.recording {
		return
	}

	if err := d.recorder.Stop(); err != nil {
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

func (d *baseDevice) interceptOut() {
	for {
		df, ok := <-d.out
		if !ok {
			return
		}

		d.lock.Lock()
		if d.recording {
			// TODO: make async?
			d.recorder.ProcessFrame(df)
		}
		d.lock.Unlock()

		// otherwise, pump data into publicOut
		d.publicOut <- df
	}
}
