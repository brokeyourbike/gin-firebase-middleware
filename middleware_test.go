package ginfirebasemw_test

import (
	_ "embed"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	ginfirebasemw "github.com/brokeyourbike/gin-firebase-middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

//go:embed testdata/no-required-fields.json
var noRequiredFields []byte

//go:embed testdata/no-second-factor.json
var noSecondFactor []byte

//go:embed testdata/second-factor-phone.json
var secondFactorPhone []byte

//go:embed testdata/service-account.json
var serviceAccount []byte

func TestMain(m *testing.M) {
	gin.SetMode(gin.ReleaseMode)
	os.Exit(m.Run())
}

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name        string
		headers     map[string]string
		wantStatus  int
		assertInCtx gin.HandlerFunc
	}{
		{
			"missing header",
			map[string]string{},
			http.StatusForbidden,
			func(ctx *gin.Context) { ctx.Status(http.StatusOK) },
		},
		{
			"header is not base64 encoded",
			map[string]string{"X-Apigateway-Api-Userinfo": "not-base64-encoded"},
			http.StatusForbidden,
			func(ctx *gin.Context) { ctx.Status(http.StatusOK) },
		},
		{
			"encoded value is not json",
			map[string]string{"X-Apigateway-Api-Userinfo": base64.RawURLEncoding.EncodeToString([]byte("I am not JSON"))},
			http.StatusForbidden,
			func(ctx *gin.Context) { ctx.Status(http.StatusOK) },
		},
		{
			"json does not contain required fields",
			map[string]string{"X-Apigateway-Api-Userinfo": base64.RawURLEncoding.EncodeToString(noRequiredFields)},
			http.StatusForbidden,
			func(ctx *gin.Context) { ctx.Status(http.StatusOK) },
		},
		{
			"no second factor",
			map[string]string{"X-Apigateway-Api-Userinfo": base64.RawURLEncoding.EncodeToString(noSecondFactor)},
			http.StatusOK,
			func(ctx *gin.Context) {
				info := ginfirebasemw.GetUserInfo(ctx)
				assert.Equal(t, "83ffc78a-6457-4103-9912-ac070fbb6151", info.Sub)
				assert.Equal(t, "john@doe.com", info.Email)
				assert.Equal(t, ginfirebasemw.ProviderPassword, info.Firebase.SignInProvider)
				assert.False(t, info.EmailVerified)
				assert.False(t, info.IsServiceAccount())

				id := ginfirebasemw.GetUserID(ctx)
				assert.Equal(t, "83ffc78a-6457-4103-9912-ac070fbb6151", id)

				ctx.Status(http.StatusOK)
			},
		},
		{
			"second factor phone",
			map[string]string{"X-Apigateway-Api-Userinfo": base64.RawURLEncoding.EncodeToString(secondFactorPhone)},
			http.StatusOK,
			func(ctx *gin.Context) {
				info := ginfirebasemw.GetUserInfo(ctx)
				assert.Equal(t, "83ffc78a-6457-4103-9912-ac070fbb6151", info.Sub)
				assert.Equal(t, "john@doe.com", info.Email)
				assert.Equal(t, ginfirebasemw.SecondFactorPhone, info.Firebase.SignInSecondFactor)
				assert.Equal(t, ginfirebasemw.ProviderPassword, info.Firebase.SignInProvider)
				assert.True(t, info.EmailVerified)
				assert.False(t, info.IsServiceAccount())

				id := ginfirebasemw.GetUserID(ctx)
				assert.Equal(t, "83ffc78a-6457-4103-9912-ac070fbb6151", id)

				ctx.Status(http.StatusOK)
			},
		},
		{
			"service account",
			map[string]string{"X-Apigateway-Api-Userinfo": base64.RawURLEncoding.EncodeToString(serviceAccount)},
			http.StatusOK,
			func(ctx *gin.Context) {
				info := ginfirebasemw.GetUserInfo(ctx)
				assert.Equal(t, "ab0b166e-c725-4921-b919-fd1cbf43a442", info.Sub)
				assert.Equal(t, "john@example.iam.gserviceaccount.com", info.Email)
				assert.Equal(t, "", info.Firebase.SignInSecondFactor)
				assert.Equal(t, "", info.Firebase.SignInProvider)
				assert.False(t, info.EmailVerified)
				assert.True(t, info.IsServiceAccount())

				id := ginfirebasemw.GetUserID(ctx)
				assert.Equal(t, "ab0b166e-c725-4921-b919-fd1cbf43a442", id)

				ctx.Status(http.StatusOK)
			},
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
			router.GET("/", test.assertInCtx)
			router.ServeHTTP(w, req)

			assert.Equal(t, test.wantStatus, w.Code)
		})
	}
}
