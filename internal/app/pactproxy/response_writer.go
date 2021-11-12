package pactproxy

import (
	"net/http"
)

type wrappedResponseWriter struct {
	originalResponseWriter http.ResponseWriter
	modifiers              []interactionModifier
}

func (w *wrappedResponseWriter) Header() http.Header {
	return w.originalResponseWriter.Header()
}

func (w *wrappedResponseWriter) Write(b []byte) (int, error) {
	return w.originalResponseWriter.Write(b)
}

func (w *wrappedResponseWriter) WriteHeader(statusCode int) {
	for _, modifier := range w.modifiers {
		if ok := modifier.modifyStatusCode(w.originalResponseWriter); ok {
			return
		}
	}
	w.originalResponseWriter.WriteHeader(statusCode)
}

func wrapResponseWriter(res http.ResponseWriter, modifiers []interactionModifier) http.ResponseWriter {
	return &wrappedResponseWriter{
		originalResponseWriter: res,
		modifiers:              modifiers,
	}
}
