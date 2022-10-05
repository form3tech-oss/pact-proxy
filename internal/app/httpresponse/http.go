package httpresponse

import (
	"fmt"

	log "github.com/sirupsen/logrus"
)

func Error(error string) *APIError {
	log.Error(error)
	e := &APIError{
		ErrorMessage: error,
	}
	return e
}

func Errorf(error string, a ...interface{}) *APIError {
	return Error(fmt.Sprintf(error, a...))
}
