package pactproxy

import (
	"io"
	"net/http"
	"strconv"
)

type ResponseModificationWriter struct {
	res              http.ResponseWriter
	interactions     []*interaction
	originalResponse []byte
	statusCode       int
}

func (m *ResponseModificationWriter) Header() http.Header {
	return m.res.Header()
}

func (m *ResponseModificationWriter) Write(b []byte) (int, error) {
	originalResponseLength, err := strconv.Atoi(m.res.Header().Get("Content-Length"))
	if err != nil {
		return 0, err
	}

	m.originalResponse = append(m.originalResponse, b...)
	if len(m.originalResponse) != originalResponseLength {
		return len(b), nil
	}

	var modifiedBody []byte
	for _, i := range m.interactions {
		modifiedBody, err = i.Modifiers.modifyBody(m.originalResponse)
		if err != nil {
			return 0, err
		}
	}

	m.Header().Set("Content-Length", strconv.Itoa(len(modifiedBody)))
	m.res.WriteHeader(m.statusCode)
	writtenBytes, err := m.res.Write(modifiedBody)
	if err != nil {
		return 0, err
	}

	if writtenBytes != len(modifiedBody) {
		return writtenBytes, io.ErrShortWrite
	}
	return len(b), nil
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
