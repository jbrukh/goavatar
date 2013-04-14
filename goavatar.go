package goavatar

import (
	"io"
	"log"
	"os"
	"time"
)

// ----------------------------------------------------------------- //
// Constants
// ----------------------------------------------------------------- //

// constants used for parsing the
// AvatarEEG data stream
const (
	AvatarSyncByte          = 0xAA
	AvatarExpectedVersion   = 3 // version
	AvatarExpectedFrameType = 1 // data frame
	AvatarFracSecs          = time.Duration(4096)
	AvatarDataPointBytes    = 3
	AvatarSanePayload       = 8 * 32 * AvatarDataPointBytes
	DataBufferSize          = 1000
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

// Device represents an AvatarEEG device on a particular port that you
// can connect and disconnect from.
type Device struct {
	connected  bool
	serialPort string
	offSignal  chan bool
	reader     io.ReadCloser
	output     chan *DataFrame
}

// NewDevice creates a new Device. The user can then start
// streaming data by calling Connect().
func NewDevice(serialPort string) *Device {
	return &Device{
		connected:  false,
		serialPort: serialPort,
		offSignal:  make(chan bool, 1),
		output:     make(chan *DataFrame, DataBufferSize),
	}
}

// Connect to the device.
func (d *Device) Connect() (err error, output <-chan *DataFrame) {
	if d.connected {
		log.Printf("Tried to connect to the device, but it is already connected. (ignoring)")
		return
	}
	reader, err := d.connect()
	if err != nil {
		log.Printf("Connection to the device has failed: %s", err)
		return err, nil
	}
	d.reader = reader
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Got an error in the parser thread: %v\n", r)
				d.cleanup() // TODO: this is not threadsafe
			}
		}()
		parseByteStream(d.reader, d.offSignal, d.output)
	}()
	return nil, d.output
}

// Disconnect from the device.
func (d *Device) Disconnect() {
	if d.connected {
		d.offSignal <- true // send the off signal
		d.cleanup()
		d.connected = false
		log.Printf("Disconnected.")
	} else {
		log.Printf("Already disconnected. (ignoring)")
	}
}

// connect will connect to the serial port and set internal
// state of the Device appropriately. This method probably
// needs to be synchronized externally.
func (d *Device) connect() (device io.ReadCloser, err error) {
	device, err = os.Open(d.serialPort)
	if err != nil {
		return nil, err
	}
	d.connected = true
	log.Printf("Connected to the device on port %s", d.serialPort)
	return
}

func (d *Device) cleanup() {
	if err := d.reader.Close(); err != nil {
		log.Printf("Error closing the reader: %v", err)
	}

	// close the output channel
	close(d.output)

	// now disconnected
	d.connected = false
}

// parseByteStream parses the byte stream coming out of the device and writes the output
// to the output channel parameter. It also listens on the offSignal channel for any
// data, in which case it will stop listening the device and return.
func parseByteStream(r io.ReadCloser, offSignal <-chan bool, output chan<- *DataFrame) {
	defer r.Close()
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

		err = reader.ConsumePayload(header)
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
			crc:             crc,
		}
		ourCrc := reader.Crc()
		log.Printf("Frame: %+v, Crc: %v", *frame, ourCrc)
		if ourCrc != crc {
			log.Printf("Skipping this frame (bad crc)...")
			continue
		}
		// output <- frame
	}
	log.Printf("Closing parser...")
}

func shouldBreak(offSignal <-chan bool) bool {
	select {
	case <-offSignal:
		log.Printf("Received off signal...")
		return true
	default:
	}
	return false
}
