package request

import (
	// "io"
	"strings"
	"fmt"
)

type RequestLine struct {
	HttpVersion string
	RequestTarget string
	Method string
}

type Request struct {
	RequestLine RequestLine
}

func (r *RequestLine) ValidHTTP () bool {
	return r.HttpVersion == "HTTP/1.1" 
} 

var ERROR_BAD_REQUEST_LINE = fmt.Errorf("malformed request-line")
var SEPARATOR = "\r\n"

func parseRequestLine(b string) (*RequestLine, string, error) {
	idx := strings.Index(b, SEPARATOR)
	if idx == -1 {
		return nil, b, nil
	}

	startLine := b[:idx]
	restOfMsg := b[idx+len(SEPARATOR):]

	parts := strings.Split(startLine, "  ")

	if len(parts) != 3 {
		return nil, restOfMsg, ERROR_BAD_REQUEST_LINE		
	}

	rl := &RequestLine{
		Method: parts[0],
		RequestTarget: parts[1],
		HttpVersion: parts[2],
	}

	if !rl.ValidHTTP() {
		return nil, restOfMsg, ERROR_BAD_REQUEST_LINE
	}
	return  rl, restOfMsg, nil
}
// func RequestFromReader(reader io.Reader) (*Request, error) {
	
// }
