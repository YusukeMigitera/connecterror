package a

import (
	"context"

	"connectrpc.com/connect"
)

func (s *Server) goodMethodWithDeferredConversion(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	defer func() {
		if e != nil {
			e = connect.NewError(connect.CodeInternal, e)
		}
	}()
	e = plainErr()
	return
}

func (s *Server) goodMethodWithDeferredConversionExplicit(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	defer func() {
		if nil != e {
			e = toConnectErr(e)
		}
	}()
	return nil, err
}

func (s *Server) goodMethodWithUnconditionalDeferredConversion(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	defer func() { e = connect.NewError(connect.CodeInternal, e) }()
	return nil, err
}

func (s *Server) badMethodWithConversionUndone(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	defer func() { e = err }()
	defer func() {
		if e != nil {
			e = connect.NewError(connect.CodeInternal, e)
		}
	}()
	return nil, err // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}

func (s *Server) badMethodWithConversionReturningEarly(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	defer func() {
		if req == nil {
			return
		}
		if e != nil {
			e = connect.NewError(connect.CodeInternal, e)
		}
	}()
	return nil, err // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}

func (s *Server) badMethodReturningBeforeConversion(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	if req == nil {
		return nil, err // want "should return \\*connect.Error when returning nil for \\*connect.Response"
	}
	defer func() {
		if e != nil {
			e = connect.NewError(connect.CodeInternal, e)
		}
	}()
	return nil, err
}

func (s *Server) badMethodWithConversionOfOtherVar(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	var other error
	defer func() {
		if other != nil {
			e = connect.NewError(connect.CodeInternal, other)
		}
	}()
	return nil, err // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}

func (s *Server) goodMethodWithReassignBeforeBareReturn(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	if e = plainErr(); e != nil {
		e = connect.NewError(connect.CodePermissionDenied, e)
		return
	}
	res = connect.NewResponse(&Response{})
	return
}

func (s *Server) goodMethodWithReassignedLocalVar(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	e := plainErr()
	if e != nil {
		e = connect.NewError(connect.CodeInternal, e)
		return nil, e
	}
	return connect.NewResponse(&Response{}), nil
}

func (s *Server) goodMethodWithReassignInSwitch(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	e = plainErr()
	switch {
	case req == nil:
		e = connect.NewError(connect.CodeInvalidArgument, e)
		return
	case ctx == nil:
		e = connect.NewError(connect.CodeInternal, e)
		return
	}
	return connect.NewResponse(&Response{}), nil
}

func (s *Server) badMethodWithReadmeExample(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	if e = plainErr(); e != nil {
		return // want "should return \\*connect.Error when returning nil for \\*connect.Response"
	}
	res = connect.NewResponse(&Response{})
	return
}

func (s *Server) badMethodWithReassignInBranch(
	ctx context.Context,
	req *connect.Request[Request],
) (res *connect.Response[Response], e error) {
	e = plainErr()
	if req == nil {
		e = connect.NewError(connect.CodeInvalidArgument, e)
	}
	return // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}

func (s *Server) badMethodWithPlainErrorAfterConnectError(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	e := helper()
	if req == nil {
		e := plainErr()
		return nil, e // want "should return \\*connect.Error when returning nil for \\*connect.Response"
	}
	return nil, e
}

func (s *Server) badMethodWithZeroErrorVar(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	var e error
	return nil, e // want "should not return nil for both \\*connect.Response and error"
}

func (s *Server) goodMethodWithReassignInLoopBody(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	e := plainErr()
	for range 1 {
		e = connect.NewError(connect.CodeInternal, e)
		return nil, e
	}
	return connect.NewResponse(&Response{}), nil
}

// The order of statements is not followed out of a loop, since an earlier
// iteration may have assigned the variable, so all assignments count.
func (s *Server) badMethodWithAssignBeforeLoop(
	ctx context.Context,
	req *connect.Request[Request],
) (*connect.Response[Response], error) {
	var e error = helper()
	for i := range 2 {
		if i == 1 {
			return nil, e // want "should return \\*connect.Error when returning nil for \\*connect.Response"
		}
		e = plainErr()
	}
	return connect.NewResponse(&Response{}), nil
}

func toConnectErrDeferred(e error) (ce error) { // want toConnectErrDeferred:"returnsConnectError"
	defer func() {
		if ce != nil {
			ce = connect.NewError(connect.CodeInternal, ce)
		}
	}()
	return e
}

func toConnectErrReassigned(e error) (ce error) { // want toConnectErrReassigned:"returnsConnectError"
	if ce = e; ce != nil {
		ce = connect.NewError(connect.CodeInternal, ce)
		return
	}
	return nil
}
