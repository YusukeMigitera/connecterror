// Package connect has the same name and types as connectrpc.com/connect, but
// a different import path that ends in connectrpc.com/connect.
package connect

type Request[T any] struct{}

type Response[T any] struct{}

type Error struct{}

func (*Error) Error() string { return "fake" }

func NewError() *Error { return &Error{} }
