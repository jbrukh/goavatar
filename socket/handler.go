//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package socket

import (
	"bytes"
	"code.google.com/p/go.net/websocket"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	. "github.com/jbrukh/goavatar"
	. "github.com/jbrukh/goavatar/device"
	. "github.com/jbrukh/goavatar/obf/recorder"
	. "github.com/jbrukh/goavatar/util"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// --------------------------------------------------------- //
// Endpoints
// --------------------------------------------------------- //

const (
	DefaultControlEndpoint = "/control"
	DefaultDataEndpoint    = "/device"
	DefaultListenPort      = 8000
)

var (
	controlEndpoint *string = flag.String("controlEndpoint", DefaultControlEndpoint, "the websocket url for control messages")
	dataEndpoint    *string = flag.String("dataEndpoint", DefaultDataEndpoint, "the websocket url for data messages")
	listenPort      *int    = flag.Int("listenPort", DefaultListenPort, "the websocket port on which to listen")
	verboseSocket   *bool   = flag.Bool("verboseSocket", false, "the websocket is verbose")
)

// --------------------------------------------------------- //
// Constants
// --------------------------------------------------------- //

const (
	DefaultPps       = 125
	DefaultBatchSize = 25
)

// The OctopusSocket.
type OctopusSocket struct {
	pps       int                 // points per second to deliver
	batchSize int                 // points per batch to deliver (hence frequency = pps/batchSize)
	kickoff   chan *SocketSession // blocker channel for the data handler

	device Device // device to serve
	conn   *websocket.Conn
}

func NewOctopusSocket(device Device) *OctopusSocket {
	return &OctopusSocket{
		device:  device,
		kickoff: make(chan *SocketSession, 1),
	}
}

func (s *OctopusSocket) ListenAndServe() {
	var (
		port      = fmt.Sprintf(":%d", *listenPort)
		wsControl = websocket.Handler(s.handleControlConn)
		wsData    = websocket.Handler(s.handleDataConn)
	)

	absRepo, err := filepath.Abs(s.device.Repo().Basedir())
	if err != nil {
		absRepo = s.device.Repo().Basedir()
	}

	fmt.Printf(`Octopus Connector
Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.

`)

	fmt.Printf("Version:  %s\n", Version())
	fmt.Printf("Device:   %v\n", s.device.Name())
	fmt.Printf("Control:  http://localhost:%d%s\n", *listenPort, *controlEndpoint)
	fmt.Printf("Data:     http://localhost:%d%s\n", *listenPort, *dataEndpoint)
	fmt.Printf("Repo:     %v\n\n", absRepo)

	// ensure the repository exists
	if err := os.MkdirAll(absRepo, 0755); err != nil {
		log.Fatalf("could not create the device repo: %s", absRepo)
	}

	http.Handle(*controlEndpoint, wsControl)
	http.Handle(*dataEndpoint, wsData)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("could not start OctopusSocket: %v", err)
	}
}

// --------------------------------------------------------- //
// Control Handler
// --------------------------------------------------------- //

func (s *OctopusSocket) handleControlConn(conn *websocket.Conn) {
	defer conn.Close()
	defer s.device.Disengage() // TODO: this will kill the device on multiple conns

	uuid, _ := Uuid()
	session := &SocketSession{
		conn:      conn,
		pairingId: uuid,
		device:    s.device, // in the future we can instantiate device based on message
		pps:       s.pps,
		batchSize: s.batchSize,
		kickoff:   s.kickoff,
		recorder:  NewDeviceRecorder(s.device, NewObfRecorder(s.device.Repo())),
	}
	log.Printf("got session: %+v", session)
	// keep processing as long as we are connected
	for {
		msgBytes, msgBase, err := Receive(conn)
		if err != nil {
			if err == io.EOF || err.Error() == "EOF" {
				break
			}
			continue
		}
		log.Printf("Octopus Socket: RECEIVED %s", msgBytes)
		session.Process(msgBytes, msgBase)
	}
}

// --------------------------------------------------------- //
// Data Handler
// --------------------------------------------------------- //

func (s *OctopusSocket) handleDataConn(conn *websocket.Conn) {
	defer conn.Close()
	// gate to see if it is armed
	select {
	case session := <-s.kickoff:
		sendData(conn, session)
	default:
		// kickoff is blocked, meaning no one has
		// armed the device; we close the socket
		return
	}
}

// --------------------------------------------------------- //
// Socket Helpers
// --------------------------------------------------------- //

// Receive receives control messages. If there is
// a problem with the connection, or the message
// you send has a bad "header", then an err is
// reported.
func Receive(conn *websocket.Conn) (msgBytes []byte, msgBase Message, err error) {
	// get the raw bytes
	err = websocket.Message.Receive(conn, &msgBytes)
	if err != nil {
		log.Printf("websocket says: %v", err)
		return
	}

	// get the type
	err = json.Unmarshal(msgBytes, &msgBase)
	if err != nil {
		SendError(conn, msgBase.Id, "error getting message type")
	}
	return
}

// Send an arbitrary message on the connection.
func Send(conn *websocket.Conn, r interface{}) {
	log.Printf("Octopus Socket: RESPONDED %+v", r)
	// send it off
	err := websocket.JSON.Send(conn, r)
	if err != nil {
		log.Printf("error sending: %v\n", err)
	}
}

// Send a file as binary over the socket.
func SendFile(conn *websocket.Conn, filename string, correlationId int32) (err error) {
	log.Printf("Octopus Socket: SENDING FILE %s", filename)
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("Could not read file %s: %v", filename, err)
		data = []byte{0}
	}

	// put in the correlation id
	id := new(bytes.Buffer)

	// write the id
	if err = binary.Write(id, binary.BigEndian, correlationId); err != nil {
		return err
	}

	// combine with data
	stampedData := append(id.Bytes(), data...)

	err = websocket.Message.Send(conn, stampedData)
	if err != nil {
		log.Printf("error sending: %v\n", err)
	}
	return
}

// Send an error response on the connection.
func SendError(conn *websocket.Conn, id string, errStr string) {
	// create the error message
	r := new(Response)
	r.MessageType = "error"
	r.Id = id
	r.Success = false
	r.Err = errStr

	// log
	log.Printf("sending error response: %+v", r)

	// send it off
	err := websocket.JSON.Send(conn, r)
	if err != nil {
		log.Printf("error sending: %v\n", err)
	}
	return
}
