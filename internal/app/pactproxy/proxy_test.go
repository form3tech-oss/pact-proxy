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
				interactions.Store(newInteraction("existing"))
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

func newInteraction(alias string) *Interaction {
	i := &Interaction{
		Alias:       alias,
		Description: alias,
		constraints: map[string]interactionConstraint{},
	}
	i.modifiers = interactionModifiers{
		interaction: i,
		modifiers:   map[string]*interactionModifier{},
	}
	return i
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
		code         int
		body         string
	}{
		{
			name: "interaction found - without request history",
			interactions: func() *Interactions {
				interactions := Interactions{}
				request := map[string]interface{}{
					"body": map[string]interface{}{"foo": "bar"},
					"path": "/testpath",
				}
				i := newInteraction("test")
				i.StoreRequest(request)
				interactions.Store(i)
				return &interactions
			}(),
			code: http.StatusOK,
			// request_count is 0 as the interaction has not been matched yet
			body: `{"method":"","alias":"test","description":"test","request_count":0,"last_request":{"body":{"foo":"bar"},"path":"/testpath"}}`,
		},
		{
			name: "interaction found - with request history",
			interactions: func() *Interactions {
				interactions := Interactions{}
				request := map[string]interface{}{
					"body": map[string]interface{}{"foo": "bar"},
					"path": "/testpath",
				}
				i := newInteraction("test")
				i.recordHistory = true
				i.StoreRequest(request)
				interactions.Store(i)
				return &interactions
			}(),
			code: http.StatusOK,
			// request_count is 0 as the interaction has not been matched yet
			body: `{"method":"","alias":"test","description":"test","request_count":0,"request_history":[{"body":{"foo":"bar"},"path":"/testpath"}],"last_request":{"body":{"foo":"bar"},"path":"/testpath"}}`,
		},
		{
			name:         "interaction not found",
			interactions: &Interactions{},
			code:         http.StatusNotFound,
			body:         `{"error_message":"interaction \"test\" not found"}`,
		},
	} {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/interactions/test", nil)
			e := echo.New()
			c := e.NewContext(req, rec)
			c.SetParamNames("alias")
			c.SetParamValues("test")
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
