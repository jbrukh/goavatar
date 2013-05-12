package goavatar

import (
	"bytes"
	"io"
	//"log"
)

const (
	BlockBufferValueSize     = 8
	BlockBufferTimestampSize = 8
)

type BlockBuffer struct {
	channels  int // number of channels per sample
	parity    int // plucking offset
	pluckRate int // downsampling, or plucking rate
	blockSize int // fixed size of blocks

	buf *bytes.Buffer // data
}

// Create a new BlockBuffer anticipating the given
// number of channels and the given sample size.
func NewBlockBuffer(channels, size int) *BlockBuffer {
	blockSize := channels*BlockBufferValueSize + 1*BlockBufferTimestampSize
	if channels < 0 || size < 1 {
		panic("bad parameters")
	}
	return &BlockBuffer{
		channels:  channels,
		pluckRate: 1,
		blockSize: blockSize,
		buf:       bytes.NewBuffer(make([]byte, 0, size*blockSize)),
	}
}

// Set the downsampling rate. This means that down
// sampling will select 1 out of k samples.
func (b *BlockBuffer) PluckRate(k int) {
	b.pluckRate = k
}

// Append a data from a BlockBuffer to the existing BlockBuffer,
// ignoring the latter's pluck rate. The BlockBuffers must be
// comparable in the sense of channels.
func (b *BlockBuffer) Append(bb *BlockBuffer) {
	if bb.channels != b.channels {
		panic("not comparable")
	}
	b.appendBlocks(bb.buf)
}

func (b *BlockBuffer) AppendBlock(v []float64, ts int64) {
	if len(v) != b.channels {
		panic("not comparable")
	}
	binary.Write(b.buf, binary.BigEndian, v)
	binary.Write(b.buf, binary.BigEndian, ts)
}

// The number of samples in this BlockBuffer.
func (b *BlockBuffer) Size() int {
	return b.buf.Len() / b.blockSize
}

// Get the next n samples and downsamples them according
// to the downsampling rate.
func (b *BlockBuffer) DownSample(n int) (bb *BlockBuffer) {
	bb = NewBlockBuffer(b.channels, n/b.pluckRate+1)
	blockSize := b.blockSize
	for n > 0 {
		if b.parity == 0 {
			_, err := io.CopyN(bb.buf, b.buf, int64(blockSize))
			if err != nil {
				return
			}
		} else {
			if block := b.buf.Next(blockSize); len(block) == 0 {
				return
			}
		}
		n--
		// increment the parity
		b.parity = (b.parity + 1) % b.pluckRate
	}
	return
}

func (b *BlockBuffer) appendBlocks(buf *bytes.Buffer) {
	io.Copy(b.buf, buf)
}
