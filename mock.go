package goavatar

import (
	"log"
	"time"
)

// ----------------------------------------------------------------- //
// Mock Avatar Device
// ----------------------------------------------------------------- //

type MockDevice struct {
	baseDevice
}

// NewDevice creates a new Device. The user can then start
// streaming data by calling Connect() and reading the 
// output channel.
func NewMockDevice() *MockDevice {

	// CONNECT
	connFunc := func() error {
		// simulate startup time
		time.Sleep(time.Second * 1)
		return nil
	}

	// DISCONNECT
	disconnFunc := func() error {
		return nil // do nothing
	}

	// STREAM
	streamFunc := func(control <-chan ControlCode, output chan<- *DataFrame) {
		tick := 0
		for {
			select {
			case cc := <-control:
				if cc == Terminate {
					break
				} else if cc == RecordStart {
					log.Printf("REC: START")
				} else if cc == RecordStop {
					log.Printf("REC: STOP")
				}
				// ignore weird control codes
			default:
				// continue streaming
			}
			output <- frames[tick%len(frames)]
			tick++
			time.Sleep(time.Millisecond * 64) // 15.625 fps == 1 frame every 64 milliseconds
		}
	}

	return &MockDevice{
		baseDevice: *newBaseDevice(
			"MockAvatarEEG",
			connFunc,
			disconnFunc,
			streamFunc,
		),
	}
}
