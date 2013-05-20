//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package goavatar

import (
	"fmt"
	"log"
	"sync"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

// ----------------------------------------------------------------- //
// Constants
// ----------------------------------------------------------------- //

const DataFrameBufferSize = 1024

// ----------------------------------------------------------------- //
// Device -- interface for devices
// ----------------------------------------------------------------- //

// Device represents an AvatarEEG device (or a mock device).
type Device interface {

	// Name of the device.
	Name() string

	// Return the path of the directory where recorder files are
	// stored.
	Repo() string

	// Obtain the device information
	Info() *DeviceInfo

	// Engage to the device and return the output channel.
	// Engageing to a device that is already engaged is
	// an error.
	Engage() error

	// Disengages from the device, closes the output channel,
	// and cleans relevant resources. Calls to disengage are
	// idempotent.
	Disengage() error

	// Engaged returns true if and only if the device is
	// currently engaged.
	Engaged() bool

	// Returns the output channel for the device.
	Out() <-chan DataFrame

	// Starts recording the streaming data to a file.
	Record() (err error)

	// Stops recording the streaming data.
	Stop() (outFile string, err error)

	// Recording returns true if and only if the device is currently
	// recording.
	Recording() bool
}

// ----------------------------------------------------------------- //
// Subscriptions
// ----------------------------------------------------------------- //

// Device implementation interface.
type DeviceImpl interface {
	// Performs the low-level operation to engage
	// to the device. This usually means opening the port of the
	// device for reading.
	Engage() error

	// Perfoms the low-level operation to disengage
	// from the device. This usually means closing the port of the
	// device.
	Disengage() error

	// Performs the operation of reading the stream and
	// writing data frames to the output channel. This function is
	// expected to obey the following contract with the Control:
	//
	// (1) The first possible call shalt be to SendInfo(), or else
	//     the device Engage() function will wait indefinitely.
	// (2) It shalt not perform any resource cleanup, this is the
	//     job of Disengage(). It shalt NOT try to disengage the device.
	// (3) It shalt obey c.ShouldTerminate() and exit without error.
	// (4) Upon any error, it shalt return that error.
	//
	// Note returning DeviceInfo in this way is a hardware limitation.
	Stream(*Control) error

	// Produces a recorder. This recorder will record a single recording
	// to a single file, or fail, and be destroyed.
	ProvideRecorder() Recorder

	// The name of the device.
	Name() string

	// The directory where recordings are stored.
	Repo() string
}

// ----------------------------------------------------------------- //
// Device Control -- used by implementation providers to report
// data and know when to disengage
// ----------------------------------------------------------------- //

// Control is a control structure used by client workers
// that stream data.
type Control struct {
	done chan bool
	out  chan DataFrame
	info chan *DeviceInfo
	d    *BaseDevice
}

// Create a new Control.
func newControl(d *BaseDevice) *Control {
	return &Control{
		done: make(chan bool),
		out:  make(chan DataFrame, DataFrameBufferSize),
		info: make(chan *DeviceInfo, 1),
		d:    d,
	}
}

// ShouldTerminate returns true if and only if the
// Device is calling for streaming operations to stop.
func (control *Control) ShouldTerminate() bool {
	select {
	case <-control.done:
		return true
	default:
	}
	return false
}

// The client worker should send data frames to the
// Device by calling this method.
func (control *Control) Send(df DataFrame) {
	control.out <- df
	if control.d.Recording() {
		control.d.recorder.ProcessFrame(df)
	}
}

// The client must send DeviceInfo before sending
// data.
func (control *Control) SendInfo(info *DeviceInfo) {
	control.info <- info
}

// The client worker should call this method before
// exiting.
func (control *Control) Close() {
	close(control.out)
}

// ----------------------------------------------------------------- //
// Device Info -- basic info about the device that should
// be ascertained on every connect.
// ----------------------------------------------------------------- //

type DeviceInfo struct {
	Channels   int // how many channels are streaming
	SampleRate int // what is the sample rate of the device
}

// ----------------------------------------------------------------- //
// Base Device -- skeleton implementation for Octopus devices
// ----------------------------------------------------------------- //

// BaseDevice provides the basic framework for devices, including
// the skeleton implementation that keeps track of engageion and
// recording state and thread-safety. However, the BaseDevice provides
// no logic for streaming data and expects this functionality to
// be parameterized.
//
// In particular, implementors should respect the Control object
// they are passed. See the contract of Stream() function above.
type BaseDevice struct {
	sync.Mutex
	rlock      sync.Mutex
	engaged    bool
	recording  bool
	recorder   Recorder
	control    *Control
	deviceImpl DeviceImpl
	info       *DeviceInfo
}

// Create a new device based on some given
// device implementation.
func NewDevice(deviceImpl DeviceImpl) Device {
	return &BaseDevice{
		deviceImpl: deviceImpl,
	}
}

// The name of the device.
func (d *BaseDevice) Name() string {
	return d.deviceImpl.Name()
}

// The recording repository directory for
// this device.
func (d *BaseDevice) Repo() string {
	return d.deviceImpl.Repo()
}

func (d *BaseDevice) Info() *DeviceInfo {
	d.Lock()
	defer d.Unlock()
	return d.info
}

func (d *BaseDevice) Engage() (err error) {
	d.Lock()
	defer d.Unlock()

	// check engageion
	if d.engaged {
		return fmt.Errorf("already engaged to the device")
	}

	log.Printf("%s: CONNECT", d.Name())

	// perform engage
	if err = d.deviceImpl.Engage(); err != nil {
		return fmt.Errorf("could not engage to the device: %v", err)
	}

	// create the controller
	d.control = newControl(d)

	// begin to stream
	go func() {
		// run the streamer and listen for errors
		if err := d.deviceImpl.Stream(d.control); err != nil {
			log.Printf("error in streamer: %v", err)
		}

		// on error or exit, we will disengage the device;
		// since we know the streamer has exited we will
		// not send the done signal
		if err := d.disengage(true); err != nil {
			log.Printf("error on disengage: %v", err)
		}

	}()

	// listen for info
	info := <-d.control.info

	d.info = info
	log.Printf("%s: DEVICE INFO %+v", d.Name(), info)

	// mark engaged
	d.engaged = true
	return nil
}

func (d *BaseDevice) Disengage() (err error) {
	return d.disengage(false)
}

func (d *BaseDevice) disengage(ignoreDone bool) (err error) {
	d.Lock()
	defer d.Unlock()

	// check for idempotency
	if !d.engaged {
		return
	}

	log.Printf("%s: DISCONNECT", d.Name())

	// when we know the streamer goroutine has
	// exited, we should skip this step
	if !ignoreDone {
		d.control.done <- true
	}

	d.rlock.Lock()
	// if we are in the process of recording, we
	// should stop
	if d.recording {
		d.stop()
	}
	d.rlock.Unlock()

	// disengage
	err = d.deviceImpl.Disengage()
	d.engaged = false
	return err
}

func (d *BaseDevice) Engaged() bool {
	d.Lock()
	defer d.Unlock()
	return d.engaged
}

func (d *BaseDevice) Out() <-chan DataFrame {
	d.Lock()
	defer d.Unlock()
	return d.control.out
}

func (d *BaseDevice) Record() (err error) {
	d.rlock.Lock()
	defer d.rlock.Unlock()
	return d.record()
}

// record is the unsynchronized version of Record,
// used internally.
func (d *BaseDevice) record() (err error) {
	if d.recording {
		return fmt.Errorf("already recording")
	}

	if !d.engaged {
		return fmt.Errorf("device is not engaged")
	}

	log.Printf("%s: RECORD", d.Name())

	if d.recorder = d.deviceImpl.ProvideRecorder(); d.recorder == nil {
		return fmt.Errorf("no recorder was provided")
	}

	if err := d.recorder.Start(); err != nil {
		return fmt.Errorf("could not start the recorder: %v", err)
	}

	d.recording = true
	return
}

func (d *BaseDevice) Stop() (outFile string, err error) {
	d.rlock.Lock()
	defer d.rlock.Unlock()
	return d.stop()
}

func (d *BaseDevice) stop() (outFile string, err error) {
	if !d.recording {
		return
	}

	log.Printf("%s: STOP RECORDING", d.Name())

	if outFile, err = d.recorder.Stop(); err != nil {
		log.Printf("could not shut down the recorder: %v", err)
	}
	d.recorder = nil
	d.recording = false
	return
}

func (d *BaseDevice) Recording() bool {
	d.rlock.Lock()
	defer d.rlock.Unlock()
	return d.recording
}
