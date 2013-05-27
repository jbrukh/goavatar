//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package socket

import (
	"code.google.com/p/go.net/websocket"
	"encoding/json"
	"fmt"
	. "github.com/jbrukh/goavatar"
	. "github.com/jbrukh/goavatar/device"
	"log"
	"os"
	"path/filepath"
	"strings"
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
	recorder  *DeviceRecorder
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

	case "repository":
		s.ProcessRepositoryMessage(msgBytes, msgBase.Id)

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
	shouldRespond := true

	// by default, send the response
	defer func() {
		if shouldRespond {
			Send(s.conn, r)
		}
	}()

	if !s.device.Engaged() {
		r.Err = "device is not streaming"
		return
	}

	if msg.Record {
		if s.recorder.Recording() {
			r.Err = "already recording"
			return
		}

		// if this is a fixed-time session,
		// then wait for the recording to stop
		if msg.Seconds > 0 {
			// calculate how many data points we need
			points := msg.Seconds * s.device.Info().SampleRate
			log.Printf("FIXED TIME RECORDING: %d seconds, %d points", msg.Seconds, points)

			s.recorder.SetMax(points)
			go func() {
				ar := new(RecordResponse)
				ar.MessageType = "record"
				ar.Id = msg.Id
				ar.Success = false
				ar.Seconds = msg.Seconds
				outFile, err := s.recorder.Wait()
				if err != nil {
					log.Printf("error during fixed-time recording: %v", err)
					ar.Err = err.Error()
				}
				ar.Success = true
				ar.ResourceId = outFile
				Send(s.conn, ar)
			}()
		}

		err = s.recorder.RecordAsync()
		if err != nil {
			r.Err = err.Error()
			return
		}
		r.Success = true

	} else if !msg.Record {
		if s.recorder.RecordingTimed() {
			s.recorder.Release()
			// don't send a response in this case
			shouldRespond = false
		} else {
			outFile, err := s.recorder.Stop()
			if err == nil {
				r.Success = true
				r.ResourceId = outFile
			}
		}
		return
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

	if msg.Clear {
		// best-effort removal, will still return
		// Success: true even if removal fails
		if err := os.Remove(file); err != nil {
			r.Err = err.Error()
		}
	}
	r.Success = true
}

func (s *SocketSession) ProcessRepositoryMessage(msgBytes []byte, id string) {
	var (
		msg RepositoryMessage
		err error
	)
	if err = json.Unmarshal(msgBytes, &msg); err != nil {
		SendError(s.conn, id, err.Error())
	}

	r := new(RepositoryResponse)
	r.MessageType = "repository"
	r.Id = msg.Id
	r.Success = false
	defer Send(s.conn, r)

	switch msg.Operation {
	case "list":
		if infos, err := listFiles(s.device.Repo()); err != nil {
			r.Err = err.Error()
			return
		} else {
			r.ResourceInfos = infos
			r.Success = true
			return
		}
	case "clear":
		if err := removeFiles(s.device.Repo()); err != nil {
			r.Err = err.Error()
			return
		} else {
			r.Success = true
			return
		}
	default:
		r.Err = fmt.Sprintf("unknown operation: %s", msg.Operation)
	}
}

func listFiles(repo string) ([]*ResourceInfo, error) {
	infos := make([]*ResourceInfo, 0)
	err := filepath.Walk(repo, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		base := filepath.Base(path)
		if !f.IsDir() && !strings.HasPrefix(base, ".") {
			log.Printf("LIST\t%s", path)
			infos = append(infos, &ResourceInfo{
				ResourceId:   base,
				File:         path,
				SizeBytes:    f.Size(),
				LastModified: f.ModTime().Unix(),
			})
		}
		return nil
	})
	return infos, err
}

func removeFiles(repo string) error {
	return filepath.Walk(repo, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !f.IsDir() && !strings.HasPrefix(filepath.Base(path), ".") {
			if err := os.RemoveAll(path); err != nil {
				log.Printf("could not remove file: %v", err)
			}
			log.Printf("DELETE\t%s", path)
		}
		return nil
	})
}
