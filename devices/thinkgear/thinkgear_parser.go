package thinkgear

import (
	"bufio"
	"fmt"
	. "github.com/jbrukh/goavatar"
	"io"
	"log"
	"time"
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
syncUp:
	// sync up
	for {
		if p.next() != SYNC || p.next() != SYNC {
			continue
		}
		break
	}
	var plen byte // payload length

syncLength: // using a label makes code 2 lines shorter :)
	plen = p.next()
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
	return parseRawPayload(payload)
}

// parseRawPayload will parse the payload buffer for
// raw signal only, and deliver that signal on the
// given channel
func parseRawPayload(payload []byte) (df DataFrame, err error) {
	// check if there is not enough payload
	// or if the raw code is missing, or if
	// the raw value does not have two bytes
	if len(payload) < 4 || payload[0] != CODE_RAW_VALUE || payload[1] != 2 {
		return nil, fmt.Errorf("bad data")
	}

	b := NewBlockBuffer(1, 1)
	v := float64(int16(payload[2])<<8 | int16(payload[3]))
	b.AppendSample(
		[]float64{v},
		time.Now().UnixNano(),
	)
	df = NewDataFrame(b, SampleRate)
	return
}
