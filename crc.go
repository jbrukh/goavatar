//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package goavatar

// --------------------------------------------------------------------//
// CRC Writer -- for calculating CRC-16-CCIT, according to Avatar Spec
// ------------------------------------------------------------------- //

// CrcWriter is a summary data structure
// which continuously calculates the CRC
// or bytes you put in it.
type CrcWriter struct {
	crc uint16
}

// Crc returns the current CRC value.
func (w *CrcWriter) Crc() uint16 {
	return w.crc
}

// Reset the CRC calculation.
func (w *CrcWriter) Reset() {
	w.crc = uint16(0)
}

// Write a series of bytes to the CrcWriter.
func (w *CrcWriter) Write(p []byte) (n int, err error) {
	for _, b := range p {
		w.WriteByte(b)
	}
	return len(p), nil
}

// Write a byte to the CrcWriter.
func (w *CrcWriter) WriteByte(b byte) {
	w.crc = (w.crc >> 8) | ((w.crc & 0xFF) << 8)
	w.crc ^= uint16(b)
	w.crc ^= (w.crc & 0xFF) >> 4
	w.crc ^= (w.crc << 12) & 0xFFFF
	w.crc ^= (w.crc & 0xFF) << 5
}
