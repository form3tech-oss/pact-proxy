package pactproxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
)

func TestInteractionsWaitHandler(t *testing.T) {
	r := require.New(t)
	cc := &ProxyContext{
		// target:       target,
		// proxy:        proxy,
		interactions: &Interactions{},
		notify:       NewNotify(),
		delay:        20 * time.Millisecond,
		duration:     150 * time.Millisecond,
	}

	// api := api{
	// 	interactions: &Interactions{},
	// 	notify:       NewNotify(),
	// 	delay:        20 * time.Millisecond,
	// 	duration:     150 * time.Millisecond,
	// }

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
			cc.Context = e.NewContext(tt.req, rec)
			cc.interactions = tt.interactions

			r.NotPanics(func() { interactionsWaitHandler(cc) })
			r.Equal(tt.code, rec.Code)
		})
	}
}
