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
	"path/filepath"
)

// constants
const (
	DEFAULT_ACL             = "private"
	SUCCESS_ACTION_REDIRECT = "http://localhost/"
	AWS_ENDPOINT_FMT        = "https://%s.s3.amazonaws.com"
)

// AWS request fields
const (
	AF_KEY                     = "key"
	AF_AWS_ACCESS_KEY_ID       = "AWSAccessKeyId"
	AF_ACL                     = "acl"
	AF_SUCCESS_ACTION_REDIRECT = "success_action_redirect"
	AF_POLICY                  = "policy"
	AF_SIGNATURE               = "signature"
	AF_CONTENT_TYPE            = "Content-Type"
	AF_FILE                    = "file"
)

// S3 parameters from uploading, including
// security credentials and the path, resourceId of the
// local file we are uploading
type S3UploadParameters struct {
	file           string // path of local file for upload
	resourceId     string // usually name of file == resourceId
	awsAccessKeyId string
	awsBucket      string
	acl            string
	policy         string
	signature      string
	contentType    string
}

// Upload a file to S3
// Following http://aws.amazon.com/articles/1434?_encoding=UTF8&jiveRedirect=1
func UploadS3(p S3UploadParameters) (err error) {
	// check consistency
	if p.file == "" || p.awsAccessKeyId == "" || p.awsBucket == "" ||
		p.policy == "" || p.signature == "" {
		return fmt.Errorf("You must provide all of: file, awsAccessKeyId, awsBucket, policy, signature.")
	}

	// default acl
	if p.acl != "private" && p.acl != "public-read" {
		p.acl = DEFAULT_ACL // private
	}

	// fill in resource id if it is not given
	if p.resourceId == "" {
		p.resourceId = filepath.Base(p.file)
	}

	// starting upload
	log.Printf("uploading file to S3 endpoint")
	var (
		buf = new(bytes.Buffer) // TODO may be bad for large files
		w   = multipart.NewWriter(buf)
	)

	// this handles the addressing of data files in S3; currently
	// we are sticking all data files into the same bucket nakedly.
	// TODO
	keyField, err := w.CreateFormField(AF_KEY)
	if err != nil {
		return
	}
	keyField.Write([]byte(p.resourceId))

	// AWS_ACCESS_KEY_ID field
	accessKeyIdField, err := w.CreateFormField(AF_AWS_ACCESS_KEY_ID)
	if err != nil {
		return
	}
	accessKeyIdField.Write([]byte(p.awsAccessKeyId))

	// acl
	aclField, err := w.CreateFormField(AF_ACL)
	if err != nil {
		return
	}
	aclField.Write([]byte(DEFAULT_ACL)) // private

	// success_action_redirect
	redirectField, err := w.CreateFormField(AF_SUCCESS_ACTION_REDIRECT)
	if err != nil {
		return
	}
	redirectField.Write([]byte(SUCCESS_ACTION_REDIRECT))

	// policy
	policyField, err := w.CreateFormField(AF_POLICY)
	if err != nil {
		return
	}
	policyField.Write([]byte(p.policy))

	// signature
	signatureField, err := w.CreateFormField(AF_SIGNATURE)
	if err != nil {
		return
	}
	signatureField.Write([]byte(p.signature))

	// Content-Type
	contentTypeField, err := w.CreateFormField(AF_CONTENT_TYPE)
	if err != nil {
		return
	}
	contentTypeField.Write([]byte(p.contentType))

	// create file field
	fw, err := w.CreateFormFile(AF_FILE, p.file)
	if err != nil {
		return
	}

	// ...and copy the file over
	fd, err := os.Open(p.file)
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

	// create the endpoint URL
	endpoint := fmt.Sprintf(AWS_ENDPOINT_FMT, p.awsBucket)

	// create the request
	req, err := http.NewRequest("POST", endpoint, buf)
	if err != nil {
		return
	}

	// set the headers
	req.Header.Set("Content-Type", w.FormDataContentType())

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
	if res.StatusCode != http.StatusMovedPermanently {
		return fmt.Errorf("expecting HTTP 301 but: %v", res.StatusCode)
	}
	return
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
	req, err := http.NewRequest("PATCH", authenticatedEndpoint, buf)
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
