//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package avatar

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	. "github.com/jbrukh/goavatar/datastruct"
	. "github.com/jbrukh/goavatar/util"
	"io"
	"time"
)

// These values are valid with the firmware version
// on the device that we have acquired. In the future
// they may change as the format is updated.
const (
	AvatarSyncByte          = 0xAA
	AvatarExpectedVersion   = 3                   // version
	AvatarExpectedFrameType = 1                   // data frame
	AvatarFracSecs          = time.Duration(4096) // fractional second parts
	AvatarMaxFramesSize     = 454                 //  22 + 3*9*16 = 454 (including trigger channel)
	AvatarAdcRange          = 16777216            // 2^24
	AvatarPointSize         = 3
	AvatarExpectedSamples   = 16
	AvatarSanePayload       = 8 * AvatarExpectedSamples * AvatarPointSize
	AvatarMaxChannels       = 8
	AvatarHeaderSize        = 19 // not including sync byte
)

// The possible sample rates that the Avatar
// currently supports.
var AvatarSampleRates = []int{250, 500, 1000}

// ----------------------------------------------------------------- //
// AvatarEEG stream parser
// ----------------------------------------------------------------- //

// a parser/crc calculator for the stream
type avatarParser struct {
	reader *bufio.Reader // reader of the stream
	crc    CrcWriter
}

// create a new parser
func NewAvatarParser(reader io.ReadCloser) *avatarParser {
	return &avatarParser{
		reader: bufio.NewReader(reader),
	}
}

// TODO: we know the size of the entire frame, so we can peek at it
// and check the CRC; also, the frame has a maximum size...
//
// "With the version of firmware you have there will always be 16 samples in
// a frame and if trigger channel enabled this would be 9 channels. So max
// frame size with your firmware is 22 + 3*9*16 = 454 bytes.
//
// With future versions we may adjust number of samples to optimize
// Bluetooth performance and may have hardware that supports up to 24
// channels."
func (r *avatarParser) ParseFrame() (dataFrame *AvatarDataFrame, err error) {
	// reset the crc calculation
	r.crc.Reset()

	// sync up with the stream, reading up
	// until the sync up value
	_, err = r.reader.ReadBytes(AvatarSyncByte)
	if err != nil {
		return nil, err
	}

	timeReceived := time.Now()

	// note the sync byte
	r.crc.WriteByte(AvatarSyncByte)

	// at this point, we will peek ahead
	// just 3 bytes which is enough to read the frame size
	three, err := r.reader.Peek(3)
	if err != nil {
		return nil, err
	}

	// check the frame size; using bit shifting for efficiency;
	// this allows us to determine early whether the frame is
	// good without consuming the reader and possibly skipping
	// sync bytes if there is corruption
	frameSize := int(uint16(three[1])<<8 | uint16(three[2]))
	if frameSize > AvatarMaxFramesSize {
		return nil, fmt.Errorf("this frame is over max frame size: %d", frameSize)
	}

	// now that we know the frame size, we can read the
	// whole frame and check the CRC; the frame size
	// includes the sync byte and the CRC
	frame, err := r.reader.Peek(frameSize - 1) // frame minus sync
	if err != nil {
		return nil, err
	}

	// the stated CRC
	l := len(frame)
	crc := uint16(frame[l-2])<<8 | uint16(frame[l-1])

	// the calculated CRC
	r.crc.Write(frame[:l-2])
	ourCrc := r.crc.Crc()

	// check the crc
	if crc != ourCrc {
		return nil, fmt.Errorf("crc doesn't match: expected %d but calculated %d", crc, ourCrc)
	}

	// everything is okay, now
	// we actually read the frame
	_, err = io.ReadFull(r.reader, frame) // careful, overwriting data and making assumptions
	if err != nil {
		return nil, err
	}

	header := new(AvatarHeader)
	buf := bytes.NewBuffer(frame[:AvatarHeaderSize])
	err = binary.Read(buf, binary.BigEndian, header)
	if err != nil {
		return nil, err
	}

	// get the size of the payload
	pSize := header.PayloadSize()

	// do a sanity check
	expFrameSize := 1 + AvatarHeaderSize + pSize + 2
	if expFrameSize != frameSize {
		return nil, fmt.Errorf("frameSize didn't jive, expected: %d got: %d", expFrameSize, frameSize)
	}

	// get the payload
	payload := frame[AvatarHeaderSize : l-2] // excluding header and crc

	// allocate the slices for the data
	var (
		samples    = header.Samples()
		channels   = header.Channels()
		hasTrigger = header.HasTriggerChannel()
		δ          = time.Second / time.Duration(header.SampleRate())
	)

	data := NewBlockBuffer(channels, samples)

	// write the samples in blocks
	for j := 0; j < samples; j++ {
		if hasTrigger {
			// skip the trigger channel
			payload = payload[AvatarPointSize:]
		}
		p := make([]float64, channels)
		for c := range p {
			p[c] = consumeDataPoint(payload, float64(header.VoltRange()))
			payload = payload[AvatarPointSize:]
		}

		// put the block into the buffer
		ts := InterpolateTs(header.Generated().UnixNano(), j, δ)
		data.AppendSample(p, ts)
	}

	dataFrame = &AvatarDataFrame{
		AvatarHeader: *header,
		data:         data,
		received:     timeReceived,
		crc:          crc,
	}
	return
}

func consumeDataPoint(payload []byte, voltRange float64) float64 {
	raw := uint32(payload[0])<<16 | uint32(payload[1])<<8 | uint32(payload[2])
	return ((float64(raw) / float64(1000) / float64(AvatarAdcRange)) * voltRange)
}
