package ginfirebasemw

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// gatewayUserInfoHeader is the header that contains the user info.
const gatewayUserInfoHeader = "X-Apigateway-Api-Userinfo"

// userInfoCtx is a context key for the UserInfo.
const userInfoCtx = "FirebaseApiGatewayUserInfo"

const ProviderPassword = "password"
const SecondFactorPhone = "phone"

type UserInfo struct {
	Name          string `json:"name,omitempty"`
	Sub           string `json:"sub" binding:"required"`
	Email         string `json:"email" binding:"required,email"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	Firebase      struct {
		SignInProvider     string `json:"sign_in_provider,omitempty"`
		SignInSecondFactor string `json:"sign_in_second_factor,omitempty"`
	} `json:"firebase,omitempty"`
}

func (u UserInfo) IsServiceAccount() bool {
	return strings.HasSuffix(u.Email, "gserviceaccount.com")
}

func Middleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		encodedUser := ctx.GetHeader(gatewayUserInfoHeader)
		if encodedUser == "" {
			ctx.AbortWithStatus(http.StatusForbidden)
			return
		}

		decodedBytes, err := base64.RawURLEncoding.DecodeString(encodedUser)
		if err != nil {
			ctx.AbortWithStatus(http.StatusForbidden)
			return
		}

		var userInfo UserInfo

		if err := json.NewDecoder(bytes.NewReader(decodedBytes)).Decode(&userInfo); err != nil {
			ctx.AbortWithStatus(http.StatusForbidden)
			return
		}

		if err := binding.Validator.ValidateStruct(&userInfo); err != nil {
			ctx.AbortWithStatus(http.StatusForbidden)
			return
		}

		ctx.Set(userInfoCtx, userInfo)
		ctx.Next()
	}
}

// GetUserInfo returns the firebase user info from the context.
func GetUserInfo(ctx *gin.Context) UserInfo {
	userInfo := ctx.MustGet(userInfoCtx).(UserInfo)
	return userInfo
}

// GetUserID returns the firebase user ID from the context.
func GetUserID(ctx *gin.Context) string {
	userInfo := GetUserInfo(ctx)
	return userInfo.Sub
}
