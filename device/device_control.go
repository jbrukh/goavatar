//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package device

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
func (c *Control) ShouldTerminate() bool {
	select {
	case <-c.done:
		return true
	default:
	}
	return false
}

// The client worker should send data frames to the
// Device by calling this method.
func (c *Control) Send(df DataFrame) {
	c.out <- df
}

// The client must send DeviceInfo before sending
// data.
func (c *Control) SendInfo(info *DeviceInfo) {
	c.info <- info
}

// The client worker should call this method before
// exiting.
func (c *Control) Close() {
	close(c.out)
}
