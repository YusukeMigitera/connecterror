package a

import (
	"context"
	"fmt"
	"strconv"

	"connectrpc.com/connect"
)

func (s *Server) goodMethodWithErrorVar(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	var e error = connect.NewError(connect.CodeInternal, err)
	return nil, e
}

func (s *Server) goodMethodWithReassignedErrorVar(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	var e error = helper()
	if req == nil {
		e = connect.NewError(connect.CodeInvalidArgument, err)
	}
	return nil, e
}

func (s *Server) goodMethodWithChainedErrorVars(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	var e1 error = helper()
	e2 := e1
	return nil, e2
}

func (s *Server) goodMethodWithSelfWrappedErrorVar(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	var e error = helper()
	e = fmt.Errorf("wrap: %w", e)
	return nil, e
}

func (s *Server) badMethodWithMixedErrorVar(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	var e error = helper()
	if req == nil {
		e = err
	}
	return nil, e // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}

func (s *Server) badMethodWithErrorVarViaPointer(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	var e error = helper()
	p := &e
	*p = err
	return nil, e // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}

func (s *Server) badMethodWithErrorVarInClosure(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	var e error = helper()
	func() { e = err }()
	return nil, e // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}

func (s *Server) badMethodWithMultiValueAssign(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	_, e := strconv.Atoi("x")
	return nil, e // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}

func (s *Server) badMethodWithRangeVar(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	for _, e := range []error{helper(), err} {
		return nil, e // want "should return \\*connect.Error when returning nil for \\*connect.Response"
	}
	return nil, helper()
}

func (s *Server) badMethodWithRangeAssign(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	var e error = helper()
	for _, e = range []error{err} {
	}
	return nil, e // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}
