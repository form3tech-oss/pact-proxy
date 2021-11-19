package pactproxy

import (
	"net/http"
	"strconv"
)

type wrappedResponseWriter struct {
	originalResponseWriter http.ResponseWriter
	modifiers              []*interactionModifier
	responseStatusCode     int
	headersWritten         bool
}

func (w *wrappedResponseWriter) Header() http.Header {
	return w.originalResponseWriter.Header()
}

func (w *wrappedResponseWriter) Write(b []byte) (int, error) {
	body := b
	var err error
	for _, modifier := range w.modifiers {
		body, err = modifier.modifyBody(body)
		if err != nil {
			return 0, err
		}
	}

	lenB := len(b)
	lenBody := len(body)
	if lenB != lenBody {
		w.originalResponseWriter.Header().Set("Content-Length", strconv.Itoa(lenBody))
	}

	return w.originalResponseWriter.Write(body)
}

func (w *wrappedResponseWriter) WriteHeader(statusCode int) {
	for _, modifier := range w.modifiers {
		if ok, modifiedStatusCode := modifier.modifyStatusCode(); ok {
			statusCode = modifiedStatusCode
			break
		}
	}
	w.originalResponseWriter.WriteHeader(statusCode)
}

func wrapResponseWriter(res http.ResponseWriter, modifiers []*interactionModifier) (http.ResponseWriter) {
	return &wrappedResponseWriter{
		originalResponseWriter: res,
		modifiers:              modifiers,
	}
}
