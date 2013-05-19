//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package socket

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"fmt"
	. "github.com/jbrukh/goavatar"
	"path/filepath"
)

//---------------------------------------------------------//
// Constants
//-------------------------------------------------------

// SocketSession encapsulates all of the
// business logic of sending and receiving
// control messages.
type SocketSession struct {
	conn      *websocket.Conn
	device    Device
	sessionId string
	pps       int
	batchSize int
	kickoff   chan *SocketSession
}

func (s *SocketSession) Process(msgBytes []byte, msgBase Message) {
	// message types
	msgType := msgBase.MessageType
	switch msgType {

	case "info":
		s.ProcessInfoMessage(msgBytes, msgBase.Id)

	case "connect":
		s.ProcessConnectMessage(msgBytes, msgBase.Id)

	case "record":
		s.ProcessRecordMessage(msgBytes, msgBase.Id)

	case "upload":
		s.ProcessUploadMessage(msgBytes, msgBase.Id)

	default:
		errStr := fmt.Sprintf("unknown message type: '%s'", msgType)
		SendError(s.conn, msgBase.Id, errStr)
	}
}

func (s *SocketSession) ProcessInfoMessage(msgBytes []byte, id string) {
	var msg InfoMessage
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		SendError(s.conn, id, err.Error())
	}

	r := new(InfoResponse)
	r.MessageType = "info"
	r.Id = msg.Id
	r.Success = true
	r.Version = Version()
	r.DeviceName = s.device.Name()

	Send(s.conn, r)
}

func (s *SocketSession) ProcessConnectMessage(msgBytes []byte, id string) {
	var msg ConnectMessage
	if err := json.Unmarshal(msgBytes, &msg); err != nil {
		SendError(s.conn, id, err.Error())
	}

	// start building the response
	r := new(ConnectResponse)
	r.MessageType = "connect"
	r.Id = msg.Id
	r.Success = false
	defer Send(s.conn, r)

	// should we disconnect?
	if !msg.Connect {
		err := s.device.Disengage()
		if err != nil {
			r.Status = "error"
			r.Err = err.Error()
		} else {
			r.Success = true
			r.Status = "disarmed"
			// also, disarm the device
			select {
			case <-s.kickoff:
			default:
			}
		}
		return
	}

	// should we connect?
	if msg.Connect {

		// are the parameters sane?
		if msg.Pps < 1 || msg.Pps > 250 {
			r.Err = "pps should be between 1 and 250"
			return
		}

		if msg.BatchSize > msg.Pps {
			r.Err = "batchSize should not exceed pps"
			return
		}

		// maybe someone is already using it
		if s.device.Engaged() {
			r.Status = "busy"
			r.Err = "device is already connected"
			return
		}

		// ok, now we can tell the data endpoint
		// to stream when it is connected

		select {
		case s.kickoff <- s: // send self on the kickoff channel
			// set the parameters; WARNING: since the
			// device is already armed at this point, users
			// should wait for our ConnectResponse before
			// attempting to connect to the data endpoint
			s.pps = msg.Pps
			s.batchSize = msg.BatchSize

			// device can accept a value, meaning
			// no one request for connection is in
			// an "armed" state, so we have succeeded
			r.Success = true
			r.Status = "armed"
			return

		default:
			// the kickoff channel is blocked, so some
			// other request has armed the device for
			// streaming
			r.Status = "armed"
			r.Err = "device is already armed"
			return
		}
	}
}

func (s *SocketSession) ProcessRecordMessage(msgBytes []byte, id string) {
	var msg RecordMessage
	var err error
	if err = json.Unmarshal(msgBytes, &msg); err != nil {
		SendError(s.conn, id, err.Error())
	}

	r := new(RecordResponse)
	r.MessageType = "record"
	r.Id = msg.Id
	r.Success = false
	defer Send(s.conn, r)

	if !s.device.Engaged() {
		r.Err = "device is not streaming"
		return
	}

	if !msg.Record {
		outFile, err := s.device.Stop()
		if err == nil {
			r.Success = true
			r.ResourceId = outFile
		}
		return
	}

	if msg.Record {
		if s.device.Recording() {
			r.Err = "already recording"
			return
		}

		err = s.device.Record()
		if err != nil {
			r.Err = err.Error()
			return
		}
		r.Success = true
	}
}

func (s *SocketSession) ProcessUploadMessage(msgBytes []byte, id string) {
	var msg UploadMessage
	var err error
	if err = json.Unmarshal(msgBytes, &msg); err != nil {
		SendError(s.conn, id, err.Error())
	}

	r := new(UploadResponse)
	r.MessageType = "upload"
	r.Id = msg.Id
	r.Success = false
	defer Send(s.conn, r)

	// perform the upload
	var (
		resId    = msg.ResourceId
		token    = msg.Token
		endpoint = msg.Endpoint
		file     = filepath.Join(s.device.Repo(), resId)
	)

	err = UploadOBFFile(s.device.Name(), s.sessionId, file, endpoint, token)
	if err != nil {
		r.Err = err.Error()
		return
	}
	r.Success = true

}
