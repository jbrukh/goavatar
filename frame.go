//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package goavatar

// A generic data frame interface.
type DataFrame interface {
	Buffer() *BlockBuffer
	SampleRate() int
}

// A generic data frame implementation.
type dataFrame struct {
	buffer     *BlockBuffer
	sampleRate int
}

// Create a new generic DataFrame.
func NewDataFrame(buffer *BlockBuffer, sampleRate int) DataFrame {
	return &dataFrame{
		buffer:     buffer,
		sampleRate: sampleRate,
	}
}

func (df *dataFrame) Buffer() *BlockBuffer {
	return df.buffer
}

func (df *dataFrame) SampleRate() int {
	return df.sampleRate
}
