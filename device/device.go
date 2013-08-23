//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package device

import (
	"fmt"
	. "github.com/jbrukh/goavatar/datastruct"
	. "github.com/jbrukh/goavatar/repo"
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
	Repo() *Repository

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

	// Subscribe to device data.
	Subscribe(string) (chan DataFrame, error)

	// Unsubscribe from device data.
	Unsubscribe(string)
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

	// The name of the device.
	Name() string

	// The directory where recordings are stored.
	Repo() *Repository
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
	engaged    bool
	control    *Control
	deviceImpl DeviceImpl
	info       *DeviceInfo
	ps         *PubSub
}

// Create a new device based on some given
// device implementation.
func NewDevice(deviceImpl DeviceImpl) Device {
	return &BaseDevice{
		deviceImpl: deviceImpl,
		ps:         NewPubSub(),
	}
}

// The name of the device.
func (d *BaseDevice) Name() string {
	return d.deviceImpl.Name()
}

// The recording repository directory for
// this device.
func (d *BaseDevice) Repo() *Repository {
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

	d.ps.UnsubscribeAll()

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

func (d *BaseDevice) Subscribe(name string) (chan DataFrame, error) {
	return d.ps.Subscribe(name)
}

func (d *BaseDevice) Unsubscribe(name string) {
	d.ps.Unsubscribe(name)
}

func (d *BaseDevice) publish(df DataFrame) {
	d.ps.publish(df)
}
