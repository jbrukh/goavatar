goavatar
========

Go parser and websocket server for the AvatarEEG data stream.

Overview
========

    bin/          scripts for installing and testing

    cmd/
        obf/      the Octopus Binary Format viewer        (see: obf --help)
        octopus/  the Octopus Connector websocket server  (see: octopus --help)
        streamer/ the gplot-based real-time data streamer (see: streamer --help)
    
    devices/      devices we currently support (AvatarEEG and MockAvatarEEG)

    etc/          tools for testing

    formats/      codecs and file recorders for OBF

    var/          empty directory for testing files

    .             framework files      

Installation
============

You will need Go 1.1 and <code>gplot</code> if you want to use the streamer. To get the repo:

    $ go get -u github.com/jbrukh/goavatar

To compile, test, and install everything:

    $ bin/release.sh


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
        MessageType string `json:"message_type"` // will be one of {"info", connect", "record", "upload", "error"}
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
    }

    // UploadMessage is used to trigger upload of a
    // recorded resource available in the local repository.
    type UploadMessage struct {
        Id          string `json:"id"`           // should be non-empty
        MessageType string `json:"message_type"` // should be "upload"
        Token       string `json:"token"`        // authentication token for upload
        ResourceId  string `json:"resource_id"`  // id of the resource to upload
    }

    // Base type for response messages.
    type Response struct {
        Id          string `json:"id"`           // echo of your correlation id
        MessageType string `json:"message_type"` // will be one of {"info", connect", "record", "upload", "error"}
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
        ResourceId  string `json:"resource_id"`  // id of the resource
    }

    // UploadResponse is sent in response to an UploadMessage, providing
    // the URL of the uploaded resource.
    type UploadResponse struct {
        Id          string `json:"id"`           // echo of your correlation id
        MessageType string `json:"message_type"` // will be "upload"
        Success     bool   `json:"success"`      // whether or not the control message was successful
        Err         string `json:"err"`          // error text, if any
        ResourceUrl string `json:"resource_url"` // url of the uploaded resource
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

OBF Viewer
==========

    $ time obf var/12fb51e907ea112a | head -40
    # Octopus Binary Format.
    #
    # Copyright (c) 2013. Jake Brukhman/Octopus.
    # All rights reserved.
    #
    # HEADER ----------------------------------
    # DataType:       1
    # FormatVersion:  1
    # StorageMode:    1
    # Channels:       2
    # Samples:        1520
    # SampleRate:     250
    # ------------------------------------------
    timestamp,channel1,channel2
    1345796045054931640,0.50088380277156829834,0.50130996108055114746
    1345796045058931640,0.50079300999641418457,0.50126963853836059570
    1345796045062931640,0.50051495432853698730,0.50136803090572357178
    1345796045066931640,0.50027900934219360352,0.50124102830886840820
    1345796045070931640,0.50031083822250366211,0.50137455761432647705
    1345796045074931640,0.50034177303314208984,0.50138631463050842285
    1345796045078931640,0.50037132203578948975,0.50136722624301910400
    1345796045082931640,0.50039720535278320312,0.50129100680351257324
    1345796045086931640,0.50042653083801269531,0.50134509801864624023
    1345796045090931640,0.50045697391033172607,0.50128161907196044922
    1345796045094931640,0.50048710405826568604,0.50140835344791412354
    1345796045098931640,0.50051616132259368896,0.50131402909755706787
    1345796045102931640,0.50054503977298736572,0.50132882595062255859
    1345796045106931640,0.50057396292686462402,0.50134214758872985840
    1345796045110931640,0.50060141086578369141,0.50126892328262329102
    1345796045114931640,0.50062935054302215576,0.50138886272907257080
    1345796045118896484,0.50065769255161285400,0.50123070180416107178
    1345796045122896484,0.50068885087966918945,0.50134089589118957520
    1345796045126896484,0.50071911513805389404,0.50133928656578063965
    1345796045130896484,0.50074808299541473389,0.50131858885288238525
    1345796045134896484,0.50077910721302032471,0.50128304958343505859
    1345796045138896484,0.50081008672714233398,0.50137670338153839111
    1345796045142896484,0.50084093213081359863,0.50131501257419586182
    1345796045146896484,0.50086618959903717041,0.50137008726596832275
    1345796045150896484,0.50089269876480102539,0.50136709213256835938
    1345796045154896484,0.50092291831970214844,0.50139266252517700195

    real    0m0.018s
    user    0m0.013s
    sys 0m0.005s

License
=======

Copyright (c) 2013. Jake Brukhman/Octopus. All rights reserved. This is proprietary software. Do not distribute.
