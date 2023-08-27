package ginfirebasemw_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	ginfirebasemw "github.com/brokeyourbike/gin-firebase-middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.ReleaseMode)
	os.Exit(m.Run())
}

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		wantStatus int
	}{
		{
			"missing header",
			map[string]string{},
			http.StatusForbidden,
		},
		{
			"header is not base64 encoded",
			map[string]string{"X-Apigateway-Api-Userinfo": "not-base64-encoded"},
			http.StatusForbidden,
		},
		{
			"encoded value is not json",
			map[string]string{"X-Apigateway-Api-Userinfo": "SSBhbSBub3QgSlNPTg=="}, // I am not JSON
			http.StatusForbidden,
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			for k, v := range test.headers {
				req.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			router := gin.New()
			router.Use(ginfirebasemw.Middleware())
			router.GET("/", func(ctx *gin.Context) {
				info := ginfirebasemw.GetUserInfo(ctx)
				assert.Equal(t, "c2133353-6547-4429-a453-4c8fa2fdbacd", info.Sub)

				id := ginfirebasemw.GetUserID(ctx)
				assert.Equal(t, "c2133353-6547-4429-a453-4c8fa2fdbacd", id)

				ctx.String(http.StatusOK, "the end.")
			})
			router.ServeHTTP(w, req)

			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
