package goavatar

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"time"
)

// ----------------------------------------------------------------- //
// Constants
// ----------------------------------------------------------------- //

const (
	DataBufferSize = 1024
)

type AvatarChannel int

// AvatarEEG channels
const (
	AvatarChannelTrigger AvatarChannel = iota
	AvatarChannel1
	AvatarChannel2
	AvatarChannel3
	AvatarChannel4
	AvatarChannel5
	AvatarChannel6
	AvatarChannel9
	AvatarChannel8
)

// ----------------------------------------------------------------- //
// AvatarEEG Device
// ----------------------------------------------------------------- //

// Device represents an AvatarEEG device on a particular port.
type Device interface {
	Connect() (<-chan *DataFrame, error)
	Disconnect()
	Out() <-chan *DataFrame
}

type AvatarDevice struct {
	serialPort string          // serial port like /dev/tty.AvatarEEG03009-SPPDev
	offSignal  chan bool       // send a value to disconnect the device
	reader     io.ReadCloser   // the reader of the serial port
	output     chan *DataFrame // channel that delivers raw Avatar output
}

// NewDevice creates a new Device. The user can then start
// streaming data by calling Connect() and reading the 
// output channel.
func NewAvatarDevice(serialPort string) *AvatarDevice {
	return &AvatarDevice{
		serialPort: serialPort,
		offSignal:  make(chan bool),
		output:     make(chan *DataFrame, DataBufferSize),
	}
}

// Connect to the device.
func (d *AvatarDevice) Connect() (output <-chan *DataFrame, err error) {
	// connect to the reader for the port; this will
	// fail if we are already reading from this port
	reader, err := os.Open(d.serialPort)
	if err != nil {
		return nil, fmt.Errorf("Cannot connect: %v", err)
	}

	// remember the reader and begin streaming data
	// on a separate thread
	d.reader = reader
	go func() {
		parseByteStream(d.reader, d.offSignal, d.output)
	}()
	return d.output, nil
}

// Disconnect from the device.
func (d *AvatarDevice) Disconnect() {
	// send the off signal; will block until the
	// offSignal is processed on the output thread
	d.offSignal <- true

	// close the reader
	if err := d.reader.Close(); err != nil {
		log.Printf("Error closing the reader: %v", err)
	}

	// close the output channel
	close(d.output)
}

func (d *AvatarDevice) Out() <-chan *DataFrame {
	return d.output
}

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
}

// NewDevice creates a new Device. The user can then start
// streaming data by calling Connect() and reading the 
// output channel.
func NewMockDevice() *MockDevice {
	return &MockDevice{
		offSignal: make(chan bool),
		output:    make(chan *DataFrame, DataBufferSize),
	}
}

func (d *MockDevice) Connect() (output <-chan *DataFrame, err error) {
	// simulate startup time
	time.Sleep(time.Second * 1)

	go func() {
		mockConnection(d.offSignal, d.output)
	}()
	return d.output, nil

}

// Disconnect from the device.
func (d *MockDevice) Disconnect() {
	// send the off signal; will block until the
	// offSignal is processed on the output thread
	d.offSignal <- true

	// close the output channel
	close(d.output)
}

func (d *MockDevice) Out() <-chan *DataFrame {
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
		time.Sleep(time.Millisecond * 100)
	}
}

func mockFrame() (frame *DataFrame) {
	var data [9][]float64
	for i := 1; i <= 2; i++ {
		data[i] = make([]float64, 16)
		for j := 0; j < 16; j++ {
			data[i][j] = rand.Float64()*float64(0.02) + float64(i)
		}
	}
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
