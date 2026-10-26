package errutil

import "connectrpc.com/connect"

func ToConnectErr(err error) error { // want ToConnectErr:"returnsConnectError"
	return connect.NewError(connect.CodeInternal, err)
}
