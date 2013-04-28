package goavatar

import (
	"testing"
)

func TestNewBuffer(t *testing.T) {
	b := NewSamplingBuffer(2, 40, 2)

	if b.Size() != 0 || len(b.data) != 0 {
		t.Errorf("should be empty")
	}

	if b.Channels() != 2 {
		t.Errorf("wrong number of channels")
	}

	if b.SampleRate() != 2 {
		t.Errorf("wrong sample rate")
	}
}

func TestAppendNext(t *testing.T) {
	b1 := NewSamplingBuffer(2, 40, 2)
	b2 := NewSamplingBuffer(2, 40, 3)

	p := []float64{1, 1}
	b1.PushSlice(p)

	if b1.Size() != 1 {
		t.Errorf("wrong data point count")
	}

	next := b1.Next(1)
	if next.Size() != 1 {
		t.Errorf("bad Next()")
	}
	if b1.Size() != 0 {
		t.Errorf("size didn't go down")
	}

	// try appending an empty buffer
	b1.Append(b2)
	if b1.Size() != 0 {
		t.Errorf("oops, should have been length 0")
	}

	p2 := []float64{1, 1, 2, 2}
	b2.PushSlice(p2)
	b1.Append(b2)
	if b1.Size() != 2 || b2.Size() != 2 {
		t.Errorf("should have been size 2")
	}

	next = b1.Next(1)
	if next.Size() != 1 {
		t.Errorf("bad Next()")
	}
	if b1.Size() != 1 && b2.Size() != 2 {
		t.Errorf("bad Next()")
	}
}

func TestIncongruentBuffers(t *testing.T) {
	b1 := NewSamplingBuffer(2, 40, 2)
	b2 := NewSamplingBuffer(3, 40, 3)
	p := []float64{1, 1, 2, 2, 3, 3}

	b1.PushSlice(p)
	b2.PushSlice(p)

	defer func() {
		if r := recover(); r != nil {
			// ok!
		}
	}()

	b1.Append(b2)
}

func TestSampling(t *testing.T) {
	b := NewSamplingBuffer(2, 40, 3)
	p := []float64{0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9}
	b.PushSlice(p)

	r := b.SampleNext(4)
	if r.Size() != 2 {
		t.Errorf("wrong number of samples")
	}
	if !(r.data[0] == 0 && r.data[1] == 0 && r.data[2] == 3 && r.data[3] == 3) {
		t.Errorf("wrong samples")
	}
	r = b.SampleNext(4)
	if r.Size() != 1 {
		t.Errorf("wrong number of samples")
	}
	if !(r.data[0] == 6 && r.data[1] == 6) {
		t.Errorf("wrong samples")
	}
	r = b.SampleNext(1)
	if r.Size() != 0 {
		t.Errorf("wrong number of samples")
	}
	r = b.SampleNext(1)
	if r.Size() != 1 {
		t.Errorf("wrong number of samples")
	}
	if !(r.data[0] == 9 && r.data[1] == 9) {
		t.Errorf("wrong samples")
	}

}
