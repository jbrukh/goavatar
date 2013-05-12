//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package goavatar

import (
	"time"
)

// A generic data frame.
type DataFrame interface {
	Buffer() *BlockBuffer
	Channels() int
	Samples() int
	SampleRate() int
	Received() time.Time
	Generated() time.Time
}

type GenericDataFrame struct {
	buffer     *BlockBuffer
	channels   int
	samples    int
	sampleRate int
	timestamps []uint32
}

func NewGenericDataFrame(buffer *BlockBuffer, channels, samples, sampleRate int, timestamps []uint32) *GenericDataFrame {
	return &GenericDataFrame{
		buffer:     buffer,
		channels:   channels,
		samples:    samples,
		sampleRate: sampleRate,
		timestamps: timestamps,
	}
}

func (df *GenericDataFrame) Buffer() *BlockBuffer {
	return df.buffer
}

func (df *GenericDataFrame) Channels() int {
	return df.channels
}

func (df *GenericDataFrame) Samples() int {
	return df.samples
}

func (df *GenericDataFrame) SampleRate() int {
	return df.sampleRate
}

// func (df *GenericDataFrame) Received() time.Time {
// 	return NanosToTime(df.timestamps[0]) // TODO
// }

// func (df *GenericDataFrame) Generated() time.Time {
// 	return NanosToTime(df.timestamps[0])
// }

func (df *GenericDataFrame) Timestamps() []uint32 {
	return df.timestamps
}
