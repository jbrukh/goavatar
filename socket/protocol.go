//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package socket

//---------------------------------------------------------//
// Messages
//---------------------------------------------------------//

type (

	// Base type for messages.
	Message struct {
		Id          string `json:"id"`           // should be non-empty
		MessageType string `json:"message_type"` // will be one of {"info", connect", "record", "upload", "error"}
	}

	// Basic information about the server.
	InfoMessage struct {
		Id          string `json:"id"`           // should be non-empty
		MessageType string `json:"message_type"` // should be "info"
	}

	// ConnectMessage is used to connect to the device
	// and begin streaming. A ConnectResponseMessage is
	// sent to indicate success or failure, and data
	// immediately begins to flow on the data endpoint.
	ConnectMessage struct {
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
	RecordMessage struct {
		Id          string `json:"id"`           // should be non-empty
		MessageType string `json:"message_type"` // should be "record"
		Record      bool   `json:"record"`       // start or stop recording
		Seconds     int    `json:"seconds"`      // number of seconds after which to cease recording
	}

	// UploadMessage is used to trigger upload of a
	// recorded resource available in the local repository.
	UploadMessage struct {
		Id          string `json:"id"`           // should be non-empty
		MessageType string `json:"message_type"` // should be "upload"
		Token       string `json:"token"`        // authentication token for upload
		ResourceId  string `json:"resource_id"`  // id of the resource to upload
		Endpoint    string `json:"endpoint"`     // domain-qualified endpoint to upload to
		Clear       string `json:"clear"`        // delete the file after upload
	}

	// RepositoryMessage performs operations on the
	// device repository.
	RepositoryMessage struct {
		Id          string `json:"id"`           // should be non-empty
		MessageType string `json:"message_type"` // should be "repository"
		Operation   string `json:"operation"`    // one of {"list", "clear"}
		ResourceId  string `json:"resource_id"`  // delete a specific file
	}

	// Base type for response messages.
	Response struct {
		Id          string `json:"id"`           // echo of your correlation id
		MessageType string `json:"message_type"` // will be one of {"info", connect", "record", "upload", "error"}
		Success     bool   `json:"success"`      // whether or not the control message was successful
		Err         string `json:"err"`          // error text, if any
	}

	// ConnectResponseMessage is sent in response to a ConnectMessage.
	// The MessageType is set to "connect".
	ConnectResponse struct {
		Id          string `json:"id"`           // echo of your correlation id
		MessageType string `json:"message_type"` // will be "connect"
		Success     bool   `json:"success"`      // whether or not the control message was successful
		Err         string `json:"err"`          // error text, if any
		Status      string `json:"status"`       // device status, one of {"armed", "busy", "disconnected"}
	}

	// RecordResponseMessage is sent in response to a RecordMessage.
	// The MessageType is set to "record".
	RecordResponse struct {
		Id          string `json:"id"`           // echo of your correlation id
		MessageType string `json:"message_type"` // will be "record"
		Success     bool   `json:"success"`      // whether or not the control message was successful
		Err         string `json:"err"`          // error text, if any
		ResourceId  string `json:"resource_id"`  // id of the resource
		Seconds     int    `json:"seconds"`      // number of seconds recorder if this was a fixed-time recording
	}

	// UploadResponse is sent in response to an UploadMessage, providing
	// the URL of the uploaded resource.
	UploadResponse struct {
		Id          string `json:"id"`           // echo of your correlation id
		MessageType string `json:"message_type"` // will be "upload"
		Success     bool   `json:"success"`      // whether or not the control message was successful
		Err         string `json:"err"`          // error text, if any
	}

	// InfoResponse sends back information about the device and server.
	InfoResponse struct {
		Id          string `json:"id"`           // echo of your correlation id
		MessageType string `json:"message_type"` // will be "info"
		Success     bool   `json:"success"`      // whether or not the control message was successful
		Err         string `json:"err"`          // error text, if any
		Version     string `json:"version"`      // octopus server version
		DeviceName  string `json:"device_name"`  // device name
	}

	// RepositoryResponse sends back messages about repository operations.
	RepositoryResponse struct {
		Id          string   `json:"id"`           // echo of your correlation id
		MessageType string   `json:"message_type"` // will be "repository"
		Success     bool     `json:"success"`      // whether or not the control message was successful
		Err         string   `json:"err"`          // error text, if any
		ResourceIds []string `json:"resource_ids"` // list of files
	}

	// DataMessage returns datapoints from the device across
	// the channels. These data points represent incremental data
	// that has not been seen before. The data messages come at a
	// frequency specified in the initial control messages.
	DataMessage struct {
		Data      [][]float64 `json:"data"`       // the data for each channel, only first n relevant, n == # of channels
		Ints      [][]int64   `json:"ints"`       // the data for each channel, as integers
		LatencyMs float64     `json:"latency_ms"` // the running latency
		//Timestamp int64      `json:"timestamp"` // timestamp corresponding to this data sample

	}
)
