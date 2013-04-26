package goavatar

// --------------------------------------------------------------------//
// CRC Writer -- for calculating CRC-16-CCIT, according to Avatar Spec
// ------------------------------------------------------------------- //

type CrcWriter struct {
	crc uint16
}

// return the current CRC value
func (w *CrcWriter) Crc() uint16 {
	return w.crc
}

// reset the CRC calculation
func (w *CrcWriter) Reset() {
	w.crc = uint16(0)
}

// write a series of bytes to the CRC calculation
func (w *CrcWriter) Write(p []byte) (n int, err error) {
	for _, b := range p {
		w.WriteByte(b)
	}
	return len(p), nil
}

// write a byte to the CRC calculation
func (w *CrcWriter) WriteByte(b byte) {
	w.crc = (w.crc >> 8) | ((w.crc & 0xFF) << 8)
	w.crc ^= uint16(b)
	w.crc ^= (w.crc & 0xFF) >> 4
	w.crc ^= (w.crc << 12) & 0xFFFF
	w.crc ^= (w.crc & 0xFF) << 5
}
