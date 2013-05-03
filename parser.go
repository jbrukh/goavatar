package goavatar

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"time"
)

// These values are valid with the firmware version
// on the device that we have acquired. In the future
// they may change as the format is updated.
const (
	AvatarSyncByte          = 0xAA
	AvatarExpectedVersion   = 3                   // version
	AvatarExpectedFrameType = 1                   // data frame
	AvatarFracSecs          = time.Duration(4096) // fractional second parts
	AvatarMaxFramesSize     = 454                 //  22 + 3*9*16 = 454 (including trigger channel)
	AvatarAdcRange          = 16777216            // 2^24
	AvatarPointSize         = 3
	AvatarExpectedSamples   = 16
	AvatarSanePayload       = 8 * AvatarExpectedSamples * AvatarPointSize
	AvatarMaxChannels       = 8
	AvatarHeaderSize        = 19 // not including sync byte
)

// The possible sample rates that the Avatar
// currently supports.
var AvatarSampleRates = []int{250, 500, 1000}

// ----------------------------------------------------------------- //
// AvatarEEG Data Frame and Parsing
// ----------------------------------------------------------------- //

type DataFrameHeader struct {
	// a header is 1+2+1+4+1+2+2+4+2 = 19 bytes
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
	data     *SamplingBuffer // processed data, in a multibuffer
	received time.Time       // time this frame was received locally
	crc      uint16          // crc of the frame
}

// String
func (df *DataFrame) String() string {
	return fmt.Sprintf("\n%+v\n", *df)
}

// Return this dataframe as Go code.
func (df *DataFrame) AsCode() string {
	f := `&DataFrame{
		DataFrameHeader:DataFrameHeader{
			FieldSampleRateVersion:%#v, 
			FieldFrameSize:%#v, 
			FieldFrameType:%#v, 
			FieldFrameCount:%#v, 
			FieldChannels:%#v, 
			FieldSamples:%#v, 
			FieldVoltRange:%#v, 
			FieldTimestamp:%#v, 
			FieldFracSecs:%#v,
		},
		data:NewSamplingBufferFromSlice(%d, 1, %#v), 
		crc:%#v, 
		received:time.Unix(%v, %v),
	}`
	return fmt.Sprintf(f,
		df.FieldSampleRateVersion,
		df.FieldFrameSize,
		df.FieldFrameType,
		df.FieldFrameCount,
		df.FieldChannels,
		df.FieldSamples,
		df.FieldVoltRange,
		df.FieldTimestamp,
		df.FieldFracSecs,
		df.Buffer().Channels(),
		df.Buffer().data,
		df.crc,
		df.received.Unix(),
		df.received.Nanosecond(),
	)

}

func (df *DataFrame) Buffer() *SamplingBuffer {
	return df.data
}

func (df *DataFrame) ChannelData(channel int) []float64 {
	return df.data.ChannelData(channel)
}

// the time this data framed was received locally
func (df *DataFrame) Received() time.Time {
	return df.received
}

// SampleRate: the number of data samples delivered in one
// second (per channel)
func (h *DataFrameHeader) SampleRate() (sampleRate int) {
	sr := int(h.FieldSampleRateVersion >> 6)
	if sr < 0 || sr > 2 {
		return 0
	}
	return AvatarSampleRates[sr]
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

// HasTriggerChannel
func (h *DataFrameHeader) HasTriggerChannel() bool {
	return (h.FieldChannels >> 7) > 0
}

// Channels (number of, not including trigger)
func (h *DataFrameHeader) Channels() int {
	// zero the first bit for the trigger channel
	return int(h.FieldChannels & 0x7F)
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

// Payload size
func (h *DataFrameHeader) PayloadSize() int {
	return h.Channels() * h.Samples() * AvatarPointSize
}

// ----------------------------------------------------------------- //
// AvatarEEG stream parser
// ----------------------------------------------------------------- //

// a parser/crc calculator for the stream
type avatarParser struct {
	reader *bufio.Reader // reader of the stream
	crc    CrcWriter
}

// create a new parser
func NewAvatarParser(reader io.ReadCloser) *avatarParser {
	return &avatarParser{
		reader: bufio.NewReader(reader),
	}
}

// TODO: we know the size of the entire frame, so we can peek at it
// and check the CRC; also, the frame has a maximum size...
//
// "With the version of firmware you have there will always be 16 samples in
// a frame and if trigger channel enabled this would be 9 channels. So max
// frame size with your firmware is 22 + 3*9*16 = 454 bytes.
//
// With future versions we may adjust number of samples to optimize
// Bluetooth performance and may have hardware that supports up to 24
// channels."
func (r *avatarParser) ParseFrame() (dataFrame *DataFrame, err error) {
	// reset the crc calculation
	r.crc.Reset()

	log.Printf("reading sync")

	// sync up with the stream, reading up
	// until the sync up value
	_, err = r.reader.ReadBytes(AvatarSyncByte)
	if err != nil {
		return nil, err
	}

	timeReceived := time.Now()

	// note the sync byte
	r.crc.WriteByte(AvatarSyncByte)

	// at this point, we will peek ahead
	// just 3 bytes which is enough to read the frame size
	three, err := r.reader.Peek(3)
	if err != nil {
		return nil, err
	}

	log.Printf("checking frame size")

	// check the frame size; using bit shifting for efficiency;
	// this allows us to determine early whether the frame is
	// good without consuming the reader and possibly skipping
	// sync bytes if there is corruption
	frameSize := int(uint16(three[1]) << 8 & uint16(three[2]))
	log.Printf("frame size: %d", frameSize)

	if frameSize > AvatarMaxFramesSize {
		return nil, fmt.Errorf("this frame is over max frame size: %d", frameSize)
	}

	log.Printf("reading frame")

	// now that we know the frame size, we can read the
	// whole frame and check the CRC; the frame size
	// includes the sync byte and the CRC
	frame, err := r.reader.Peek(frameSize) // frame minus sync
	if err != nil {
		return nil, err
	}

	log.Printf("read frame")

	// the stated CRC
	l := len(frame)
	log.Printf("frame size: %d", l)

	crc := uint16(frame[l-2]) << 8 & uint16(frame[l-1])

	log.Printf("read crc")

	// the calculated CRC
	r.crc.Write(frame[:l-2])
	ourCrc := r.crc.Crc()

	// check the crc
	if crc != ourCrc {
		return nil, fmt.Errorf("crc doesn't match: expected %d but calculated %d", crc, ourCrc)
	}

	log.Printf("crc ok")

	// everything is okay, now
	// we actually read the frame
	_, err = io.ReadFull(r.reader, frame) // careful, overwriting data and making assumptions
	if err != nil {
		return nil, err
	}

	header := new(DataFrameHeader)
	buf := bytes.NewBuffer(frame[:AvatarHeaderSize])
	err = binary.Read(buf, binary.BigEndian, header)
	if err != nil {
		return nil, err
	}

	// get the size of the payload
	pSize := header.PayloadSize()

	// do a sanity check
	expFrameSize := 1 + AvatarHeaderSize + pSize + 2
	if expFrameSize != frameSize {
		return nil, fmt.Errorf("frameSize didn't jive, expected: %d got: %d", expFrameSize, frameSize)
	}

	// get the payload
	payload := frame[AvatarHeaderSize : l-2] // excluding header and crc

	// allocate the slices for the data
	var (
		samples    = header.Samples()
		channels   = header.Channels()
		hasTrigger = header.HasTriggerChannel()
		data       = NewSamplingBuffer(channels, samples, 1)
		p          = make([]float64, channels*samples*AvatarPointSize)
	)

	// write the samples in blocks
	for j := 0; j < samples; j++ {
		if hasTrigger {
			// skip the trigger channel
			payload = payload[AvatarPointSize:]
		}
		for k := 0; k < channels; k++ {
			p[j*channels+k] = consumeDataPoint(payload, float64(header.VoltRange()))
			payload = payload[AvatarPointSize:]
		}
	}
	data.PushSlice(p)

	dataFrame = &DataFrame{
		DataFrameHeader: *header,
		data:            data,
		received:        timeReceived,
		crc:             crc,
	}
	return
}

func consumeDataPoint(payload []byte, voltRange float64) float64 {
	raw := uint32(payload[0])<<16 | uint32(payload[1])<<8 | uint32(payload[2])
	return ((float64(raw) / float64(1000) / float64(AvatarAdcRange)) * voltRange)
}
