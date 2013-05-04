package goavatar

// A SamplingBuffer is an efficient multi-channel buffer that
// supports sampling of the underlying data.
type SamplingBuffer struct {
	channels   int
	data       []float64
	parity     int // sampling offset between downsamples
	sampleRate int
}

// Create a new SamplingBuffer with enough capacity for
// "capacity" datapoints across "channels" channels.
func NewSamplingBuffer(channels, capacity, sampleRate int) *SamplingBuffer {
	if channels < 1 || capacity < 1 || sampleRate < 1 {
		panic("incorrect parameters")
	}
	return &SamplingBuffer{
		channels:   channels,
		data:       make([]float64, 0, channels*capacity),
		sampleRate: sampleRate,
	}
}

func NewSamplingBufferFromSlice(channels, sampleRate int, data []float64) *SamplingBuffer {
	if len(data)%channels != 0 {
		panic("wrong data size")
	}
	return &SamplingBuffer{
		channels:   channels,
		data:       data,
		sampleRate: sampleRate,
	}
}

func (b *SamplingBuffer) assertSize(size int) {
	if size%b.channels != 0 {
		panic("wrong size")
	}
}

func (b *SamplingBuffer) Channels() int {
	return b.channels
}

func (b *SamplingBuffer) SampleRate() int {
	return b.sampleRate
}

// Size returns the number of data points per channel.
func (b *SamplingBuffer) Size() int {
	return len(b.data) / b.channels
}

// Push raw data into the buffer.
func (b *SamplingBuffer) PushSlice(p []float64) {
	b.assertSize(len(p))
	b.data = append(b.data, p...)
}

// Append one buffer to another.
func (b *SamplingBuffer) Append(buf *SamplingBuffer) {
	if b.channels != buf.channels {
		panic("buffers are not comparable")
	}
	b.data = append(b.data, buf.data...)
}

func (b *SamplingBuffer) Next(n int) *SamplingBuffer {
	if n > b.Size() {
		panic("not enough elements to return")
	}
	end := n * b.channels
	buf := NewSamplingBufferFromSlice(b.channels, b.sampleRate, b.data[:end])
	b.data = b.data[end:]
	return buf
}

// Returns the channel data for a particular channel [0...channels).
func (b *SamplingBuffer) ChannelData(channel int) []float64 {
	if channel < 0 || channel >= b.channels {
		panic("no such channel")
	}
	size := b.Size()
	result := make([]float64, size)
	for i, _ := range result {
		result[i] = b.data[i*b.channels+channel]
	}
	return result
}

// Remove the next n points and sample them according
// to the sample rate.
func (b *SamplingBuffer) SampleNext(n int) *SamplingBuffer {
	buf := b.Next(n).data

	// allocate a new buffer for the sampled result
	result := NewSamplingBuffer(b.channels, n/b.sampleRate+1, 1)

	// now go through each data point
	for inx := 0; inx < n*b.channels; inx += b.channels {

		// this is a sample
		if b.parity == 0 {
			p := buf[inx : inx+b.channels]
			result.PushSlice(p)
		}

		// increment the parity
		b.parity = (b.parity + 1) % b.sampleRate
	}
	return result
}

// Return the raw data.
func (b *SamplingBuffer) RawData() []float64 {
	return b.data
}
