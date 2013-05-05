//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package goavatar

import (
	"time"
)

// A generic data frame.
type DataFrame interface {
	Buffer() *SamplingBuffer
	Channels() int
	Samples() int
	SampleRate() int
	Received() time.Time
	Generated() time.Time
	Timestamps() []int64
}

type GenericDataFrame struct {
	buffer     *SamplingBuffer
	channels   int
	samples    int
	sampleRate int
	received   time.Time
	generated  time.Time
	timestamps []int64
}

func (df *GenericDataFrame) Buffer() *SamplingBuffer {
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

func (df *GenericDataFrame) Received() time.Time {
	return time.Now() // TODO
}

func (df *GenericDataFrame) Generated() time.Time {
	return time.Now() // TODO
}

func (df *GenericDataFrame) Timestamps() []int64 {
	return df.timestamps
}
