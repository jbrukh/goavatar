Octopus Connector
=================

Local component that interfaces with external devices and the Octopus website.



Overview
========

    bin/              scripts for installing and testing
    cmd/
        obf/          the Octopus Binary Format viewer        (see: obf --help)
        octopus/      the Octopus Connector websocket server  (see: octopus --help)
        printer/      print device streams in the console     (see: printer --help)  
    datastruct/       data structures used in the application layer
    device/           application layer generic device
    drivers/          devices we currently support
        avatar/       the AvatarEEG
        mock_avatar/  a fake AvatarEEG for testing
        thinkgear/    devices with NeuroSky ThinkGear protocol
    etc/              tools for testing
    formats/          codecs and file recorders for OBF
    socket/           the octopus socket and protocol
    util/             generic utilities
    var/              empty directory for testing files

Installation
============

Use Go 1.1 (or later). Make sure your <code>$GOPATH</code> is set. Make sure <code>$GOPATH/bin</code> is on your <code>PATH</code>. To get the repo:

    $ go get -u -v github.com/jbrukh/goavatar/...

This installs the repo into <code>$GOPATH/src/github.com/jbrukh/goavatar</code>. If you are not finding dependencies, then try go-getting them:

    $ go get <package>

To compile, test, and install everything:

    $ bin/release.sh
    
To recompile certain builds you sometimes need to do:

    $ go clean -i && bin/release.sh


Devices
=======

Commands that take command-line parameters will usually take:

    -device="avatar": one of {'avatar', 'mock_avatar', 'thinkgear'}
    -mockChannels=2: the number of channels to mock in the mock device
    -mockDevice=false: whether to use the mock device
    -mockFile="etc/1fabece1-7a57-96ab-3de9-71da8446c52c": OBF file to play back in the mock device
    -port="/dev/tty.AvatarEEG03009-SPPDev": the serial port for the device
    -repo="var": directory where recordings are stored
  
The option <code>--mockDevice</code> is short for <code>--device="mock_avatar"</code>. Usually the NeuroSky MindBand lives on port <code>/dev/tty.BrainBand-DevB</code>, so you would run like this:

    $ octopus --device=thinkgear --port="/dev/tty.BrainBand-DevB"

To simulate multiple channels when using the MockAvatarEEG, do:

    $ octopus --mockDevice --mockChannels=8
    

Protocol
========

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
            Clear       bool   `json:"clear"`        // delete the file after upload?
        }

        // RepositoryMessage performs operations on the
        // device repository.
        RepositoryMessage struct {
            Id          string `json:"id"`           // should be non-empty
            MessageType string `json:"message_type"` // should be "repository"
            Operation   string `json:"operation"`    // one of {"list", "clear", "delete", "get"}
            ResourceId  string `json:"resource_id"`  // delete a specific file, in the case of "delete" or "get"
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
            SessionId   string `json:"session_id"`   // the connector session that made this recording (see InfoResponse.SessionId)
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
            SessionId   string `json:"session_id"`   // session id, lives for the life of control socket connection
        }

        // RepositoryResponse sends back messages about repository operations.
        RepositoryResponse struct {
            Id            string          `json:"id"`             // echo of your correlation id
            MessageType   string          `json:"message_type"`   // will be "repository"
            Success       bool            `json:"success"`        // whether or not the control message was successful
            Err           string          `json:"err"`            // error text, if any
            ResourceInfos []*ResourceInfo `json:"resource_infos"` // list of files and infos
        }

        // Resource information from the repo.
        ResourceInfo struct {
            Id           string `json:"id"` // this is the resourceId
            File         string `json:"file"`
            SizeBytes    int64  `json:"size_bytes"`
            LastModified int64  `json:"last_modified"`
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

OBF Viewer
==========

Use the <code>obf</code> command to view OBF files.

    Usage of obf:
      -csv=false: output strict CSV
      -humanTime=false: format timestamps
      -plot=false: ouput the series on a gplot graph
      -seq=false: read sequential data, if available

For example:

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
