package a

import (
	"context"
	"errors"

	"connectrpc.com/connect"
)

var err = errors.New("message")

type Server struct{}
type Request struct{}
type Response struct{}

func helper() *connect.Error {
	return connect.NewError(connect.CodeInternal, err)
}

func (s *Server) goodMethod(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	return nil, connect.NewError(connect.CodeInternal, err)
}

func (s *Server) goodMethodWithHelper(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	return nil, helper()
}

func (s *Server) badMethod(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	return nil, err // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}
