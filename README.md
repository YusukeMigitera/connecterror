# connecterror

`connecterror` reports [Connect](https://connectrpc.com/) handlers that return errors other than `*connect.Error`.

Errors other than `*connect.Error` reach the client as `CodeUnknown`, so it is easy to forget to give an error a proper status code.

## Example

```go
func (s *Server) GetUser(
	ctx context.Context,
	req *connect.Request[userv1.GetUserRequest],
) (*connect.Response[userv1.GetUserResponse], error) {
	user, err := s.repo.Find(ctx, req.Msg.Id)
	if err != nil {
		return nil, err // should return *connect.Error when returning nil for *connect.Response
	}
	// ...
}
```

Return a `*connect.Error` with a status code instead:

```go
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
```

## What is checked

A Connect handler is a function whose first result is `*connect.Response[T]` and whose last result is `error`. When a handler returns `nil` for the response, the error is accepted if it is

- a `*connect.Error`,
- a `*connect.Error` wrapped with `fmt.Errorf`'s `%w` verb, which Connect still finds with `errors.As`,
- a local variable or named result whose last assignment before the `return` is such an error, or that is only ever assigned such errors or `nil`,
- a named result converted to such an error by a deferred closure, or
- the result of a function that only returns such errors or `nil`, even in another package:

  ```go
  func toConnectErr(err error) error {
  	return connect.NewError(connect.CodeInternal, err)
  }

  return nil, toConnectErr(err) // OK
  ```

`return nil, nil` is also reported, because Connect panics when a handler returns nil for both results.

Named results are followed like local variables, so a bare `return` is checked too. A named response that is not assigned before the `return`, or is only ever assigned `nil`, counts as `nil`:

```go
func (s *Server) GetUser(
	ctx context.Context,
	req *connect.Request[userv1.GetUserRequest],
) (res *connect.Response[userv1.GetUserResponse], err error) {
	if err = s.authorize(ctx); err != nil {
		return // should return *connect.Error when returning nil for *connect.Response
	}
	// ...
}
```

Either fix is accepted: reassign the error before returning,

```go
	if err = s.authorize(ctx); err != nil {
		err = connect.NewError(connect.CodePermissionDenied, err)
		return
	}
```

or convert the named error in a deferred closure at the top of the handler, which covers every `return` after it:

```go
	defer func() {
		if err != nil {
			err = connect.NewError(connect.CodeInternal, err)
		}
	}()
```

### Limitations

- Only `return nil, err` is checked; returns with a non-nil response are not.
- Only a literal `return nil, nil`, or named results that are always `nil`, are reported; an error that turns out to be nil at runtime is not.
- The last assignment before a `return` is followed only through blocks, `if` and `switch`. Across loops, `select` and labeled statements, in functions with `goto` or `fallthrough`, and for variables assigned in closures, every assignment to the variable counts.
- A deferred conversion is recognized only as a `defer func() { ... }()` at the top level of the function, assigning a Connect error to the named error either directly or inside `if err != nil`, without returning earlier. Every closure in the function must assign the named error only Connect errors or `nil`.
- Calls through interfaces or function values are not followed.
- `fmt.Errorf` formats with explicit argument indexes, such as `%[1]w`, are not followed.

## Install

```sh
go install github.com/YusukeMigitera/connecterror/cmd/connecterror@latest
```

## Usage

```sh
go vet -vettool=$(which connecterror) ./...
```
