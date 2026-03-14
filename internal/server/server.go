package server

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"sync"

	"tcp/internal/request"
	"tcp/internal/response"
)

type (
	Handler func(w io.Writer, req *request.Request) *HandlerError
)

type HandlerError struct {
	StatusCode response.StatusCode
	Message    string
}

type Server struct {
	closed   bool
	closeMu  sync.Mutex
	handler  Handler
	listener net.Listener
}

func runConnection(s *Server, conn io.ReadWriteCloser) {
	defer conn.Close()

	headers := response.GetDefaultHeaders(0)
	r, err := request.RequestFromReader(conn)
	if err != nil {
		response.WriteStatusLine(conn, response.StatusBadRequest)
		response.WriteHeaders(conn, headers)
		return
	}

	writer := bytes.NewBuffer([]byte{})
	handlerError := s.handler(writer, r)

	var body []byte = nil
	var status response.StatusCode = response.StatusOK
	if handlerError != nil {
		// response.WriteStatusLine(conn, handlerError.StatusCode)
		// response.WriteHeaders(conn, headers)
		// conn.Write([]byte(handlerError.Message))
		status = handlerError.StatusCode
		body = []byte(handlerError.Message)
	} else {
		body = writer.Bytes()
	}

	headers.Replace("Content-length", fmt.Sprintf("%d", len(body)))
	response.WriteStatusLine(conn, status)
	response.WriteHeaders(conn, headers)
	conn.Write(body)
}

func runServer(s *Server, listener net.Listener) {
	for {
		s.closeMu.Lock()
		isClosed := s.closed
		s.closeMu.Unlock()

		if isClosed {
			return
		}

		conn, err := listener.Accept()
		if err != nil {
			return
		}
		go runConnection(s, conn)
	}
}

func Serve(port int) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	server := &Server{
		listener: listener,
	}
	go runServer(server, listener)

	return server, nil
}

func ServeWithHandler(port int, handler Handler) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	server := &Server{
		handler:  handler,
		listener: listener,
	}
	go runServer(server, listener)

	return server, nil
}

func (s *Server) Close() error {
	s.closeMu.Lock()
	s.closed = true
	s.closeMu.Unlock()

	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}
