package goavatar

import (
	"testing"
)

func TestCrc(t *testing.T) {
	var crc CrcWriter
	crc.WriteByte(0x00)
	if crc.Crc() != 0x0000 {
		t.Fail()
	}

	crc.Reset()
	crc.WriteByte(0x01)
	if crc.Crc() != 0x1021 {
		t.Fail()
	}

	crc.Reset()
	crc.WriteByte(0x02)
	if crc.Crc() != 0x2042 {
		t.Fail()
	}
}
