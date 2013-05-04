package goavatar

import (
	"time"
)

type Frame interface {
	Buffer() *SamplingBuffer
	Channels() int
	SampleRate() int
	Received() time.Time
	Generated() time.Time
}
