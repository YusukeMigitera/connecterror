package a

import (
	"context"

	c "connectrpc.com/connect"
)

func (s *Server) goodMethodWithAlias(
	ctx context.Context,
	req *c.Request[Request],
) (*c.Response[Response], error) {
	return nil, c.NewError(c.CodeInternal, err)
}

func (s *Server) badMethodWithAlias(
	ctx context.Context,
	req *c.Request[Request],
) (*c.Response[Response], error) {
	return nil, err // want "should return \\*connect.Error when returning nil for \\*connect.Response"
}

// notHandler does not return error last, so it is not a Connect handler.
func notHandler() (*Response, *c.Response[Response]) {
	return nil, &c.Response[Response]{}
}
