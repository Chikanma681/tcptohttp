package response

import (
	"fmt"
	"io"

	"tcp/internal/headers"
)

type Response struct{}

type StatusCode int

const (
	StatusOK                  StatusCode = 200
	StatusBadRequest          StatusCode = 400
	StatusInternalServerError StatusCode = 500
)

func GetDefaultHeaders(contentLen int) *headers.Headers {
	h := headers.NewHeaders()
	h.Set("Content-Length", fmt.Sprintf("%d", contentLen))
	h.Set("Connection", "close")
	h.Set("Content-Type", "text/plain")

	return h
}

func WriteHeaders(w io.Writer, h *headers.Headers) error {
	var err error = nil
	h.ForEach(func(n, v string) {
		if err != nil {
			return
		}
		_, error := w.Write([]byte(fmt.Sprintf("%s: %s\r\n", n, v)))
		err = error
	})
	// Write blank line to separate headers from body
	if err == nil {
		_, err = w.Write([]byte("\r\n"))
	}
	return err
}

func WriteStatusLine(w io.Writer, statusCode StatusCode) error {
	statusLine := []byte{}
	switch statusCode {
	case StatusOK:
		statusLine = []byte("HTTP/1.1 200 OK\r\n")
	case StatusBadRequest:
		statusLine = []byte("HTTP/1.1 400 Bad Request\r\n")
	case StatusInternalServerError:
		statusLine = []byte("HTTP/1.1 500 Internal Server Error\r\n")
	default:
		return fmt.Errorf("unrecognized error code")
	}
	_, err := w.Write(statusLine)
	return err
}
