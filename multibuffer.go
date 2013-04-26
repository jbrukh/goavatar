package goavatar

import (
	"fmt"
	//"log"
)

type MultiBuffer struct {
	channels int
	data     [][]float64
}

func NewMultiBuffer(channels, capacity int) *MultiBuffer {
	if channels < 1 || capacity < 1 {
		panic("nonsensical size")
	}
	data := make([][]float64, channels)
	for i := 0; i < channels; i++ {
		data[i] = make([]float64, 0, capacity)
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
			str := fmt.Sprintf("inconsistent widths: %d and %d", len(data[c]), w)
			panic(str)
		}
	}
	m := &MultiBuffer{
		channels: l,
		data:     data,
	}
	//log.Printf("returning multibuffer %+v", m)
	return m
}

func (b *MultiBuffer) Size() int {
	return len(b.data[0])
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
	if len(data) != b.channels {
		panic("buffer sizes not comparable")
	}
	for c := 0; c < b.channels; c++ {
		b.data[c] = append(b.data[c], data[c]...)
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
	buf := NewMultiBuffer(b.channels, n)
	for c := 0; c < b.channels; c++ {
		buf.data[c] = append(buf.data[c], b.data[c][:n]...)
		b.data[c] = b.data[c][n:]
	}
	return buf, true
}
