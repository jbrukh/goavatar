package goavatar

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"sync"
	"time"
)

// ----------------------------------------------------------------- //
// Constants
// ----------------------------------------------------------------- //

const (
	DataBufferSize = 1024
)

// parseByteStream parses the byte stream coming out of the device and writes the output
// to the output channel parameter. It also listens on the offSignal channel for any
// data, in which case it will stop listening the device and return.
func parseByteStream(r io.ReadCloser, offSignal <-chan bool, output chan<- *DataFrame) {
	reader := newAvatarParser(r)

	for {
		// break the loop if 
		// there is an off signal
		if shouldBreak(offSignal) {
			break
		}

		// read the frame
		err := reader.ConsumeSync()
		if err != nil {
			log.Printf("Error: %v", err)
			break // since the underlying reader must be hosed
		}

		header, err := reader.ConsumeHeader()
		if err != nil {
			log.Printf("Error: %v", err)
			continue // will break on next loop if reader hosed
		}

		data, err := reader.ConsumePayload(header)
		if err != nil {
			log.Printf("Error: %v", err)
			continue // will break on next loop if reader hosed
		}

		crc, err := reader.ConsumeCrc()
		if err != nil {
			log.Printf("Error: %v", err)
			continue // will break on next loop if reader hosed
		}

		// collect the frame
		frame := &DataFrame{
			DataFrameHeader: *header,
			data:            data,
			crc:             crc,
		}
		ourCrc := reader.Crc()
		if ourCrc != crc {
			log.Printf("Bad CRC: %+v (expected: %d)", *frame, ourCrc)
			continue
		}

		output <- frame
	}
}

func shouldBreak(offSignal <-chan bool) bool {
	select {
	case <-offSignal:
		return true
	default:
	}
	return false
}

// ----------------------------------------------------------------- //
// Mock Avatar Device
// ----------------------------------------------------------------- //

type MockDevice struct {
	offSignal chan bool       // send a value to disconnect the device
	output    chan *DataFrame // output channel
	connected bool
	lock      *sync.Mutex
}

// NewDevice creates a new Device. The user can then start
// streaming data by calling Connect() and reading the 
// output channel.
func NewMockDevice() *MockDevice {
	return &MockDevice{
		offSignal: make(chan bool),
		output:    make(chan *DataFrame, DataBufferSize),
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

	// close the output channel
	close(d.output)
	d.connected = false
}

func mockConnection(offSignal <-chan bool, output chan<- *DataFrame) {
	for {
		// break the loop if 
		// there is an off signal
		if shouldBreak(offSignal) {
			break
		}
		output <- mockFrame()
		time.Sleep(time.Millisecond * 100)
	}
}

func mockFrame() (frame *DataFrame) {
	// some fake data
	var data [9][]float64
	for i := 1; i <= 2; i++ {
		data[i] = make([]float64, 16)
		for j := 0; j < 16; j++ {
			data[i][j] = rand.Float64()*float64(0.02) + float64(i)
		}
	}

	// TODO: make timestamps realistic
	frame = &DataFrame{
		DataFrameHeader: DataFrameHeader{
			FieldSampleRateVersion: 3,
			FieldFrameSize:         118,
			FieldFrameType:         1,
			FieldFrameCount:        268,
			FieldChannels:          2,
			FieldSamples:           16,
			FieldVoltRange:         750,
			FieldTimestamp:         1345192284,
			FieldFracSecs:          2436,
		},
		data: data,
		crc:  uint16(0),
	}
	return
}
