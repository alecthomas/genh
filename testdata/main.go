//go:generate genh
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"
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

type ID int

func (i *ID) UnmarshalText(text []byte) error {
	id, err := strconv.Atoi(string(text))
	if err != nil {
		return err
	}
	*i = ID(id)
	return nil
}

type User struct {
	ID   ID     `json:"id"`
	Name string `json:"name"`
}

type Service struct {
	users []User
}

//genh:api GET /users/:id
func (s *Service) GetUser(id ID) (User, error) {
	for _, user := range s.users {
		if user.ID == id {
			return user, nil
		}
	}
	return User{}, Errorf(http.StatusNotFound, "user %d not found", id)
}

//genh:api GET /users/:id/avatar
func (s *Service) GetAvatar(id ID) ([]byte, error) {
	return nil, Errorf(http.StatusNotFound, "avatar %d not found", id)
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

//genh:api GET /users
func (s *Service) ListUsers() ([]User, error) {
	return s.users, nil
}

//genh:api POST /shutdown
func (s *Service) Shutdown(w http.ResponseWriter) {
	fmt.Fprintln(w, "Shutting down...")
	go func() { time.Sleep(time.Second); os.Exit(0) }()
}

func main() {
	service := &Service{
		users: []User{{ID: 1, Name: "Alice"}, {ID: 2, Name: "Bob"}},
	}
	http.ListenAndServe("127.0.0.1:8080", service)
}
