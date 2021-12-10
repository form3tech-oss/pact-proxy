package pactproxy

import (
	"io"
	"net/http"
	"strconv"
)

type ResponseModificationWriter struct {
	res          http.ResponseWriter
	interactions []*interaction
	statusCode   int
}

func (m *ResponseModificationWriter) Header() http.Header {
	return m.res.Header()
}

func (m *ResponseModificationWriter) Write(b []byte) (written int, err error) {
	written = len(b)

	for _, i := range m.interactions {
		b, err = i.Modifiers.modifyBody(b)
		if err != nil {
			return 0, err
		}
	}

	m.Header().Set("Content-Length", strconv.Itoa(len(b)))
	m.res.WriteHeader(m.statusCode)
	actualBytes, err := m.res.Write(b)
	if err != nil {
		return 0, err
	}

	if actualBytes != len(b) {
		return actualBytes, io.ErrShortWrite
	}
	return
}

func (m *ResponseModificationWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
	for _, i := range m.interactions {
		ok, code := i.Modifiers.modifyStatusCode()
		if ok {
			m.statusCode = code
			break
		}
	}

	contentLength, err := strconv.Atoi(m.Header().Get("Content-Length"))
	if err != nil || contentLength == 0 {
		m.res.WriteHeader(m.statusCode)
	}
}
