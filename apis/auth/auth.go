package auth

import (
	errorCheck "errors"
	"os"

	"github.com/HezerSantos/alzher-api/common/argon"
	"github.com/HezerSantos/alzher-api/common/errorfuncs"
	"github.com/HezerSantos/alzher-api/common/jwt"
	"github.com/HezerSantos/alzher-api/services/railway"
	"github.com/HezerSantos/alzher-api/services/railway/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type VerifyUserJSON struct {
	Email    string `json:"email" binding:"email,required"`
	Password string `json:"password" binding:"required"`
}

func GetAuthTokenHandler(ginCtx *gin.Context) {
	var verifyUserJson VerifyUserJSON
	err := ginCtx.ShouldBind(&verifyUserJson)

	if err != nil {
		errorfuncs.ErrorHelper(
			ginCtx,
			errorfuncs.JsonError{
				Message: "JSON ERROR 001",
				Status:  400,
				Json:    errorfuncs.JsonResponseType{Code: "INVALID_BODY", Msg: "JSON ERROR 001"},
			},
		)
		return
	}

	var user models.User

	result := railway.DB.Model(&models.User{}).Where("email = ?", verifyUserJson.Email).First(&user)

	//Error if record not found
	if errorCheck.Is(result.Error, gorm.ErrRecordNotFound) {
		errorfuncs.ErrorHelper(
			ginCtx,
			errorfuncs.JsonError{
				Message: "User not found (Email)",
				Status:  404,
				Json:    errorfuncs.JsonResponseType{Code: "INVALID_USER", Msg: "User not found"},
			},
		)
		return
	}

	//Network error
	if result.Error != nil {
		errorfuncs.NetworkError(ginCtx, result.Error)
		return
	}

	//Verify the password
	passwordResult, err := argon.ComparePasswordAndHash(verifyUserJson.Password, user.Password)

	if err != nil {
		errorfuncs.NetworkError(ginCtx, err)
		return
	}

	if !passwordResult {
		errorfuncs.ErrorHelper(
			ginCtx,
			errorfuncs.JsonError{
				Message: "User not found (Passwords)",
				Status:  404,
				Json:    errorfuncs.JsonResponseType{Code: "INVALID_USER", Msg: "User not found"},
			},
		)
		return
	}

	jwtToken, err := jwt.GenerateUserJWT(user.ID, user.Email, 1)

	if err != nil {
		errorfuncs.NetworkError(ginCtx, err)
		return
	}

	domain := ""

	if os.Getenv("GO_ENV") == "production" {
		domain = ".hallowedvisions.com"
	}

	ginCtx.SetCookie(
		"__Secure-secure-auth.access",
		jwtToken,
		60*1000*60,
		"/",
		domain,
		true,
		true,
	)

	ginCtx.JSON(200, gin.H{"msg": "User Logged In"})
}
