package goavatar

import (
	"fmt"
	//"log"
	//"math/rand"
	"sync"
	"time"
)

// ----------------------------------------------------------------- //
// Mock Avatar Device
// ----------------------------------------------------------------- //

type MockDevice struct {
	offSignal chan bool       // send a value to disconnect the device
	output    chan *DataFrame // output channel
	connected bool
	lock      sync.Mutex
}

// NewDevice creates a new Device. The user can then start
// streaming data by calling Connect() and reading the 
// output channel.
func NewMockDevice() *MockDevice {
	return &MockDevice{
		offSignal: make(chan bool),
		connected: false,
	}
}

func (d *MockDevice) Connected() bool {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.connected
}

func (d *MockDevice) Connect() (output <-chan *DataFrame, err error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	// already connected?
	if d.connected {
		return nil, fmt.Errorf("Device is already connected.")
	}
	d.output = make(chan *DataFrame, DataBufferSize)

	// simulate startup time
	time.Sleep(time.Second * 1)

	go func() {
		mockConnection(d.offSignal, d.output)
	}()
	d.connected = true
	return d.output, nil
}

// Disconnect from the device.
func (d *MockDevice) Disconnect() {
	d.lock.Lock()
	defer d.lock.Unlock()

	if !d.connected {
		return
	}

	// send the off signal; will block until the
	// offSignal is processed on the output thread
	d.offSignal <- true
	close(d.output)
	d.connected = false
}

func (d *MockDevice) Out() <-chan *DataFrame {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.output
}

func mockConnection(offSignal <-chan bool, output chan<- *DataFrame) {
	for {
		// break the loop if 
		// there is an off signal
		if shouldBreak(offSignal) {
			break
		}
		output <- mockFrame()
		time.Sleep(time.Millisecond * 64) // 15.625 fps == 1 frame every 64 milliseconds
	}
}

var tick int = 0

func mockFrame() (frame *DataFrame) {
	inx := tick
	tick++
	return frames[inx%len(frames)]
}

func (d *MockDevice) Record(file string) (err error) {
	return
}

func (d *MockDevice) Stop() {

}
