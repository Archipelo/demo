package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
)

var (
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	errorType   = reflect.TypeOf((*error)(nil)).Elem()
	httpReqType = reflect.TypeOf(&http.Request{})
	emptyJSON   = []byte("{}\n")
	jsonCT      = "application/json; charset=utf-8"
)

type nestedError interface {
	Cause() error
}

func unwrap(err interface{}) error {
	if e, ok := err.(nestedError); ok {
		return unwrap(e.Cause())
	}
	return err.(error)
}

type Error struct {
	Status  int
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

type Manager struct {
}

func NewManager() *Manager {
	return &Manager{}
}

type apiFunc struct {
	f              interface{}
	fv             reflect.Value
	ft             reflect.Type
	hasAccountID   bool
	hasRequest     bool
	hasInput       bool
	hasOutput      bool
	hasOutputError bool
	inputType      reflect.Type
}

func (a *apiFunc) prepIn() error {
	if a.fv.Type().IsVariadic() {
		return errors.New("must not be variadic")
	}
	cnt := a.ft.NumIn()
	offset := 0
	if cnt > 1 {
		if a.ft.In(1) == httpReqType {
			a.hasRequest = true
			offset++
		}
	}
	switch cnt - offset {
	default:
		return errors.New("must accept 1, 2 or 3 arguments")
	case 1:
		break
	case 2:
		// either hasAccountID or hasInput
		argType := a.ft.In(offset + 1)
		a.hasInput = true
		a.inputType = argType
	case 3:
		a.hasAccountID = true
		a.hasInput = true
		a.inputType = a.ft.In(offset + 2)
	}
	if a.ft.In(0) != contextType {
		return errors.New("first argument must be context.Context")
	}
	return nil
}

func (a *apiFunc) prepOut() error {
	switch a.ft.NumOut() {
	default:
		return errors.New("must return 0, 1 or 2 values")
	case 0:
	case 1:
		if a.ft.Out(0) != errorType {
			return errors.New("single return value must be an error")
		}
		a.hasOutputError = true
	case 2:
		if a.ft.Out(1) != errorType {
			return errors.New("second return value must be an error")
		}
		a.hasOutput = true
		a.hasOutputError = true
	}
	return nil
}

func newAPIFunc(f interface{}) (*apiFunc, error) {
	af := apiFunc{f: f}
	af.fv = reflect.ValueOf(f)
	af.ft = af.fv.Type()
	if af.fv.Kind() != reflect.Func {
		return nil, errors.New("must be a function")
	}
	if err := af.prepIn(); err != nil {
		return nil, err
	}
	if err := af.prepOut(); err != nil {
		return nil, err
	}
	return &af, nil
}

func (m *Manager) we(f interface{}) (http.HandlerFunc, error) {
	af, err := newAPIFunc(f)
	if err != nil {
		return nil, err
	}
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		in := []reflect.Value{reflect.ValueOf(ctx)}
		if af.hasRequest {
			in = append(in, reflect.ValueOf(r))
		}
		if af.hasAccountID {
			accountID := 1
			in = append(in, reflect.ValueOf(accountID))
		}
		if af.hasInput {
			arg := reflect.New(af.inputType)
			decoder := json.NewDecoder(r.Body)
			decoder.DisallowUnknownFields()
			if err := decoder.Decode(arg.Interface()); err != nil {
				m.SendError(w, r, &Error{
					Status:  http.StatusBadRequest,
					Message: err.Error(),
				})
				return
			}
			in = append(in, arg.Elem())
		} else {
			var b [1]byte
			n, err := r.Body.Read(b[:])
			if err != io.EOF || n != 0 {
				m.SendError(w, r, &Error{
					Message: "expected empty request body",
					Status:  http.StatusBadRequest,
				})
				return
			}
		}
		out := af.fv.Call(in)
		if af.hasOutputError {
			err := out[len(out)-1]
			if !err.IsNil() {
				e := unwrap(err.Interface())
				if e, ok := e.(*Error); ok {
					m.SendError(w, r, e)
				} else {
					m.SendError(w, r, err.Interface().(error))
				}
				return
			}
		}
		if af.hasOutput {
			m.sendJSON(w, r, http.StatusOK, out[0].Interface())
		} else {
			w.Header().Add("Content-Type", jsonCT)
			_, _ = w.Write(emptyJSON)
		}
	}, nil
}

func (m *Manager) sendJSON(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	v interface{},
) {
	w.Header().Add("Content-Type", jsonCT)
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(v); err != nil {
		return
	}
}

func (m *Manager) SendError(w http.ResponseWriter, r *http.Request, err error) {
	status := http.StatusInternalServerError
	message := "Internal Server Error"

	if apierr, ok := err.(*Error); ok {
		status = apierr.Status
		message = apierr.Message
	}

	m.sendJSON(w, r, status, struct {
		Error string `json:"error"`
	}{
		Error: message,
	})
}

func (m *Manager) W(f interface{}) http.HandlerFunc {
	hf, err := m.we(f)
	if err != nil {
		panic(fmt.Errorf("error binding API function %T: %+v", f, err))
	}
	return hf
}
