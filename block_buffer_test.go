package goavatar

import (
	"bytes"
	"encoding/binary"
	//"log"
	"testing"
)

func TestNewBlockBuffer__Normal(t *testing.T) {
	b := NewBlockBuffer(2, 10)
	if b.pluckRate != 1 {
		t.Errorf("default pluck rate should have been 1")
	}
	if b.blockSize != 24 {
		t.Errorf("block size is wrong")
	}
	if b.channels != 2 {
		t.Errorf("wrong number of channels")
	}

	b = NewBlockBuffer(10, 1)
	if b.pluckRate != 1 {
		t.Errorf("default pluck rate should have been 1")
	}
	if b.blockSize != 88 {
		t.Errorf("block size is wrong")
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

func TestBlockBufferAppend__Comparable(t *testing.T) {
	var (
		b1 = NewBlockBuffer(2, 10)
		b2 = NewBlockBuffer(2, 5)
	)

	appendBlock(b1.buf, []float64{1, 1}, 1)
	appendBlock(b1.buf, []float64{2, 2}, 2)
	appendBlock(b1.buf, []float64{3, 3}, 3)

	appendBlock(b2.buf, []float64{4, 4}, 4)
	appendBlock(b2.buf, []float64{5, 5}, 5)
	appendBlock(b2.buf, []float64{6, 6}, 6)

	if b1.Size() != 3 || b2.Size() != 3 {
		t.Errorf("size mismatch")
	}

	b1.Append(b2)
	if b1.Size() != 6 && b2.Size() != 3 {
		t.Errorf("size mismatch")
	}

	out1 := b1.DownSample(1)
	v, ts := readBlock(out1.buf, 2)
	if v[0] != 1 || v[1] != 1 || ts != 1 {
		t.Errorf("wrong read")
	}

	out1 = b1.DownSample(5)
	v, ts = readBlock(out1.buf, 2)
	if v[0] != 2 || v[1] != 2 || ts != 2 {
		t.Errorf("wrong read")
	}

	v, ts = readBlock(out1.buf, 2)
	if v[0] != 3 || v[1] != 3 || ts != 3 {
		t.Errorf("wrong read")
	}

	v, ts = readBlock(out1.buf, 2)
	if v[0] != 4 || v[1] != 4 || ts != 4 {
		t.Errorf("wrong read")
	}

	v, ts = readBlock(out1.buf, 2)
	if v[0] != 5 || v[1] != 5 || ts != 5 {
		t.Errorf("wrong read")
	}

	v, ts = readBlock(out1.buf, 2)
	if v[0] != 6 || v[1] != 6 || ts != 6 {
		t.Errorf("wrong read")
	}
}

func TestBlockBufferAppend__Empty(t *testing.T) {
	var (
		b1 = NewBlockBuffer(2, 10)
		b2 = NewBlockBuffer(2, 5)
	)

	appendBlock(b1.buf, []float64{1, 1}, 1)
	appendBlock(b1.buf, []float64{2, 2}, 2)
	appendBlock(b1.buf, []float64{3, 3}, 3)

	if b1.Size() != 3 || b2.Size() != 0 {
		t.Errorf("size mismatch")
	}

	b1.Append(b2)
	if b1.Size() != 3 && b2.Size() != 0 {
		t.Errorf("size mismatch")
	}

	out1 := b1.DownSample(3)
	v, ts := readBlock(out1.buf, 2)
	if v[0] != 1 || v[1] != 1 || ts != 1 {
		t.Errorf("wrong read")
	}

	v, ts = readBlock(out1.buf, 2)
	if v[0] != 2 || v[1] != 2 || ts != 2 {
		t.Errorf("wrong read")
	}

	v, ts = readBlock(out1.buf, 2)
	if v[0] != 3 || v[1] != 3 || ts != 3 {
		t.Errorf("wrong read")
	}
}

func TestBlockBufferDownSample__Normal(t *testing.T) {
	var (
		b1 = NewBlockBuffer(2, 10)
	)
	appendBlock(b1.buf, []float64{1, 1}, 1)
	appendBlock(b1.buf, []float64{2, 2}, 2)
	appendBlock(b1.buf, []float64{3, 3}, 3)
	appendBlock(b1.buf, []float64{4, 4}, 4)
	appendBlock(b1.buf, []float64{5, 5}, 5)
	appendBlock(b1.buf, []float64{6, 6}, 6)
	appendBlock(b1.buf, []float64{7, 7}, 7)
	appendBlock(b1.buf, []float64{8, 8}, 8)
	appendBlock(b1.buf, []float64{9, 9}, 9)
	appendBlock(b1.buf, []float64{10, 10}, 10)

	b1.PluckRate(3)

	out := b1.DownSample(10)
	if out.Size() != 4 {
		t.Errorf("wrong downsample size")
	}

	v, ts := readBlock(out.buf, 2)
	if v[0] != 1 || v[1] != 1 || ts != 1 {
		t.Errorf("wrong read")
	}

	v, ts = readBlock(out.buf, 2)
	if v[0] != 4 || v[1] != 4 || ts != 4 {
		t.Errorf("wrong read")
	}

	v, ts = readBlock(out.buf, 2)
	if v[0] != 7 || v[1] != 7 || ts != 7 {
		t.Errorf("wrong read")
	}

	v, ts = readBlock(out.buf, 2)
	if v[0] != 10 || v[1] != 10 || ts != 10 {
		t.Errorf("wrong read")
	}

}

func TestBlockBufferDownSample__Split(t *testing.T) {
	var (
		b1 = NewBlockBuffer(2, 10)
	)
	appendBlock(b1.buf, []float64{1, 1}, 1)
	appendBlock(b1.buf, []float64{2, 2}, 2)
	appendBlock(b1.buf, []float64{3, 3}, 3)
	appendBlock(b1.buf, []float64{4, 4}, 4)
	appendBlock(b1.buf, []float64{5, 5}, 5)
	appendBlock(b1.buf, []float64{6, 6}, 6)
	appendBlock(b1.buf, []float64{7, 7}, 7)
	appendBlock(b1.buf, []float64{8, 8}, 8)
	appendBlock(b1.buf, []float64{9, 9}, 9)
	appendBlock(b1.buf, []float64{10, 10}, 10)

	b1.PluckRate(3)

	out := b1.DownSample(5)
	if out.Size() != 2 {
		t.Errorf("wrong downsample size")
	}

	v, ts := readBlock(out.buf, 2)
	if v[0] != 1 || v[1] != 1 || ts != 1 {
		t.Errorf("wrong read")
	}

	v, ts = readBlock(out.buf, 2)
	if v[0] != 4 || v[1] != 4 || ts != 4 {
		t.Errorf("wrong read")
	}

	out = b1.DownSample(5)
	if out.Size() != 2 {
		t.Errorf("wrong downsample size")
	}

	v, ts = readBlock(out.buf, 2)
	if v[0] != 7 || v[1] != 7 || ts != 7 {
		t.Errorf("wrong read")
	}

	v, ts = readBlock(out.buf, 2)
	if v[0] != 10 || v[1] != 10 || ts != 10 {
		t.Errorf("wrong read")
	}

}

func TestBlockBufferDownSample__NotYet(t *testing.T) {
	var (
		b1 = NewBlockBuffer(2, 10)
	)
	appendBlock(b1.buf, []float64{1, 1}, 1)
	appendBlock(b1.buf, []float64{2, 2}, 2)
	appendBlock(b1.buf, []float64{3, 3}, 3)
	appendBlock(b1.buf, []float64{4, 4}, 4)
	appendBlock(b1.buf, []float64{5, 5}, 5)
	appendBlock(b1.buf, []float64{6, 6}, 6)
	appendBlock(b1.buf, []float64{7, 7}, 7)
	appendBlock(b1.buf, []float64{8, 8}, 8)
	appendBlock(b1.buf, []float64{9, 9}, 9)
	appendBlock(b1.buf, []float64{10, 10}, 10)

	b1.PluckRate(3)
	b1.parity = 1

	out := b1.DownSample(2)
	if out.Size() != 0 {
		t.Errorf("wrong downsample size")
	}
	out = b1.DownSample(4)
	if out.Size() != 2 {
		t.Errorf("wrong downsample size")
	}
}

func TestBlockBufferAppendBlock__Normal(t *testing.T) {
	b := NewBlockBuffer(2, 10)
	if b.Size() != 0 {
		t.Errorf("wrong size")
	}

	b.AppendBlock([]float64{1, 2}, 11)
	if b.Size() != 1 {
		t.Errorf("wrong size")
	}
}

func TestBlockBufferAppendBlock__NotComparable(t *testing.T) {
	b := NewBlockBuffer(2, 10)
	if b.Size() != 0 {
		t.Errorf("wrong size")
	}
	testPanic(t, func() {
		b.AppendBlock([]float64{1, 2, 3}, 11)
	})
}

func appendBlock(buf *bytes.Buffer, v []float64, ts int64) {
	binary.Write(buf, binary.BigEndian, v)
	binary.Write(buf, binary.BigEndian, ts)
}

func readBlock(buf *bytes.Buffer, n int) (v []float64, ts int64) {
	v = make([]float64, n)
	binary.Read(buf, binary.BigEndian, &v)
	binary.Read(buf, binary.BigEndian, &ts)
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
