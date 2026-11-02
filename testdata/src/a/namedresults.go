package a

import (
	"context"

	"connectrpc.com/connect"
)

func (s *Server) badMethodWithBareReturn(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	e = err
	return // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}

func (s *Server) badMethodWithExplicitNamedResults(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	e = err
	return res, e // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}

func (s *Server) badMethodWithNamedErrorReturned(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	e = err
	return nil, e // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}

func (s *Server) badMethodWithMixedNamedError(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	e = helper()
	if req == nil {
		e = err
	}
	return // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}

func (s *Server) badMethodWithNamedErrorInDefer(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	defer func() { e = err }()
	e = helper()
	return // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}

func (s *Server) badMethodWithUnassignedNamedResults(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	return // want "should not return nil for both \\*connect.Response and error"
}

func (s *Server) badMethodWithBlankNamedResults(
	ctx context.Context,
	req *connect.Request[Request],
) (_ *connect.Response[Response], _ error) {
	return // want "should not return nil for both \\*connect.Response and error"
}

func (s *Server) goodMethodWithBareReturn(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	e = connect.NewError(connect.CodeInternal, err)
	return
}

func (s *Server) goodMethodWithExplicitNamedResults(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	e = connect.NewError(connect.CodeInternal, err)
	return res, e
}

func (s *Server) goodMethodWithAllConnectNamedErrors(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	e = helper()
	if req == nil {
		e = connect.NewError(connect.CodeInvalidArgument, err)
	}
	if ctx == nil {
		e = toConnectErr(err)
	}
	return
}

func (s *Server) goodMethodWithNamedErrorInDefer(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	defer func() {
		if e != nil {
			e = connect.NewError(connect.CodeInternal, e)
		}
	}()
	e = helper()
	return
}

func (s *Server) goodMethodWithNamedResponse(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	res = connect.NewResponse(&Response{})
	return
}

func (s *Server) goodMethodWithResponseFromCall(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	res, e = s.call(ctx, req)
	return
}

func (s *Server) call(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	return connect.NewResponse(&Response{}), nil
}

func toConnectErrNamed(e error) (ce error) { // want toConnectErrNamed:"returnsConnectError"
	ce = connect.NewError(connect.CodeInternal, e)
	return
}

func toConnectErrNamedExplicit(e error) (ce error) { // want toConnectErrNamedExplicit:"returnsConnectError"
	ce = connect.NewError(connect.CodeInternal, e)
	return ce
}

func plainErrNamed() (e error) {
	e = err
	return
}

func (s *Server) goodMethodWithNamedHelper(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	return nil, toConnectErrNamed(err)
}

func (s *Server) badMethodWithNamedPlainHelper(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	return nil, plainErrNamed() // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}
