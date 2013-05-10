package socket

import (
	"bytes"
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
	fw, err := w.CreateFormFile("data", file)
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

	req, err := http.NewRequest("POST", endpoint, buf)
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", token)

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return
	}

	io.Copy(os.Stderr, res.Body) // replace this with status check
	return
}
