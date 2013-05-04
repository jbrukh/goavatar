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
