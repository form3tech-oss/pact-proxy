package pactproxy

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInteractionsWaitHandler(t *testing.T) {
	r := require.New(t)
	api := api{
		interactions: &Interactions{},
		notify:       NewNotify(),
		delay:        20 * time.Millisecond,
		duration:     150 * time.Millisecond,
	}

	for _, tt := range []struct {
		name         string
		interactions *Interactions
		req          *http.Request
		code         int
	}{
		{
			name:         "basic",
			interactions: &Interactions{},
			req: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, "/", nil)
				return req
			}(),
			code: http.StatusOK,
		},
		{
			name:         "non existing interaction",
			interactions: &Interactions{},
			req: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, "/?interaction=non-existing", nil)
				return req
			}(),
			code: http.StatusBadRequest,
		},
		{
			name: "timing out existing interaction",
			interactions: func() *Interactions {
				interactions := Interactions{}
				interactions.Store(&interaction{
					Alias:       "existing",
					Description: "Existing",
				})
				return &interactions
			}(),
			req: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, "/?interaction=existing&count=2", nil)
				return req
			}(),
			code: http.StatusRequestTimeout,
		},
	} {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			api.interactions = tt.interactions
			r.NotPanics(func() { api.interactionsWaitHandler(rec, tt.req) })
			r.Equal(tt.code, rec.Code)
		})
	}
}

func TestInteractionsGetHandler(t *testing.T) {
	r := require.New(t)
	api := api{
		interactions: &Interactions{},
		notify:       NewNotify(),
		delay:        20 * time.Millisecond,
		duration:     150 * time.Millisecond,
	}

	for _, tt := range []struct {
		name         string
		interactions *Interactions
		req          *http.Request
		code         int
		body string
	}{
		{
			name:         "empty interactions",
			interactions: &Interactions{},
			req: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, "/interactions", nil)
				return req
			}(),
			code: http.StatusOK,
			body: `{"interactions":null}`,
		},
		{
			name: "interactions by alias",
			interactions: func() *Interactions {
				interactions := Interactions{}
				interactions.Store(&interaction{
					Alias:       "existing",
					Description: "Existing",
				})
				return &interactions
			}(),
			req: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, "/interactions?alias=existing", nil)
				return req
			}(),
			code: http.StatusOK,
		},
		{
			name: "interactions by alias",
			interactions: func() *Interactions {
				interactions := Interactions{}
				interactions.Store(&interaction{
					Alias:       "existing",
					Description: "Existing",
				})
				return &interactions
			}(),
			req: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, "/interactions?alias=not", nil)
				return req
			}(),
			code: http.StatusOK,
		},
	} {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			api.interactions = tt.interactions
			r.NotPanics(func() { api.interactionsHandler(rec, tt.req) })
			r.Equal(tt.code, rec.Code)
			body := rec.Result().Body
			defer body.Close()
			data , err := ioutil.ReadAll(body)
			r.NoError(err)
			r.Equal(tt.body, string(data))
		})
	}
}
