//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package goavatar

// ----------------------------------------------------------------- //
// Constants
// ----------------------------------------------------------------- //

const (
	BlockBufferValueSize     = 8
	BlockBufferTimestampSize = 8
)

// BlockBuffer is a data structure for holding multi-channel
// time series. It is backed by slices and is optimized for
// append operations; it is good for real-time streaming of
// multi-channel data because the channel data is stored in
// a "parallel". You can also down-sample the time series in
// the buffer by setting the "pluck rate" (e.g. down-sampling
// rate) and calling DownSample().
type BlockBuffer struct {
	channels  int // number of channels per sample
	parity    int // plucking offset
	pluckRate int // downsampling, or plucking rate

	values []float64 // data
	ts     []int64   // timestamps
}

// Create a new BlockBuffer anticipating the given
// number of channels and the given sample size.
func NewBlockBuffer(channels, samples int) *BlockBuffer {
	if channels < 0 || samples < 1 {
		panic("bad parameters")
	}
	return &BlockBuffer{
		channels:  channels,
		pluckRate: 1,
		values:    make([]float64, 0, channels*samples),
		ts:        make([]int64, 0, samples),
	}
}

// Set the downsampling rate. This means that down
// sampling will select 1 out of k samples.
func (b *BlockBuffer) PluckRate(k int) {
	b.pluckRate = k
}

// Get the number of channels in this data.
func (b *BlockBuffer) Channels() int {
	return b.channels
}

// The number of samples in this BlockBuffer.
func (b *BlockBuffer) Samples() int {
	return len(b.values) / b.channels
}

// The timestamp array of this BlockBuffer. Note
// timestamps are format-agnostic.
func (b *BlockBuffer) Timestamps() []int64 {
	return b.ts
}

func (b *BlockBuffer) TransformTs(f func(s int, ts int64) int64) {
	for s, ts := range b.ts {
		b.ts[s] = f(s, ts)
	}
}

// Append a data from a BlockBuffer to the existing BlockBuffer,
// ignoring the latter's pluck rate. The BlockBuffers must be
// comparable in the sense of channels.
func (b *BlockBuffer) Append(bb *BlockBuffer) {
	if bb.channels != b.channels {
		panic("not comparable")
	}
	b.appendBlocks(bb.values, bb.ts)
}

// Append a single sample to the BlockBuffer. This is not
// particularly efficient, but must be done when translating
// low-level device data.
func (b *BlockBuffer) AppendSample(v []float64, ts int64) {
	if len(v) != b.channels {
		panic("not comparable")
	}
	b.appendBlocks(v, []int64{ts})
}

// Get the next n samples and downsamples them according
// to the down-sampling rate.
func (b *BlockBuffer) DownSample(n int) (bb *BlockBuffer) {
	bb = NewBlockBuffer(b.channels, n/b.pluckRate+1)
	samples := b.Samples()
	if n > samples {
		n = samples
	}

	for s := 0; s < n; s++ {
		if b.parity == 0 {
			v, ts := b.NextSample()
			bb.AppendSample(v, ts)
		} else {
			b.NextSample()
		}

		// increment the parity
		b.parity = (b.parity + 1) % b.pluckRate
	}
	return
}

func (b *BlockBuffer) NextSample() (v []float64, ts int64) {
	v = b.values[:b.channels]
	ts = b.ts[0]
	// get rid of the leading values
	b.values = b.values[b.channels:]
	b.ts = b.ts[1:]
	return
}

// Arrays transforms the underlying data into "sequential"
// channel arrays. This operation is O(n) on the number of
// individual channel data points and timestamps and requires
// O(2*n) space.
func (b *BlockBuffer) Arrays() ([][]float64, []int64) {
	var (
		samples = b.Samples()
		values  = make([][]float64, b.channels)
	)

	// allocate
	for i := range values {
		values[i] = make([]float64, samples)
	}

	// restructure
	for s := 0; s < samples; s++ {
		v := b.values[s*b.channels : (s+1)*b.channels]
		for c, value := range v {
			values[c][s] = value
		}
	}

	// return
	return values, b.Timestamps()
}

func (b *BlockBuffer) appendBlocks(v []float64, ts []int64) {
	b.values = append(b.values, v...)
	b.ts = append(b.ts, ts...)
}
