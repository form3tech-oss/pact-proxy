package pactproxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInteractionsWaitHandler(t *testing.T) {
	r := require.New(t)
	api := api{
		interactions: &Interactions{},
	}

	for _, tt := range []struct {
		name string
		req  *http.Request
		code int
	}{
		{
			name: "basic",
			req: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, "/", nil)
				return req
			}(),
			code: http.StatusOK,
		},
		{
			name: "non existing interaction",
			req: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, "/?interaction=non-existing", nil)
				return req
			}(),
			code: http.StatusBadRequest,
		},
	} {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			r.NotPanics(func() { api.interactionsWaitHandler(rec, tt.req) })
			r.Equal(tt.code, rec.Code)
		})
	}
}
