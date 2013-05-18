//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package formats

import (
	"encoding/binary"
	"io"
	"os"
	"testing"
)

const (
	mockChannels   = 4
	mockSamples    = 10
	mockSampleRate = 250
)

const fn = "../var/obf_test"

func init() {
	f, err := newTestFile(fn)
	if err != nil {
		panic("could not create test file")
	}
	if err = writeMockData(f); err != nil {
		panic("could not generate mock data")
	}
	if err = f.Close(); err != nil {
		panic("could not close file")
	}
}

func newTestFile(fn string) (file *os.File, err error) {
	return os.OpenFile(fn, os.O_CREATE|os.O_RDWR, 0655)
}

func writeMockData(w io.Writer) (err error) {
	// make the header
	h := &OBFHeader{
		DataType:      DataTypeRaw,
		FormatVersion: FormatVersion2,
		StorageMode:   StorageModeCombined,
		Channels:      uint8(mockChannels),
		Samples:       uint32(mockSamples),
		SampleRate:    uint16(mockSampleRate),
	}

	// write the header
	if err = binary.Write(w, ByteOrder, h); err != nil {
		return
	}

	v := make([]float64, mockChannels)
	var ts32 uint32

	// make the parallel payload
	for s := 0; s < mockSamples; s++ {
		// let each channel have values that are the channel
		for c := range v {
			v[c] = float64(c)
		}
		ts32 = uint32(s)
		if err = binary.Write(w, ByteOrder, v); err != nil {
			return
		}
		if err = binary.Write(w, ByteOrder, ts32); err != nil {
			return
		}
	}

	// make the sequential payload
	v = make([]float64, mockSamples)
	ts := make([]uint32, mockSamples)
	for c := 0; c < mockChannels; c++ {
		for s := range v {
			v[s] = float64(c)
		}
		if err = binary.Write(w, ByteOrder, v); err != nil {
			return
		}
	}
	for s := 0; s < mockSamples; s++ {
		ts[s] = uint32(s)
	}
	if err = binary.Write(w, ByteOrder, ts); err != nil {
		return
	}
	return
}

func assertMockHeader(t *testing.T, h *OBFHeader) {
	if h.DataType != DataTypeRaw || h.FormatVersion != FormatVersion2 || h.StorageMode != StorageModeCombined {
		t.Errorf("bad metadata for header: %v, %v, %v; expected: %v, %v, %v",
			h.DataType, h.FormatVersion, h.StorageMode, DataTypeRaw, FormatVersion2, StorageModeCombined)
	}
	if h.Samples != mockSamples || h.Channels != mockChannels || h.SampleRate != mockSampleRate {
		t.Errorf("bad diagnostics for header")
	}
}

func mockFile(t *testing.T) (file *os.File, err error) {
	return newTestFile(fn)
}

func testWithCodec(t *testing.T, tf func(t *testing.T, oc *obfCodec)) {
	f, err := mockFile(t)
	if err != nil {
		t.Errorf("could not create mock file: %v", err)
	}
	defer f.Close()
	oc := newObfCodec(f)
	tf(t, oc)
}

func Test__ReadHeader(t *testing.T) {
	var err error
	testWithCodec(t, func(t *testing.T, oc *obfCodec) {
		// read the header in place
		if err = oc.ReadHeader(); err != nil {
			t.Errorf("could not read header in place: %v", err)
		}

		// check the header
		h := oc.Header()
		assertMockHeader(t, h)
	})
}

func Test__SeekHeader(t *testing.T) {
	var err error
	testWithCodec(t, func(t *testing.T, oc *obfCodec) {
		// seek somewhere
		if err = oc.SeekValues(); err != nil {
			t.Errorf("could not seek to the values")
		}

		// seek back to the header
		if err = oc.SeekHeader(); err != nil {
			t.Errorf("could not seek to the values")
		}

		// read the header
		if err = oc.ReadHeader(); err != nil {
			t.Errorf("could not read header in place: %v", err)
		}

		// check the header
		h := oc.Header()
		assertMockHeader(t, h)
	})
}

func Test__SeekValues(t *testing.T) {
	var err error
	testWithCodec(t, func(t *testing.T, oc *obfCodec) {
		// seek back to the values
		if err = oc.SeekValues(); err != nil {
			t.Errorf("could not seek to the values")
		}

		// check that the values are expected
		v, ts, err := oc.ReadParallelBlock()
		if err != nil {
			t.Errorf("could not read parallel block")
		}

		// assert timestamp
		if ts != 0 {
			t.Errorf("unexpected timestamp: %d", ts)
		}

		// assert values
		for c, value := range v {
			if value != float64(c) {
				t.Fatal()
			}
		}
	})
}

func Test__SeekSequential(t *testing.T) {
	var err error
	testWithCodec(t, func(t *testing.T, oc *obfCodec) {
		// seek back to the values
		if err = oc.SeekSequential(); err != nil {
			t.Fatalf("could not seek to the values")
		}

		channels, ts, err := oc.ReadSequential()
		if err != nil {
			t.Fatalf("could not read sequential")
		}

		for _, channel := range channels {
			for i, v := range channel {
				if v != float64(i) {
					t.Fatalf("wrong channel value")
				}
			}
		}

		for i, ts64 := range ts {
			if ts64 != int64(i) {
				t.Fatalf("wrong timestamp value")
			}
		}
	})
}

func Test__Seeking(t *testing.T) {
	testWithCodec(t, func(t *testing.T, oc *obfCodec) {
		assertNoErrors(t,
			func() error {
				return oc.SeekHeader()
			},
			func() error {
				return oc.SeekValues()
			},
			func() error {
				return oc.SeekSequential()
			},
			func() error {
				return oc.SeekParallel()
			},
			func() error {
				return oc.SeekHeader()
			},
		)
	})
}

func assertNoErrors(t *testing.T, fs ...func() error) {
	for i, f := range fs {
		if err := f(); err != nil {
			t.Fatalf("the %d-th function failed", i)
		}
	}
}

func Test__MarshallUnmarshall(t *testing.T) {
	const fn = "../etc/1fabece1-7a57-96ab-3de9-71da8446c52c"
	file, err := os.Open(fn)
	if err != nil {
		t.Fatalf("could not open test file: %v", fn)
	}
	defer file.Close()
	//oc := newObfCodec(file)

	// TODO TODO TODO
}
