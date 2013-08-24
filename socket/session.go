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
	. "github.com/jbrukh/goavatar/repo"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

//---------------------------------------------------------//
// Constants
//---------------------------------------------------------//

// SocketSession encapsulates all of the
// business logic of sending and receiving
// control messages.
type SocketSession struct {
	conn      *websocket.Conn
	device    Device
	pairingId string
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
	r.PairingId = s.pairingId

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
	r.PairingId = s.pairingId
	suppress := false

	// by default, send the response
	defer func() {
		if !suppress {
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
		if msg.Milliseconds > 0 {
			// calculate how many data points we need
			points := msg.Milliseconds * s.device.Info().SampleRate / 1000
			log.Printf("FIXED TIME RECORDING: %d milliseconds, %d points", msg.Milliseconds, points)

			s.recorder.SetMax(points)
			go func() {
				ar := new(RecordResponse)
				ar.MessageType = "record"
				ar.Id = msg.Id
				ar.Success = false

				info, err := s.recorder.Wait()
				if err != nil {
					log.Printf("error during fixed-time recording: %v", err)
					ar.Err = err.Error()
				} else {
					ar.Success = true
				}

				ar.ResourceId = info.ResourceId
				ar.Milliseconds = int(info.DurationMs)
				Send(s.conn, ar)
			}()
		}

		// kick off the recording, always going to
		// the local directory
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
			suppress = true
		} else {
			info, err := s.recorder.Stop()
			if err == nil {
				r.Success = true
				r.ResourceId = info.ResourceId
				r.Milliseconds = int(info.DurationMs)
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

	resourceId := msg.ResourceId
	repo := s.device.Repo()
	file, err := repo.Lookup(resourceId)
	if err != nil {
		r.Err = err.Error()
		return
	}

	// direct uploads go through the Octopus site
	// while S3 uploads go directly to S3, with the
	// Octopus site providing authentication info
	dest := msg.Destination
	if dest == "direct" {
		var (
			params   = msg.UploadParams
			token    = params["token"]
			endpoint = params["endpoint"]
		)
		err = UploadOBFFile(s.device.Name(), s.pairingId, file, endpoint, token)
		if err != nil {
			r.Err = err.Error()
			return
		}
	} else if dest == "s3" {

		// check the size of the file
		fi, err := os.Stat(file)
		if err != nil {
			r.Err = fmt.Sprintf("couldn't get file size: %v", err.Error())
			return
		}
		fileBytes := fi.Size()

		params := msg.UploadParams
		p := S3UploadParameters{
			file:           file,
			resourceId:     resourceId,
			awsAccessKeyId: params["aws_access_key_id"],
			awsBucket:      params["aws_bucket"],
			acl:            "private",
			policy:         params["policy"],
			signature:      params["signature"],
			contentType:    "application/octet-stream",
			keyMapping:     SubdirKeyMapping,
		}

		s3Key, err := UploadS3(p)
		if err != nil {
			r.Err = err.Error()
			return
		}

		// send some response fields
		r.ResponseFields = map[string]interface{}{
			"s3_key":     s3Key,
			"file_bytes": fileBytes,
		}

	} else {
		r.Err = "your 'destination' field must be one of {'s3', 'direct'}"
		return
	}

	// cache the resource id, removing it from the
	// main repository listing
	if err := repo.Cache(resourceId); err != nil {
		r.Err = err.Error()
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
	r.Operation = msg.Operation

	// uppress the sending of JSON, particularly
	// responses for "get" operations
	suppress := false
	defer func() {
		if !suppress {
			Send(s.conn, r)
		}
	}()

	// the repository we're operating on
	repo := s.device.Repo()

	switch msg.Operation {
	// list the files in the repo (always local)
	case "list":

		infos, err := repo.List()
		if err != nil {
			r.Err = err.Error()
			return
		}

		r.ResourceInfos = infos
		r.Success = true
		return

	// clear the repository
	case "clear":
		log.Printf("CLEAR REPO")
		if err := repo.Clear(); err != nil {
			r.Err = err.Error()
			return
		}
		r.Success = true
		return
	// delete a specific file
	case "delete":
		if msg.ResourceId == "" {
			r.Err = "You must specify a resourceId!"
			return
		}
		log.Printf("CACHE %v\n", msg.ResourceId)

		// cache it!
		if err := repo.Cache(msg.ResourceId); err != nil {
			r.Err = err.Error()
			return
		}
		r.Success = true
		return
	case "get":
		if msg.ResourceId == "" {
			r.Err = "You must specify a valid resource id"
			return
		}

		log.Printf("GET %v\n", msg.ResourceId)

		// look up the file
		path, err := repo.Lookup(msg.ResourceId)
		if err != nil {
			r.Err = err.Error()
			return
		}

		// send it
		if err := sendFile(s.conn, path, msg.Id); err != nil {
			r.Err = err.Error()
			return
		} else {
			r.Success = true
			// TODO: this is a temporary hack to suppress JSON
			// responses to successful "get" operations
			suppress = true // Oh my
			return
		}
	default:
		r.Err = fmt.Sprintf("unknown operation: %s", msg.Operation)
	}
}

// TODO: deprecate!
func listFiles(basedir string) ([]*ResourceInfo, error) {
	infos := make([]*ResourceInfo, 0)
	err := filepath.Walk(basedir, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		base := filepath.Base(path)
		if !f.IsDir() && !strings.HasPrefix(base, ".") {
			log.Printf("LIST\t%s", path)
			infos = append(infos, &ResourceInfo{
				Id:           base,
				File:         path,
				SizeBytes:    f.Size(),
				LastModified: f.ModTime().Unix(),
			})
		}
		return nil
	})
	return infos, err
}

func sendFile(conn *websocket.Conn, path, correlationId string) error {
	id, err := strconv.ParseInt(correlationId, 10, 32)
	if err != nil {
		return err
	}
	log.Printf("SEND\t%s", path)
	return SendFile(conn, path, int32(id))
}
