package goavatar

type MultiBuffer struct {
	channels int
	data     [][]float64
}

func NewMultiBuffer(channels, size int) *MultiBuffer {
	if channels < 1 || size < 1 {
		panic("nonsensical size")
	}
	data := make([][]float64, channels)
	for i := 0; i < channels; i++ {
		data[i] = make([]float64, size)
	}
	return &MultiBuffer{
		data:     data,
		channels: channels,
	}
}

func NewMultiBufferFromSlice(data [][]float64) *MultiBuffer {
	l := len(data)
	if l < 1 {
		panic("must have positive size")
	}
	w := len(data[0])
	for c := 0; c < l; c++ {
		if len(data[c]) != w {
			panic("inconsistent widths")
		}
	}
	return &MultiBuffer{
		channels: l,
		data:     data,
	}
}

func (b *MultiBuffer) AppendSample(data []float64) {
	if len(data) != b.channels {
		panic("buffer sizes not comparable")
	}
	for c := 0; c < b.channels; c++ {
		b.data[c] = append(b.data[c], data[c])
	}
}

func (b *MultiBuffer) Append(data [][]float64) {
	if len(data) != len(b.channels) {
		panic("buffer sizes not comparable")
	}
	for c := 0; c < b.channels; c++ {
		b.data[c] = append(b.data[c], data[c])
	}
}

func (b *MultiBuffer) AppendBuffer(data *MultiBuffer) {
	b.Append(data.data)
}

func (b *MultiBuffer) HasNext(n int) bool {
	if len(b.data[0]) >= n {
		return true
	}
	return false
}

func (b *MultiBuffer) Next(n int) (*MultiBuffer, bool) {
	if !b.HasNext(n) {
		return nil, false
	}
	buf := NewMultiBufferFromSlice(make([][]float64, b.channels))
	for c := 0; c < b.channels; c++ {
		buf.data[c] = append(buf.data[c], b.data[c][:n])
	}
	return buf, true
}
