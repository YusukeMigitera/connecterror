package a

import (
	"context"

	"a/errutil"

	"connectrpc.com/connect"
)

// toConnectErrChained is declared before the function it calls.
func toConnectErrChained(e error) error { // want toConnectErrChained:"returnsConnectError"
	return toConnectErr(e)
}

func toConnectErr(e error) error { // want toConnectErr:"returnsConnectError"
	if e == nil {
		return nil
	}
	return connect.NewError(connect.CodeInternal, e)
}

func retry(n int) error { // want retry:"returnsConnectError"
	if n == 0 {
		return helper()
	}
	return retry(n - 1)
}

func load() (*Response, error) { // want load:"returnsConnectError"
	return nil, helper()
}

func (s *Server) toErr(e error) error { // want toErr:"returnsConnectError"
	return connect.NewError(connect.CodeInternal, e)
}

func plainErr() error {
	return err
}

func maybeConnectErr(b bool) error {
	if b {
		return helper()
	}
	return err
}

func loadPlain() (*Response, error) {
	return nil, err
}

func (s *Server) goodMethodWithHelperFunc(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	if req == nil {
		return nil, toConnectErrChained(err)
	}
	if e := toConnectErr(err); e != nil {
		return nil, e
	}
	if _, e := load(); e != nil {
		return nil, e
	}
	if e := retry(3); e != nil {
		return nil, e
	}
	if e := s.toErr(err); e != nil {
		return nil, e
	}
	return nil, errutil.ToConnectErr(err)
}

func (s *Server) badMethodWithPlainHelper(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	return nil, plainErr() // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}

func (s *Server) badMethodWithMaybeConnectHelper(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	return nil, maybeConnectErr(true) // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}

func (s *Server) badMethodWithPlainMultiValue(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	_, e := loadPlain()
	return nil, e // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}

func (s *Server) badMethodWithFuncValue(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	fn := toConnectErr
	return nil, fn(err) // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}
