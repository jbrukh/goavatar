goavatar
========

Go parser and websocket server for the AvatarEEG data stream.

Installation
============

To get the repo:

    $ go get -u github.com/jbrukh/goavatar

Then, to install the Octopus server:

    $ go install github.com/jbrukh/goavatar/octopus

Make sure the $GOPATH/bin directory is in your PATH and

    $ octopus

Mock Device
===========

    $ octopus --mockDevice
    
Options
=======

    $ octopus --help
    Usage of octopus:
      -integers=false: whether to return integral data
      -listenPort=8000: the websocket port on which to listen
      -mockDevice=false: whether to use the mock device
      -serialPort="/dev/tty.AvatarEEG03009-SPPDev": the serial port for the device
      -verbose=false: whether the socket is verbose (shows outgoing data)

Protocol
========

    // Base type for messages.
    type Message struct {
        Id          string `json:"id"`           // should be non-empty
        MessageType string `json:"message_type"` // will be one of {"info", connect", "record", "error"}
    }

    // Basic information about the server.
    type InfoMessage struct {
        Id          string `json:"id"`           // should be non-empty
        MessageType string `json:"message_type"` // should be "info"
    }

    // ConnectMessage is used to connect to the device
    // and begin streaming. A ConnectResponseMessage is 
    // sent to indicate success or failure, and data
    // immediately begins to flow on the data endpoint.
    type ConnectMessage struct {
        Id          string `json:"id"`           // should be non-empty
        MessageType string `json:"message_type"` // should be "connect"
        Connect     bool   `json:"connect"`      // boolean to engage or disengage the device
        Pps         int    `json:"pps"`          // points per second, one of 250, 125, 83, ..., 250/k
        BatchSize   int    `json:"batch_size"`   // points to return per batch
    }

    // RecordMessage is used to trigger recording on
    // a device connection that is engaged. A RecordResponseMessage
    // is sent to indicate success (if recording has commenced) or
    // failure (if the device is off, or other errors).
    type RecordMessage struct {
        Id          string `json:"id"`           // should be non-empty
        MessageType string `json:"message_type"` // should be "record"
        Record      bool   `json:"record"`       // start or stop recording
        Token       string `json:"token"`        // authentication token for upload
    }

    // Base type for response messages.
    type Response struct {
        Id          string `json:"id"`           // echo of your correlation id
        MessageType string `json:"message_type"` // will be one of {"info", connect", "record", "error"}
        Success     bool   `json:"success"`      // whether or not the control message was successful
        Err         string `json:"err"`          // error text, if any
    }

    // ConnectResponseMessage is sent in response to a ConnectMessage.
    // The MessageType is set to "connect".
    type ConnectResponse struct {
        Id          string `json:"id"`           // echo of your correlation id
        MessageType string `json:"message_type"` // will be "connect"
        Success     bool   `json:"success"`      // whether or not the control message was successful
        Err         string `json:"err"`          // error text, if any
        Status      string `json:"status"`       // device status, one of {"armed", "busy", "disconnected"}
    }

    // RecordResponseMessage is sent in response to a RecordMessage.
    // The MessageType is set to "record".
    type RecordResponse struct {
        Id          string `json:"id"`           // echo of your correlation id
        MessageType string `json:"message_type"` // will be "record"
        Success     bool   `json:"success"`      // whether or not the control message was successful
        Err         string `json:"err"`          // error text, if any
        File        string `json:"file"`         // output file
    }

    type InfoResponse struct {
        Id          string `json:"id"`           // echo of your correlation id
        MessageType string `json:"message_type"` // will be "info"
        Success     bool   `json:"success"`      // whether or not the control message was successful
        Err         string `json:"err"`          // error text, if any
        Version     string `json:"version"`      // octopus server version
        DeviceName  string `json:"device_name"`  // device name
    }

    // DataMessage returns datapoints from the device across 
    // the channels. These data points represent incremental data
    // that has not been seen before. The data messages come at a 
    // frequency specified in the initial control messages.
    type DataMessage struct {
        Data      [][]float64 `json:"data"`       // the data for each channel, only first n relevant, n == # of channels
        Ints      [][]int64   `json:"ints"`       // the data for each channel, as integers
        LatencyMs float64     `json:"latency_ms"` // the running latency
        //Timestamp int64      `json:"timestamp"` // timestamp corresponding to this data sample

    }
