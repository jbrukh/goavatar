//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package goavatar

import (
	//"log"
	"testing"
)

func TestNewBlockBuffer__Normal(t *testing.T) {
	b := NewBlockBuffer(2, 10)
	if b.pluckRate != 1 {
		t.Errorf("default pluck rate should have been 1")
	}
	if b.channels != 2 {
		t.Errorf("wrong number of channels")
	}

	b = NewBlockBuffer(10, 1)
	if b.pluckRate != 1 {
		t.Errorf("default pluck rate should have been 1")
	}
	if b.channels != 10 {
		t.Errorf("wrong number of channels")
	}
}

func TestNewBlockBuffer__NegativeChannels(t *testing.T) {
	testPanic(t, func() {
		NewBlockBuffer(-1, 10)
	})
}

func TestBlockBuffer__PluckRate(t *testing.T) {
	b := NewBlockBuffer(10, 1)
	if b.PluckRate(10); b.pluckRate != 10 {
		t.Errorf("pluck rate failed to set")
	}
}

func TestBlockBufferAppend__NotComparable(t *testing.T) {
	var (
		b1 = NewBlockBuffer(2, 10)
		b2 = NewBlockBuffer(3, 10)
		b3 = NewBlockBuffer(2, 5)
	)
	testPanic(t, func() {
		b1.Append(b2)
	})
	testPanic(t, func() {
		b2.Append(b1)
	})
	testPanic(t, func() {
		b3.Append(b2)
	})
}

func TestBlockBufferAppendSample__Comparable(t *testing.T) {
	var (
		b  = mockBlockBuffer()
		b1 = b.DownSample(3)
		b2 = b.DownSample(3)
	)

	if b1.Samples() != 3 || b2.Samples() != 3 {
		t.Errorf("size mismatch")
	}

	b1.Append(b2)
	if b1.Samples() != 6 && b2.Samples() != 3 {
		t.Errorf("size mismatch")
	}

	out := b1.DownSample(1)
	v, ts := out.NextSample()
	if v[0] != 1 || v[1] != 1 || ts != 1 {
		t.Errorf("wrong read")
	}

	out = b1.DownSample(5)
	v, ts = out.NextSample()
	if v[0] != 2 || v[1] != 2 || ts != 2 {
		t.Errorf("wrong read")
	}

	v, ts = out.NextSample()
	if v[0] != 3 || v[1] != 3 || ts != 3 {
		t.Errorf("wrong read")
	}

	v, ts = out.NextSample()
	if v[0] != 4 || v[1] != 4 || ts != 4 {
		t.Errorf("wrong read")
	}

	v, ts = out.NextSample()
	if v[0] != 5 || v[1] != 5 || ts != 5 {
		t.Errorf("wrong read")
	}

	v, ts = out.NextSample()
	if v[0] != 6 || v[1] != 6 || ts != 6 {
		t.Errorf("wrong read")
	}
}

func TestBlockBufferAppendSample__Empty(t *testing.T) {
	var (
		b1 = mockBlockBuffer().DownSample(3)
		b2 = NewBlockBuffer(2, 5)
	)

	if b1.Samples() != 3 || b2.Samples() != 0 {
		t.Errorf("size mismatch")
	}

	b1.Append(b2)
	if b1.Samples() != 3 && b2.Samples() != 0 {
		t.Errorf("size mismatch")
	}

	out := b1.DownSample(3)
	v, ts := out.NextSample()
	if v[0] != 1 || v[1] != 1 || ts != 1 {
		t.Errorf("wrong read")
	}

	v, ts = out.NextSample()
	if v[0] != 2 || v[1] != 2 || ts != 2 {
		t.Errorf("wrong read")
	}

	v, ts = out.NextSample()
	if v[0] != 3 || v[1] != 3 || ts != 3 {
		t.Errorf("wrong read")
	}
}

func TestBlockBufferDownSample__Normal(t *testing.T) {
	b1 := mockBlockBuffer()
	b1.PluckRate(3)

	out := b1.DownSample(10)
	if out.Samples() != 4 {
		t.Errorf("wrong downsample size")
	}

	v, ts := out.NextSample()
	if v[0] != 1 || v[1] != 1 || ts != 1 {
		t.Errorf("wrong read")
	}

	v, ts = out.NextSample()
	if v[0] != 4 || v[1] != 4 || ts != 4 {
		t.Errorf("wrong read")
	}

	v, ts = out.NextSample()
	if v[0] != 7 || v[1] != 7 || ts != 7 {
		t.Errorf("wrong read")
	}

	v, ts = out.NextSample()
	if v[0] != 10 || v[1] != 10 || ts != 10 {
		t.Errorf("wrong read")
	}

}

func TestBlockBufferDownSample__Split(t *testing.T) {
	b1 := mockBlockBuffer()
	b1.PluckRate(3)

	out := b1.DownSample(5)
	if out.Samples() != 2 {
		t.Errorf("wrong downsample size")
	}

	v, ts := out.NextSample()
	if v[0] != 1 || v[1] != 1 || ts != 1 {
		t.Errorf("wrong read")
	}

	v, ts = out.NextSample()
	if v[0] != 4 || v[1] != 4 || ts != 4 {
		t.Errorf("wrong read")
	}

	out = b1.DownSample(5)
	if out.Samples() != 2 {
		t.Errorf("wrong downsample size")
	}

	v, ts = out.NextSample()
	if v[0] != 7 || v[1] != 7 || ts != 7 {
		t.Errorf("wrong read")
	}

	v, ts = out.NextSample()
	if v[0] != 10 || v[1] != 10 || ts != 10 {
		t.Errorf("wrong read")
	}

}

func TestBlockBufferDownSample__NotYet(t *testing.T) {
	b1 := mockBlockBuffer()

	b1.PluckRate(3)
	b1.parity = 1

	out := b1.DownSample(2)
	if out.Samples() != 0 {
		t.Errorf("wrong downsample size")
	}
	out = b1.DownSample(4)
	if out.Samples() != 2 {
		t.Errorf("wrong downsample size")
	}
}

func TestBlockBufferAppendSample__Normal(t *testing.T) {
	b := NewBlockBuffer(2, 10)
	if b.Samples() != 0 {
		t.Errorf("wrong size")
	}

	b.AppendSample([]float64{1, 2}, 11)
	if b.Samples() != 1 {
		t.Errorf("wrong size")
	}
}

func TestBlockBufferAppendSample__NotComparable(t *testing.T) {
	b := NewBlockBuffer(2, 10)
	if b.Samples() != 0 {
		t.Errorf("wrong size")
	}
	testPanic(t, func() {
		b.AppendSample([]float64{1, 2, 3}, 11)
	})
}

func mockBlockBuffer() (b *BlockBuffer) {
	b = NewBlockBuffer(2, 10)
	b.appendBlocks(
		[]float64{
			1, 1,
			2, 2,
			3, 3,
			4, 4,
			5, 5,
			6, 6,
			7, 7,
			8, 8,
			9, 9,
			10, 10,
		},
		[]int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
	)
	return
}

func testPanic(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r != nil {
			// ok
		}
	}()
	f()
	t.Errorf("should have panicked")
}
