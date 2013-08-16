//
// Copyright (c) 2013 Jake Brukhman/Octopus. All rights reserved.
//
package socket

import (
	"bytes"
	"fmt"
	. "github.com/jbrukh/goavatar"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
)

func UploadS3(device, sessionId, file, aws_key_id, aws_bucket string) (err error) {

}

func UploadOBFFile(device, sessionId, file, endpoint, token string) (err error) {
	log.Printf("uploading file %s to endpoint: %s", file, endpoint)

	// Create buffer
	var (
		buf = new(bytes.Buffer) // TODO may be bad for large files
		w   = multipart.NewWriter(buf)
	)

	// note the file name
	fileField, err := w.CreateFormField("filename")
	if err != nil {
		return
	}
	fileField.Write([]byte(file))

	// note the device name
	deviceField, err := w.CreateFormField("device_name")
	if err != nil {
		return
	}
	deviceField.Write([]byte(device))

	// note the goavatar version
	versionField, err := w.CreateFormField("version")
	if err != nil {
		return
	}
	versionField.Write([]byte(Version()))

	sessionIdField, err := w.CreateFormField("session_id")
	if err != nil {
		return
	}
	sessionIdField.Write([]byte(sessionId))

	// create file field
	fw, err := w.CreateFormFile("result[data]", file)
	if err != nil {
		return
	}

	fd, err := os.Open(file)
	if err != nil {
		return
	}
	defer fd.Close()

	// write file field from file to upload
	_, err = io.Copy(fw, fd)
	if err != nil {
		return
	}

	// close and get the terminating boundary
	w.Close()

	authenticatedEndpoint := fmt.Sprintf("%s?auth_token=%s", endpoint, token)
	req, err := http.NewRequest("POST", authenticatedEndpoint, buf)
	if err != nil {
		return
	}

	//tokenStr := fmt.Sprintf("auth_token %s", token)
	req.Header.Set("Content-Type", w.FormDataContentType())
	//req.Header.Set("Authorization", tokenStr)
	req.Header.Set("Accept", "application/json")

	//log.Println("Making HTTP request: %v", req)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return
	}

	log.Printf("UPLOAD REQUEST -------------------------------------")
	io.Copy(os.Stdout, req.Body)
	log.Printf("UPLOAD RESPONSE -------------------------------------")
	io.Copy(os.Stdout, res.Body) // replace this with status check

	fmt.Println()
	if res.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to upload, status: %v", res.StatusCode)
	}
	return
}
