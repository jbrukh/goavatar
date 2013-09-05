//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package obf

import (
	"io"
	"testing"
)

// ----------------------------------------------------------------- //
// ObfHeader
// ----------------------------------------------------------------- //

// Test the Dim() function of the ObfHeader.
func TestObfHeader__Dim(t *testing.T) {
	h := &ObfHeader{
		Channels: 10,
		Samples:  1024,
	}
	if ch, s := h.Dim(); ch != 10 || s != 1024 {
		t.Errorf("unexpected dimension")
	}
}

// ----------------------------------------------------------------- //
// Read Ops
// ----------------------------------------------------------------- //

func TestObf__ReadHeader(t *testing.T) {
	r, err := obfData(testFile1)
	if err != nil {
		t.Errorf("could not get mock data: %v", err)
	}

	h, err := ReadHeader(r)
	if err != nil {
		t.Errorf("could not read header: %v", err)
	}

	if ch, s := h.Dim(); ch != 2 || s != 3024 {
		t.Errorf("unexpected dimensions")
	}

	if h.DataType != 1 || h.FormatVersion != 2 || h.StorageMode != 1 ||
		h.SampleRate != 250 || h.Endianness != 0 || h.IndexUnit != 0 {
		t.Errorf("unexpected header")
	}
}

func TestObf__ReadParallel(t *testing.T) {
	r, err := obfData(testFile1)
	if err != nil {
		t.Errorf("could not get mock data: %v", err)
	}

	h, err := ReadHeader(r)
	if err != nil {
		t.Errorf("could not read header: %v", err)
	}

	b, err := ReadParallel(r, h)
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

	if _, _, err := ReadSequential(r, h); err == nil {
		t.Errorf("should have thrown, no sequential mode")
	}
}

func TestObf__ReadSequential(t *testing.T) {
	r, err := obfData(testFile2)
	if err != nil {
		t.Errorf("could not get mock data: %v", err)
	}

	h, err := ReadHeader(r)
	if err != nil {
		t.Errorf("could not read header: %v", err)
	}

	if h.Channels != 1 || h.Samples != 10706 || h.DataType != 1 || h.FormatVersion != 2 || h.StorageMode != 3 ||
		h.SampleRate != 512 || h.Endianness != 0 || h.IndexUnit != 0 {
		t.Errorf("unexpected header")
	}

	ps := getPayloadSize(h.Dim())

	// fast forward to sequential
	if _, err = io.ReadFull(r, make([]byte, ps)); err != nil {
		t.Errorf("could not fast to seq payload")
	}

	v, inxs, err := ReadSequential(r, h)
	if err != nil {
		t.Errorf("could not deserialize sequential")
	}

	if len(v) != 1 || len(inxs) != 10706 {
		t.Errorf("unexpected dimensions")
	}

	if inxs[0] != 0 || inxs[1] != 1000000 {
		t.Errorf("unexpected index values: %v", inxs)
	}
}
