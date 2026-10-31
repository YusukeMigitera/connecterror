package a

import (
	"context"

	"connectrpc.com/connect"
)

func (s *Server) badMethodWithNilNil(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	return nil, nil // want "should not return nil for both \\*connect.Response and error"
}

func (s *Server) badMethodWithParenNilNil(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	return (nil), (nil) // want "should not return nil for both \\*connect.Response and error"
}

func (s *Server) goodMethodWithShadowedNil(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	nil := connect.NewResponse(&Response{})
	return nil, err
}

func threeResults() (*connect.Response[Response], *Request, error) {
	return nil, nil, helper()
}

func badThreeResults() (*connect.Response[Response], *Request, error) {
	return nil, nil, err // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}
