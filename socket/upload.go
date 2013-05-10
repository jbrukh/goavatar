package socket

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
)

const UploadEndpoint = "http://localhost:3000/recordings/%s/results"

func UploadOBFFile(file string, endpoint string, token string) (err error) {
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

	io.Copy(os.Stderr, res.Body) // replace this with status check
	return
}
