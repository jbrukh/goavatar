//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package etc

import (
	. "github.com/jbrukh/goavatar"
	"time"
)

// ----------------------------------------------------------------- //
// FOR MOCKING AVATAR FRAMES ONLY -- DO NOT USE OUTSIDE OF TESTING
// ----------------------------------------------------------------- //

var AvatarSampleRates = []int{250, 500, 1000}

const (
	AvatarFracSecs  = time.Duration(4096) // fractional second parts
	AvatarPointSize = 3
)

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

func (h *AvatarHeader) SampleRate() (sampleRate int) {
	sr := int(h.FieldSampleRateVersion >> 6)
	if sr < 0 || sr > 2 {
		return 0
	}
	return AvatarSampleRates[sr]
}

func (h *AvatarHeader) Version() int {
	return int(h.FieldSampleRateVersion & 0x3F)
}

func (h *AvatarHeader) FrameSize() int {
	return int(h.FieldFrameSize)
}

func (h *AvatarHeader) FrameType() int {
	return int(h.FieldFrameType)
}

func (h *AvatarHeader) FrameCount() int {
	return int(h.FieldFrameCount)
}

func (h *AvatarHeader) HasTriggerChannel() bool {
	return (h.FieldChannels >> 7) > 0
}

func (h *AvatarHeader) Channels() int {
	// zero the first bit for the trigger channel
	return int(h.FieldChannels & 0x7F)
}

func (h *AvatarHeader) Samples() int {
	return int(h.FieldSamples)
}

func (h *AvatarHeader) VoltRange() int {
	return int(h.FieldVoltRange)
}

func (h *AvatarHeader) Generated() time.Time {
	return time.Unix(int64(h.FieldTimestamp), int64(time.Duration(h.FieldFracSecs)*time.Second/AvatarFracSecs))
}

func (h *AvatarHeader) Timestamps() []int64 {
	timestamps := make([]int64, int(h.FieldSamples))
	ts := h.Generated().UnixNano()
	δ := 1000000000 / h.SampleRate()
	for t := range timestamps {
		timestamps[t] = ts + int64(δ)*int64(t)
	}
	return timestamps
}

func (h *AvatarHeader) PayloadSize() int {
	return h.Channels() * h.Samples() * AvatarPointSize
}

type AvatarDataFrame struct {
	AvatarHeader
	data     *SamplingBuffer // processed data, in a multibuffer
	received time.Time       // time this frame was received locally
	crc      uint16          // crc of the frame
}

func (df *AvatarDataFrame) Buffer() *SamplingBuffer {
	return df.data
}

func (df *AvatarDataFrame) ChannelData(channel int) []float64 {
	return df.data.ChannelData(channel)
}

func (df *AvatarDataFrame) Received() time.Time {
	return df.received
}

func (df *AvatarDataFrame) Crc() uint16 {
	return df.crc
}
