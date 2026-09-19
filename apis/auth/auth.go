package auth

import (
	errorCheck "errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/HezerSantos/alzher-api/apis/dashboard/settings"
	"github.com/HezerSantos/alzher-api/common/api"
	"github.com/HezerSantos/alzher-api/common/api/types"
	"github.com/HezerSantos/alzher-api/common/argon"
	"github.com/HezerSantos/alzher-api/common/errorfuncs"
	"github.com/HezerSantos/alzher-api/common/jwt"
	"github.com/HezerSantos/alzher-api/services/railway"
	"github.com/HezerSantos/alzher-api/services/railway/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VerifyUserJSON struct {
	Email    string `json:"email" binding:"email,required"`
	Password string `json:"password" binding:"required"`
}

func GetAuthTokenHandler(ginCtx *gin.Context) {
	crc, err := api.GetCallResultContainerContext(ginCtx.Request.Context())

	if err != nil {
		errorfuncs.NetworkError(ginCtx, err, nil)
		return
	}

	var verifyUserJson VerifyUserJSON
	err = ginCtx.ShouldBind(&verifyUserJson)

	if err != nil {
		crc.Add("ginCtx: ShouldBind()", nil, http.StatusBadRequest, err)
		errorfuncs.BadRequestError(ginCtx, crc)
		return
	}

	var user models.User

	result := railway.DB.Model(&models.User{}).Where("email = ?", verifyUserJson.Email).First(&user)

	//Error if record not found
	if errorCheck.Is(result.Error, gorm.ErrRecordNotFound) {
		crc.Add("Railway: queryUserByEmail()", nil, http.StatusBadRequest, result.Error)
		errorfuncs.BadRequestError(ginCtx, crc)
		return
	}

	//Network error
	if result.Error != nil {
		crc.Add("Railway: queryUserByEmail()", nil, http.StatusInternalServerError, result.Error)
		errorfuncs.NetworkError(ginCtx, nil, crc)
		return
	}

	//Verify the password
	passwordResult, err := argon.ComparePasswordAndHash(verifyUserJson.Password, user.Password)

	if err != nil {
		crc.Add("Argon: ComparePasswordAndHash()", nil, http.StatusInternalServerError, err)
		errorfuncs.NetworkError(ginCtx, nil, crc)
		return
	}

	if !passwordResult {
		errorfuncs.UnauthorizedError(ginCtx)
		return
	}

	jwtToken, err := jwt.GenerateUserJWT(user.ID, user.Email, 1)

	if err != nil {
		crc.Add("JWT: GenerateUserJWT()", nil, http.StatusInternalServerError, err)
		errorfuncs.NetworkError(ginCtx, nil, crc)
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

	ginCtx.JSON(200, gin.H{"message": "User Logged In"})
}

type CreateUserRequestBody struct {
	Email           string `json:"email" binding:"email,required"`
	Password        string `json:"password" binding:"required"`
	ConfirmPassword string `json:"confirmPassword" binding:"required"`
}

type CreateUserResponse struct {
	Message     *string            `json:"message"`
	User        *models.User       `json:"user"`
	CallResults []types.CallResult `json:"callResults"`
}

func createUserQuery(email string, passwordHash string) (*models.User, error) {
	uuid, err := uuid.NewV6()

	if err != nil {
		return nil, err
	}

	newUser := models.User{
		ID:        uuid,
		Email:     email,
		Password:  passwordHash,
		CreatedAt: time.Now(),
	}

	err = railway.DB.Create(&newUser).Error

	if err != nil {
		return nil, err
	}

	return &newUser, nil
}

func CreateUserHandler(ginCtx *gin.Context) {
	crc, err := api.GetCallResultContainerContext(ginCtx.Request.Context())

	if err != nil {
		ginCtx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var requestBody CreateUserRequestBody

	if err := ginCtx.ShouldBindJSON(&requestBody); err != nil {
		crc.Add("CreateUserHandler: ShouldBindJSON()", nil, http.StatusInternalServerError, err)
		ginCtx.JSON(http.StatusBadRequest, CreateUserResponse{CallResults: crc.CallResults})
		return
	}

	rowsAffected, err := settings.QueryUserByEmail(requestBody.Email)

	if err != nil {
		crc.Add("Railway: QueryUserByEmail()", nil, http.StatusInternalServerError, err)

		ginCtx.JSON(http.StatusInternalServerError, CreateUserResponse{CallResults: crc.CallResults})
		return
	}

	crc.Add("Railway: QueryUserByEmail()", gin.H{"Rows Affected": *rowsAffected}, http.StatusOK, nil)

	if *rowsAffected >= 1 {
		msg := "Email Already Exists"
		ginCtx.JSON(http.StatusForbidden, CreateUserResponse{Message: &msg, CallResults: crc.CallResults})
		return
	}

	if requestBody.Password != requestBody.ConfirmPassword {
		crc.Add("CreateUserHandler", nil, http.StatusBadRequest, fmt.Errorf("Passwords Do Not Match"))
		ginCtx.JSON(http.StatusBadRequest, CreateUserResponse{CallResults: crc.CallResults})
		return
	}

	passwordHash, err := argon.HashPassword(requestBody.Password)

	if err != nil {
		crc.Add("CreateUserHandler: HashPassword()", nil, http.StatusInternalServerError, err)
		ginCtx.JSON(http.StatusInternalServerError, CreateUserResponse{CallResults: crc.CallResults})
		return
	}

	newUser, err := createUserQuery(requestBody.Email, passwordHash)

	if err != nil {
		crc.Add("Railway: createUserQuery()", nil, http.StatusInternalServerError, err)
		ginCtx.JSON(http.StatusInternalServerError, CreateUserResponse{CallResults: crc.CallResults})
		return
	}

	crc.Add("Railway: createUserQuery()", &newUser, http.StatusOK, nil)

	msg := "User Created"
	ginCtx.JSON(http.StatusOK, CreateUserResponse{Message: &msg, User: newUser, CallResults: crc.CallResults})

}

func LogoutUserHandler(ginCtx *gin.Context) {
	domain := ""

	if os.Getenv("GO_ENV") == "production" {
		domain = ".hallowedvisions.com"
	}

	ginCtx.SetCookie(
		"__Secure-secure-auth.access",
		"",
		-1,
		"/",
		domain,
		true,
		true,
	)

	ginCtx.JSON(200, gin.H{"message": "Logged out successfully"})
}
