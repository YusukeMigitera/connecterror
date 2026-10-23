package a

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
)

func (s *Server) goodMethodWithWrap(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	return nil, fmt.Errorf("wrap: %w", connect.NewError(connect.CodeInternal, err))
}

func (s *Server) goodMethodWithWrapAndOtherVerbs(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	return nil, fmt.Errorf("100%% %s %*d: %w", "op", 3, 1, helper())
}

func (s *Server) goodMethodWithNestedWrap(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	return nil, fmt.Errorf("outer: %w", fmt.Errorf("inner: %w", helper()))
}

func (s *Server) badMethodWithVerbV(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	return nil, fmt.Errorf("wrap: %v", helper()) // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}

func (s *Server) badMethodWithWrapNonConnectError(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	return nil, fmt.Errorf("wrap: %w", err) // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}
