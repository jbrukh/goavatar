//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package goavatar

import (
	"time"
)

type DataFrame interface {
	Buffer() *SamplingBuffer
	Channels() int
	Samples() int
	SampleRate() int
	Received() time.Time
	Generated() time.Time
}
