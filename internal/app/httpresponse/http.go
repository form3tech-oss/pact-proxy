package httpresponse

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/sirupsen/logrus"
)

func Error(w http.ResponseWriter, code int, error string) {
	log.Error(error)
	e := &APIError{
		ErrorMessage: error,
	}

	w.WriteHeader(code)
	w.Header().Add("Content-Type", "application/json")

	payload, err := json.Marshal(e)
	if err != nil {
		log.WithError(err).Error("could not marshal error into json")
	}

	_, err = fmt.Fprintln(w, payload)
	if err != nil {
		log.WithError(err).Error("could not write httpresponse error")
	}
}

func Errorf(w http.ResponseWriter, code int, error string, a ...interface{}) {
	Error(w, code, fmt.Sprintf(error, a...))
}
