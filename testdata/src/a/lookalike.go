package a

import (
	"context"

	"connectrpc.com/connect"

	otherconnect "a/other/connectrpc.com/connect"
	fakeconnect "a/xxxconnectrpc.com/connect"
)

// Handlers of packages that only look like Connect are not checked.

func (s *Server) goodMethodWithFakeResponse(
	ctx context.Context,
	req *fakeconnect.Request[Request],
) (*fakeconnect.Response[Response], error) {
	return nil, err
}

func (s *Server) goodMethodWithOtherResponse(
	ctx context.Context,
	req *otherconnect.Request[Request],
) (*otherconnect.Response[Response], error) {
	return nil, err
}

// Errors of packages that only look like Connect are not Connect errors.

func (s *Server) badMethodWithFakeError(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	return nil, fakeconnect.NewError() // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}

func (s *Server) badMethodWithOtherError(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	return nil, otherconnect.NewError() // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}
