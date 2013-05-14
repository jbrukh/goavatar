//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package goavatar

import (
//"time"
)

// A generic data frame.
type DataFrame interface {
	Buffer() *BlockBuffer
	SampleRate() int
}

type GenericDataFrame struct {
	buffer     *BlockBuffer
	sampleRate int
}

func NewGenericDataFrame(buffer *BlockBuffer, sampleRate int) *GenericDataFrame {
	return &GenericDataFrame{
		buffer:     buffer,
		sampleRate: sampleRate,
	}
}

func (df *GenericDataFrame) Buffer() *BlockBuffer {
	return df.buffer
}

func (df *GenericDataFrame) SampleRate() int {
	return df.sampleRate
}
