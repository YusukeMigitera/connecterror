package dot

import (
	"context"
	"errors"

	. "connectrpc.com/connect"
)

var err = errors.New("message")

type Server struct{}
type Req struct{}
type Res struct{}

func (s *Server) goodMethod(
	ctx context.Context,
	req *Request[Req],
) (*Response[Res], error) {
	return nil, NewError(CodeInternal, err)
}

func (s *Server) badMethod(
	ctx context.Context,
	req *Request[Req],
) (*Response[Res], error) {
	return nil, err // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}
