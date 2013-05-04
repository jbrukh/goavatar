//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package avatar

import (
	"fmt"
	. "github.com/jbrukh/goavatar"
	"time"
)

// ----------------------------------------------------------------- //
// AvatarEEG Data Frame and Parsing
// ----------------------------------------------------------------- //

type AvatarHeader struct {
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

// AvatarDataFrame represents the raw data that is transmitted from the AvatarEEG
// device. 
type AvatarDataFrame struct {
	AvatarHeader
	data     *SamplingBuffer // processed data, in a multibuffer
	received time.Time       // time this frame was received locally
	crc      uint16          // crc of the frame
}

// String
func (df *AvatarDataFrame) String() string {
	return fmt.Sprintf("\n%+v\n", *df)
}

// Return this dataframe as Go code.
func (df *AvatarDataFrame) AsCode() string {
	f := `&AvatarDataFrame{
		AvatarHeader:AvatarHeader{
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
		df.Buffer().RawData(),
		df.crc,
		df.received.Unix(),
		df.received.Nanosecond(),
	)

}

func (df *AvatarDataFrame) Buffer() *SamplingBuffer {
	return df.data
}

func (df *AvatarDataFrame) ChannelData(channel int) []float64 {
	return df.data.ChannelData(channel)
}

// the time this data framed was received locally
func (df *AvatarDataFrame) Received() time.Time {
	return df.received
}

// SampleRate: the number of data samples delivered in one
// second (per channel)
func (h *AvatarHeader) SampleRate() (sampleRate int) {
	sr := int(h.FieldSampleRateVersion >> 6)
	if sr < 0 || sr > 2 {
		return 0
	}
	return AvatarSampleRates[sr]
}

// Version
func (h *AvatarHeader) Version() int {
	return int(h.FieldSampleRateVersion & 0x3F)
}

// FrameSize
func (h *AvatarHeader) FrameSize() int {
	return int(h.FieldFrameSize)
}

// FrameType
func (h *AvatarHeader) FrameType() int {
	return int(h.FieldFrameType)
}

// FrameCount
func (h *AvatarHeader) FrameCount() int {
	return int(h.FieldFrameCount)
}

// HasTriggerChannel
func (h *AvatarHeader) HasTriggerChannel() bool {
	return (h.FieldChannels >> 7) > 0
}

// Channels (number of, not including trigger)
func (h *AvatarHeader) Channels() int {
	// zero the first bit for the trigger channel
	return int(h.FieldChannels & 0x7F)
}

// Samples
func (h *AvatarHeader) Samples() int {
	return int(h.FieldSamples)
}

// Range returns the range, in mVpp, of each data channel which is dependent on the
// gain and is 12 by default. This is needed to convert the raw counting data from
// the analog-to-digital converter. To convert counts to voltage, simply perform:
//
//     (value) * range / 1000 / 2^24
//
func (h *AvatarHeader) VoltRange() int {
	return int(h.FieldVoltRange)
}

// Generated (what time the frame was generated)
func (h *AvatarHeader) Generated() time.Time {
	return time.Unix(int64(h.FieldTimestamp), int64(time.Duration(h.FieldFracSecs)*time.Second/AvatarFracSecs))
}

// Payload size
func (h *AvatarHeader) PayloadSize() int {
	return h.Channels() * h.Samples() * AvatarPointSize
}
