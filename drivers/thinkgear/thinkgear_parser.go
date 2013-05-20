package thinkgear

import (
	"bufio"
	. "github.com/jbrukh/goavatar/datastruct"
	. "github.com/jbrukh/goavatar/device"
	"io"
	"log"
)

// ----------------------------------------------------------------- //
// Constants
// ----------------------------------------------------------------- //

// Approx the number of data points to be
// coming in from the device per second
const SampleRate = 512

// MaxPayloadLength is the maximum number of
// bytes that can be contained in the payload
// message, not including SYNC, PLENGTH and
// CHECKSUM bytes.
const MaxPayloadLength = 169

const SamplePeriod = 1953125 // 1/512 (in nanos)

// protocol symbols
const (
	SYNC           = 0xAA
	EXCODE         = 0x55
	CODE_RAW_VALUE = 0x80 // 128
)

// ----------------------------------------------------------------- //
// ThinkGear Stream Parser
// ----------------------------------------------------------------- //

type thinkGearParser struct {
	reader *bufio.Reader // reader of the stream
	ts     int64
}

// create a new parser
func NewThinkGearParser(reader io.ReadCloser) *thinkGearParser {
	br := bufio.NewReader(reader)
	return &thinkGearParser{
		reader: br,
	}
}

func (p *thinkGearParser) next() (b byte) {
	var err error
	if b, err = p.reader.ReadByte(); err != nil {
		log.Printf("error reading stream: %v", err)
		panic(err)
	}
	return
}

func (p *thinkGearParser) ParseRaw() (df DataFrame, err error) {

	// sync up with the stream
syncUp:
	if p.next() != SYNC || p.next() != SYNC {
		goto syncUp
	}

syncLength: // using a label makes code 2 lines shorter :)
	plen := p.next()
	if plen == SYNC {
		goto syncLength
	}
	if plen > MaxPayloadLength {
		goto syncUp
	}

	// read the entire payload
	var (
		payload  = make([]byte, 0, plen)
		count    = int(plen)
		checksum byte
	)

	// populate the payload slice
	// TODO: use CopyN or something
	for count > 0 {
		b := p.next()
		payload = append(payload, b)
		checksum += b
		count--
	}

	// and check it
	checksum = 0xFF &^ checksum

	stated := p.next()
	if checksum != stated {
		log.Printf("checksum has failed: expected %v but got %v", checksum, stated)
		goto syncUp
	}

	// our frame must begin with CODE_RAW_VALUE or skip
	if len(payload) < 4 || payload[0] != CODE_RAW_VALUE {
		goto syncUp
	}
	if payload[1] != 2 {
		// if the number of bytes in the value is not 2,
		// we don't know how to read this yet; panic
		panic("expecting two bytes in the raw value")
	}

	b := NewBlockBuffer(1, 1)
	v := float64(int16(payload[2])<<8 | int16(payload[3]))
	b.AppendSample(
		[]float64{v},
		p.ts,
	)
	p.ts += SamplePeriod
	df = NewDataFrame(b, SampleRate)

	return
}
