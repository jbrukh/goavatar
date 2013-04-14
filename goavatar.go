package goavatar

import (
	"bufio"
	"encoding/binary"
	"fmt"
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

// ----------------------------------------------------------------- //
// AvatarEEG Data Frame and Parsing
// ----------------------------------------------------------------- //

type DataFrameHeader struct {
	FieldSampleRateVersion byte
	FieldFrameSize         uint16
	FieldFrameType         byte
	FieldFrameCount        uint32
	FieldChannels          byte
	FieldSamples           uint16
	FieldVoltRange         uint16
	FieldTimestamp         uint32
	FieldFracSecs          uint16
}

// DataFrame represents the raw data that is transmitted from the AvatarEEG
// device. 
type DataFrame struct {
	DataFrameHeader
	channelData [9][]uint32 // raw ADC data for the channels
	crc         uint16      // CRC-16-CCIT calculated on the entire frame not including CRC
}

// SampleRate
func (h *DataFrameHeader) SampleRate() (sampleRate int, err error) {
	sr := (h.FieldSampleRateVersion >> 6)
	if sr == 0x00 {
		return 250, nil
	} else if sr == 0x01 {
		return 500, nil
	} else if sr == 0x02 {
		return 1000, nil
	}
	return 0, fmt.Errorf("Unknown sample rate")
}

// Version
func (h *DataFrameHeader) Version() int {
	return int(h.FieldSampleRateVersion & 0x3F)
}

// FrameSize
func (h *DataFrameHeader) FrameSize() int {
	return int(h.FieldFrameSize)
}

// FrameType
func (h *DataFrameHeader) FrameType() int {
	return int(h.FieldFrameType)
}

// FrameCount
func (h *DataFrameHeader) FrameCount() int {
	return int(h.FieldFrameCount)
}

// Channels
func (h *DataFrameHeader) Channels() int {
	return int(h.FieldChannels)
}

// Samples
func (h *DataFrameHeader) Samples() int {
	return int(h.FieldSamples)
}

// Range returns the range, in mVpp, of each data channel which is dependent on the
// gain and is 12 by default. This is needed to convert the raw counting data from
// the analog-to-digital converter. To convert counts to voltage, simply perform:
//
//     (value) * range / 1000 / 2^24
//
func (h *DataFrameHeader) VoltRange() int {
	return int(h.FieldVoltRange)
}

// Time converts the timestamp data into Unix nanosecond time.
func (h *DataFrameHeader) Time() time.Time {
	return time.Unix(int64(h.FieldTimestamp), int64(time.Duration(h.FieldFracSecs)*time.Second/AvatarFracSecs))
}

// String
func (df *DataFrame) String() string {
	return fmt.Sprintf("\n%+v\n", df)
}

// ----------------------------------------------------------------- //
// AvatarEEG stream parser
// ----------------------------------------------------------------- //

// a parser/crc calculator for the stream
type avatarParser struct {
	reader *bufio.Reader // reader of the stream
	crc    CrcWriter
}

// create a new crcReader
func newAvatarParser(reader io.ReadCloser) *avatarParser {
	return &avatarParser{
		reader: bufio.NewReader(reader),
	}
}

func (r *avatarParser) Reset() {
	// reset the buffer and header
	r.crc.Reset()
}

// resets the buffer and searches for 
// the next sync byte
func (r *avatarParser) ConsumeSync() (err error) {
	r.Reset()

	// sync up with the stream, reading up
	// until the sync up value
	_, err = r.reader.ReadBytes(AvatarSyncByte)
	if err != nil {
		return err
	}

	// note the start byte
	r.crc.WriteByte(AvatarSyncByte)
	return
}

func (r *avatarParser) ConsumeHeader() (h *DataFrameHeader, err error) {
	h = new(DataFrameHeader)
	// read the data into the header
	err = binary.Read(r.reader, binary.BigEndian, h)
	if err != nil {
		return nil, err
	}

	// note the header
	err = binary.Write(&r.crc, binary.BigEndian, h)
	return
}

func (r *avatarParser) ConsumePayload(header *DataFrameHeader) (err error) {
	// read the payload
	// now read the data
	pSize := header.Channels() * header.Samples() * AvatarDataPointBytes
	payload := make([]byte, pSize)
	n := 0

	// read until the whole payload is read
	for n != pSize {
		nRead, err := r.reader.Read(payload[n:])
		if err != nil {
			return err
		}
		n += nRead
	}

	// note the payload
	r.crc.Write(payload)
	return
}

// read the crc
func (r *avatarParser) ConsumeCrc() (crc uint16, err error) {
	// read the crc
	err = binary.Read(r.reader, binary.BigEndian, &crc)
	return
}

func (r *avatarParser) Crc() (crc uint16) {
	return r.crc.Crc()
}

// ----------------------------------------------------------------- //
// CRC Writer -- for calculating CRC-16-CCIT, according to Avatar Spec
// ----------------------------------------------------------------- //

type CrcWriter struct {
	crc uint16
}

func (w *CrcWriter) Crc() uint16 {
	return w.crc
}

func (w *CrcWriter) Reset() {
	w.crc = uint16(0)
}

func (w *CrcWriter) Write(p []byte) (n int, err error) {
	for _, b := range p {
		w.WriteByte(b)
	}
	return len(p), nil
}

func (w *CrcWriter) WriteByte(b byte) {
	w.crc = (w.crc >> 8) | ((w.crc & 0xFF) << 8)
	w.crc ^= uint16(b)
	w.crc ^= (w.crc & 0xFF) >> 4
	w.crc ^= (w.crc << 12) & 0xFFFF
	w.crc ^= (w.crc & 0xFF) << 5
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
		//output <- frame
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
