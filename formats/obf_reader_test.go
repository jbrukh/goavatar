//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package formats

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

func TestNewObfReader(t *testing.T) {
	r, err := obfData(testFile1)
	if err != nil {
		t.Errorf("could not get mock data: %v", err)
	}

	re, err := NewObfReader(r)
	if err != nil {
		t.Errorf("could not init reader: %v", err)
	}

}
