# genh is an opinionated tool for generating request-handler boilerplate for Go

[![CI](https://github.com/alecthomas/genh/actions/workflows/ci.yml/badge.svg)](https://github.com/alecthomas/genh/actions/workflows/ci.yml)

genh automatically generates `http.RequestHandler` boilerplate for routing to Go
methods annotated with comment directives. The generated code decodes the
incoming HTTP request into the method's parameters, and encodes method return
values to HTTP responses.

genh only supports JSON request/response payloads. That said, see
[below](#escape-hatch) for workarounds that can leverage genh's routing for
arbitrary requests.

genh's generated code relies on only the standard library.

Here's an example annotated method that genh will generate a request handler for:

```go
//genh:api GET /users/:id
func (s *Service) GetUser(id string) (User, error) {
	for _, user := range s.users {
		if user.ID == id {
			return user, nil
		}
	}
	return User{}, Errorf(http.StatusNotFound, "user %q not found", id)
}
```

The generated request handler will map the path component `:id` to the parameter
`id`, and JSON encode the response payload or error.

See [below](#example) for a full example.

# Protocol

genh's protocol is in the form of Go comment directives. Each directive must be
placed on a method, not a free function.

Annotations and methods are in the following form:

```go
//genh:api <method> <path>
func (s Struct) Method([pathVar0, pathVar1 string][, req Request]) ([<response>, ][error]) { ... }
```

## Request signature

The `<path>` value supports variables in the form `:<name>` which are mapped directly to
method parameters of the same name. These parameters must be of type `string`,
`int`, or implement `encoding.TextUnmarshaler`.

eg.

```go
//genh:api GET /users/:id
func (s *Struct) GetUser(id string) (User, error) { ... }
```

In addition to path variables and the request payload, genh can pass any of the
following types to your handler method:

- `*http.Request`
- `http.ResponseWriter`
- `context.Context` from the incoming `*http.Request`
- `io.Reader` for the request body

Finally, a single extra struct parameter can be specified, which will be decoded
from the request payload. For PUT/POST request the "payload" is the request
body, for all other request types the "payload" is query parameters. Use JSON
annotations for struct decoding of query parameters.

## Response signature

The return signature of the method is in the form:

```
[([<response>, ][error])]
```

That is, the method may return a response, an error, both, or nothing.

 Depending on the type of the `<response>` value, the response will be encoded
 in the following ways:

 | Type | Encoding |
 | ---- | -------- |
 | `nil`/omitted | 204 No Content |
 | `string` | `text/html` |
 | `[]byte` | `application/octet-stream` |
 | `io.Reader` | `application/octet-stream` |
 | `io.ReadCloser` | `application/octet-stream` |
 | `*http.Response` | Response structure is used as-is. |
 | `*` | `application/json` |

## Error handling

If the method returns an error, genh will generate code to check the error
and return an error response. If the error value implements `http.Handler` that
will be used to generate the response, otherwise a 500 response will be
generated.

## Escape hatch

If genh's default request/response handling is not to your liking, you can still
leverage genh's routing by accepting `*http.Request` and `http.ResponseWriter`
as parameters:

```go
//genh:api POST /users
func (s Struct) CreateUser(r *http.Request, w http.ResponseWriter) { ... }
```

# Example

Create a `main.go` with the following content and run `go generate`. genh will
create a `main_api.go` file implementing `http.Handler` for `*Service`.

```go
//go:generate genh
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// An Error that implements http.Handler to write structured JSON errors.
type Error struct {
	code    int
	message string
}

func Errorf(code int, format string, args ...interface{}) error {
	return Error{code, fmt.Sprintf(format, args...)}
}

func (e Error) Error() string { return e.message }

func (e Error) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.code)
	json.NewEncoder(w).Encode(map[string]string{"error": e.message})
}

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Service struct {
	users []User
}

//genh:api GET /users/:id
func (s *Service) GetUser(id int) (User, error) {
	for _, user := range s.users {
		if user.ID == id {
			return user, nil
		}
	}
	return User{}, Errorf(http.StatusNotFound, "user %q not found", id)
}

//genh:api GET /users
func (s *Service) ListUsers() ([]User, error) {
	return s.users, nil
}

//genh:api POST /users
func (s *Service) CreateUser(user User) error {
	for _, u := range s.users {
		if u.ID == user.ID {
			return Errorf(http.StatusConflict, "user %d already exists", user.ID)
		}
	}
	s.users = append(s.users, user)
	return Errorf(http.StatusCreated, "user %d created", user.ID)
}

func main() {
	service := &Service{
		users: []User{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}},
	}
	http.ListenAndServe(":8080", service)
}
```

*genh's annotations are vaguely inspired by [Encore](https://encore.dev/).*