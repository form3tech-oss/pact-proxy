package pactproxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestInteractionsWaitHandler(t *testing.T) {
	r := require.New(t)
	a := api{
		notify:   NewNotify(),
		delay:    20 * time.Millisecond,
		duration: 150 * time.Millisecond,
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
			e := echo.New()
			c := e.NewContext(tt.req, rec)
			a.interactions = tt.interactions

			r.NotPanics(func() { a.interactionsWaitHandler(c) })
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
		body         string
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
				request := map[string]interface{}{
					"body": map[string]interface{}{"foo": "bar"},
					"path": "/test",
				}
				i := interaction{
					Alias:       "existing",
					Description: "Existing",
				}
				i.StoreRequest(request)
				interactions.Store(&i)
				return &interactions
			}(),
			req: func() *http.Request {
				req, _ := http.NewRequest(http.MethodGet, "/interactions?alias=existing", nil)
				return req
			}(),
			code: http.StatusOK,
			body: `{"interactions":[{"method":"","alias":"existing","description":"Existing","definition":null,"constraints":{},"modifiers":null,"request_count":1,"request_history":[{"body":{"foo":"bar"},"path":"/test"}]}]}`,
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
			body: `{"interactions":null}`,
		},
	} {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			e := echo.New()
			c := e.NewContext(tt.req, rec)
			api.interactions = tt.interactions
			r.NotPanics(func() { api.interactionsGetHandler(c) })
			r.Equal(tt.code, rec.Code)
			body := rec.Result().Body
			defer body.Close()
			data, err := io.ReadAll(body)
			r.NoError(err)
			r.Equal(tt.body+"\n", string(data))
		})
	}
}
