package goavatar

import (
	"testing"
)

func TestToUint16(t *testing.T) {
	bytes := []byte{0x01, 0x01}
	if toUint16(bytes) != 257 {
		t.Fail()
	}

	bytes = []byte{0xFF, 0x01}
	if toUint16(bytes) != 65281 {
		t.Fail()
	}
}

func TestToUint32(t *testing.T) {
	bytes := []byte{0xFF, 0x01, 0x10, 0xFE}
	if toUint32(bytes) != 4278259966 {
		t.Fail()
	}
}
