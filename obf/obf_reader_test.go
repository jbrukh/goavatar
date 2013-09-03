//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package obf

import (
	//"fmt"
	"io"
	"os"
	"testing"
)

const testFile1 = "../etc/1fabece1-7a57-96ab-3de9-71da8446c52c"
const testFile2 = "../etc/364a47d2-053d-d52f-3b34-85f1a82f714e"

func obfData(file string) (io.Reader, error) {
	return os.Open(file)
}

func TestObfReader__New(t *testing.T) {
	r, err := obfData(testFile1)
	if err != nil {
		t.Errorf("could not get mock data: %v", err)
	}

	re, err := NewObfReader(r)
	if err != nil {
		t.Errorf("could not init reader: %v", err)
	}

	h := re.Header()
	channels, samples := h.Dim()
	if channels != 2 || samples != 3024 {
		t.Errorf("unexpected dimensions")
	}

	if h.DataType != 1 || h.FormatVersion != 2 || h.StorageMode != 1 ||
		h.SampleRate != 250 || h.Endianness != 0 || h.IndexUnit != 0 {
		t.Errorf("unexpected header")
	}
}

func TestObfReader__Parallel(t *testing.T) {
	r, err := obfData(testFile1)
	if err != nil {
		t.Errorf("could not get mock data: %v", err)
	}

	re, err := NewObfReader(r)
	if err != nil {
		t.Errorf("could not init reader: %v", err)
	}

	b, err := re.Parallel()
	if err != nil {
		t.Errorf("could not deserialize: %v", err)
	}

	if b.Channels() != 2 || b.Samples() != 3024 {
		t.Errorf("unexpected dimensions")
	}

	v, inx := b.Sample(0)
	if inx != 0 || len(v) != 2 {
		t.Errorf("unexpected first sample")
	}

	v, inx = b.Sample(1)
	if inx != 4000000 || len(v) != 2 {
		t.Errorf("unexpected second sample")
	}
}

func TestObfReader__Sequential(t *testing.T) {
	r, err := obfData(testFile2)
	if err != nil {
		t.Errorf("could not get mock data: %v", err)
	}

	re, err := NewObfReader(r)
	if err != nil {
		t.Errorf("could not init reader: %v", err)
	}

	h := re.Header()
	if h.Channels != 1 || h.Samples != 10706 || h.DataType != 1 || h.FormatVersion != 2 || h.StorageMode != 3 ||
		h.SampleRate != 512 || h.Endianness != 0 || h.IndexUnit != 0 {
		t.Errorf("unexpected header")
	}

	v, inxs, err := re.Sequential()
	if err != nil {
		t.Errorf("could not deserialize sequential")
	}

	if len(v) != 1 || len(inxs) != 10706 {
		t.Errorf("unexpected dimensions")
	}

	if inxs[0] != 0 || inxs[1] != 1000000 {
		t.Errorf("unexpected index values")
	}
}
