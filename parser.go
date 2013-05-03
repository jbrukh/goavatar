package goavatar

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

const (
	AvatarSyncByte          = 0xAA
	AvatarExpectedVersion   = 3 // version
	AvatarExpectedFrameType = 1 // data frame
	AvatarFracSecs          = time.Duration(4096)
	AvatarDataPointBytes    = 3
	AvatarSanePayload       = 8 * 32 * AvatarDataPointBytes
	AvatarAdcRange          = 16777216 // 2^24
	AvatarMaxChannels       = 8
)

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
	crc      uint16          // CRC-16-CCIT calculated on the entire frame not including CRC
	received time.Time       // time this frame was received locally
}

// String
func (df *DataFrame) String() string {
	return fmt.Sprintf("\n%+v\n", *df)
}

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

// func (df *DataFrame) ChannelData(channel int) []float64 {
// 	if channel < 0 || channel > AvatarMaxChannels {
// 		panic("you are trying to select a channel that doesn't exist")
// 	}
// 	return df.data[channel]
// }

// func (df *DataFrame) ChannelDatas() *MultiBuffer {
// 	return NewMultiBufferFromSlice(df.data[1 : df.Channels()+1])
// }

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

// SampleRate: the number of data samples delivered in one second (per channel)
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

// HasTriggerChannel
func (h *DataFrameHeader) HasTriggerChannel() bool {
	return (h.FieldChannels >> 7) > 0
}

// Channels
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
	return h.Channels() * h.Samples() * AvatarDataPointBytes
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
func (r *avatarParser) ConsumeHeader() (h *DataFrameHeader, err error) {
	h = new(DataFrameHeader)

	// let's peek at the header; there's a chance it is corrupted, and
	// if so, we will want just skip to the next sync instead of reading
	// it in (or else we risk losing more data because we're not properly
	// synced up)
	buf, err := r.reader.Peek(19)
	if err != nil {
		return nil, err
	}
	buffer := bytes.NewBuffer(buf)

	// read-ahead the header into the buffer to check it
	err = binary.Read(buffer, binary.BigEndian, h)
	if err != nil {
		return nil, err
	}

	// check that it is sane
	pSize := h.PayloadSize()
	if pSize > AvatarSanePayload || h.FrameSize()-22 != pSize {
		return nil, fmt.Errorf("Size of payload over threshhold (or doesn't match); wanted %d", pSize)
	} else {
		// buf should be 19 bytes long
		if _, err := r.reader.Read(buf); err != nil {
			return nil, err
		}
	}

	// note the header
	err = binary.Write(&r.crc, binary.BigEndian, h)
	return
}

func (r *avatarParser) ConsumePayload(header *DataFrameHeader) (b *SamplingBuffer, err error) {
	// ascertain the size of the payload; if the frame is corrupted,
	// this size will probably be too large, which will result in a
	// bad reading of the data...

	pSize := header.PayloadSize()
	// ok, now read it
	payload := make([]byte, pSize)
	n := 0

	// read until the whole payload is read
	for n != pSize {
		nRead, err := r.reader.Read(payload[n:])
		if err != nil { // BUG! will be err when nRead < expected
			return nil, err
		}
		n += nRead
	}

	// note the payload
	r.crc.Write(payload)

	// allocate the slices for the data
	samples, channels := header.Samples(), header.Channels()
	hasTrigger := header.HasTriggerChannel()
	b = NewSamplingBuffer(channels, samples, 1)

	// TODO use one array
	for j := 0; j < samples; j++ {
		if hasTrigger {
			// just skip this
			payload = payload[3:]
		}
		p := make([]float64, channels)
		for i, _ := range p {
			p[i] = consumeDataPoint(payload, header)
			payload = payload[3:]
		}
		b.PushSlice(p)
	}

	return
}

func consumeDataPoint(payload []byte, header *DataFrameHeader) float64 {
	raw := uint32(payload[0])<<16 | uint32(payload[1])<<8 | uint32(payload[2])
	return ((float64(raw) / float64(1000) / float64(AvatarAdcRange)) * float64(header.VoltRange()))
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
