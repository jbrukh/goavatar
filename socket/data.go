//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package socket

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	. "github.com/jbrukh/goavatar/datastruct"
	. "github.com/jbrukh/goavatar/device"
	"log"
)

func sendData(dataConn *websocket.Conn, s *SocketSession) {
	msg := new(ConnectResponse)
	msg.MessageType = "connect"
	msg.Success = false
	// we connect and begin to stream
	if !s.device.Engaged() {
		err := s.device.Engage()
		if err != nil {
			log.Printf("could not connect: %v", err)
			msg.Err = fmt.Sprintf("could not connect to the device")
			msg.Status = "disarmed"
			Send(s.conn, msg)
			return
		}
		msg.Success = true
		msg.Status = "streaming"
		Send(s.conn, msg)
		stream(dataConn, s)
	} else {
		log.Printf("WARNING: device was already operating")
		return
	}
}

func stream(dataConn *websocket.Conn, s *SocketSession) {
	// logging
	name := s.device.Name()
	log.Printf("%s: STREAMING ON", name)
	defer log.Printf("%s: STREAMING OFF", name)

	// device stuff
	defer s.device.Disengage()
	out, err := s.device.Subscribe("datasocket")
	if err != nil {
		log.Printf("could not subscribe to device: %s", err)
		return
	}

	// diagnose the situation
	df, ok := <-out
	if !ok {
		return
	}
	var (
		channels   = df.Buffer().Channels()
		sampleRate = df.SampleRate()
	)

	// check the parameters
	if s.pps < 1 || s.pps > sampleRate {
		s.pps = DefaultPps
		log.Printf("WARNING: setting default PPS")
	}
	if s.batchSize > s.pps || s.batchSize < 1 {
		s.batchSize = DefaultBatchSize
		log.Printf("WARNING: setting default batchSize")
	}

	streamLoop(dataConn, s, channels, sampleRate, out)
}

func streamLoop(dataConn *websocket.Conn, s *SocketSession, channels, sampleRate int, out <-chan DataFrame) {
	var (
		kill         = make(chan bool)
		pluckRate    = sampleRate / s.pps      // now we need to sample every sampleRate/pps points
		absBatchSize = s.batchSize * pluckRate // actual number of data points we must read in order to obtain a sampled batch of batchSize
		b            = NewBlockBuffer(channels, sampleRate*s.batchSize*10)

		shouldReturn = func() bool {
			select {
			case <-kill:
				return true
			default:
			}
			return false
		}
	)

	b.PluckRate(pluckRate)

	for {
		// break if there was an error sending
		// a message over the socket
		var (
			df DataFrame
			ok bool
		)

		if df, ok = <-out; !ok || shouldReturn() {
			return
		}

		// put the frame into our memory buffer
		b.Append(df.Buffer())

		// while there are batches, return them
		for b.Samples() > absBatchSize {
			if shouldReturn() {
				return
			}

			var (
				batch = b.PopDownSample(absBatchSize)
				msg   = new(DataMessage)
			)

			msg.Data, _ = batch.Arrays()
			if *verboseSocket {
				log.Printf("sending data msg: %+v", msg)
			}
			err := websocket.JSON.Send(dataConn, msg)
			if err != nil {
				log.Printf("error sending data msg: %v\n", err)
				kill <- true
			}
		}
	}
}
