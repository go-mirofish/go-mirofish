package worker

import "errors"

var (
	ErrWorkerNotFound    = errors.New("worker simulation not found")
	ErrWorkerNotReady    = errors.New("worker environment not ready")
	ErrWorkerBadRequest  = errors.New("worker bad request")
	ErrWorkerUnavailable = errors.New("worker unavailable")
	ErrWorkerTimeout     = errors.New("worker timeout")
)

type Error struct {
	Op     string
	Kind   error
	Detail string
	Err    error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Detail != "" {
		return e.Detail
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	if e.Kind != nil {
		return e.Kind.Error()
	}
	return "worker error"
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	if e.Err != nil {
		return e.Err
	}
	return e.Kind
}
