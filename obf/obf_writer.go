//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package obf

import (
	"io"
)

type (
	// obfCodec will read and write the OBF
	// format on various levels of abstraction.
	obfWriter struct {
		file        io.Writer
		payloadSize int64
	}
)

// NewObfReader creates a vanilla OBF deserializer that reads
// the OBF stream sequentially.
func NewObfWriter(r io.Writer) (or ObfWriter, err error) {
	return
}
