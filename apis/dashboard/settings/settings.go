package settings

import (
	"errors"
	"net/http"

	"github.com/HezerSantos/alzher-api/common/api"
	"github.com/HezerSantos/alzher-api/common/api/types"
	"github.com/HezerSantos/alzher-api/common/argon"
	"github.com/HezerSantos/alzher-api/common/errorfuncs"
	"github.com/HezerSantos/alzher-api/common/userinfo"
	"github.com/HezerSantos/alzher-api/services/railway"
	"github.com/HezerSantos/alzher-api/services/railway/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func queryUserSettings(id uuid.UUID) (*string, error) {
	var email string

	err := railway.DB.Model(models.User{}).Select(`"email"`).Where(`"id" = ?`, id).First(&email).Error

	if err != nil {
		return nil, err
	}

	return &email, nil
}
func GetSettingsHandler(ginCtx *gin.Context) {
	user, err := userinfo.GetUserContext(ginCtx.Request.Context())

	if err != nil {
		errorfuncs.UnauthorizedError(ginCtx)
		return
	}

	var callResults []types.CallResult
	var email *string

	emailResult, err := queryUserSettings(user.ID)

	if err != nil {
		api.MakeCallResults(&callResults, "Railway: queryUserSettings()", nil, http.StatusInternalServerError, err)
		ginCtx.JSON(http.StatusInternalServerError, gin.H{
			"callResults": callResults,
		})
		return
	}
	api.MakeCallResults(&callResults, "Railway: queryUserSettings()", emailResult, http.StatusOK, nil)
	email = emailResult

	ginCtx.JSON(http.StatusOK, gin.H{
		"email":       &email,
		"callResults": callResults,
	})
}

type PatchUserEmailRequestBody struct {
	Email    string `json:"newEmail" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func queryUserByEmail(email string) (*int64, error) {
	var user models.User
	result := railway.DB.Model(&models.User{}).Where(`"email" = ?`, email).First(&user)

	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}

	return &result.RowsAffected, nil
}

func queryUserByID(id uuid.UUID) (*models.User, error) {
	var user models.User

	err := railway.DB.Model(models.User{}).Where(`"id" = ?`, id).First(&user).Error

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func updateUserSettings(user models.User, update map[string]interface{}) (*int64, error) {
	result := railway.DB.Model(user).Updates(update)

	if result.Error != nil {
		return nil, result.Error
	}

	return &result.RowsAffected, nil
}

func PatchUserEmail(ginCtx *gin.Context) {
	user, err := userinfo.GetUserContext(ginCtx.Request.Context())

	if err != nil {
		errorfuncs.UnauthorizedError(ginCtx)
		return
	}

	var requestBody PatchUserEmailRequestBody

	err = ginCtx.ShouldBindJSON(&requestBody)

	if err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	var callResults []types.CallResult
	rowsAffected, err := queryUserByEmail(requestBody.Email)

	if err != nil {
		api.MakeCallResults(&callResults, "Railway: queryUserByEmail()", nil, http.StatusInternalServerError, err)

		ginCtx.JSON(http.StatusInternalServerError, gin.H{
			"callResults": callResults,
		})
		return
	}

	api.MakeCallResults(&callResults, "Railway: queryUserByEmail()", gin.H{"Rows Affected": *rowsAffected}, http.StatusOK, nil)
	if *rowsAffected >= 1 {
		ginCtx.JSON(http.StatusForbidden, gin.H{
			"error":       "Email Already Exists",
			"callResults": callResults,
		})
		return
	}

	queriedUser, err := queryUserByID(user.ID)

	if err != nil {
		api.MakeCallResults(&callResults, "Railway: queryUserByID()", nil, http.StatusInternalServerError, err)
		ginCtx.JSON(http.StatusInternalServerError, gin.H{
			"callResults": callResults,
		})
		return
	}

	api.MakeCallResults(&callResults, "Railway: queryUserByID()", *queriedUser, http.StatusOK, nil)

	match, err := argon.ComparePasswordAndHash(requestBody.Password, queriedUser.Password)

	if err != nil {
		api.MakeCallResults(&callResults, "Argon: ComparePasswordAndHash()", nil, http.StatusInternalServerError, err)
		ginCtx.JSON(http.StatusInternalServerError, gin.H{
			"callResults": callResults,
		})
		return
	}
	if match == false {
		ginCtx.JSON(http.StatusForbidden, gin.H{
			"error":       "Unable to Update Email",
			"callResults": callResults,
		})
		return
	}

	updates := map[string]interface{}{
		"email": requestBody.Email,
	}

	rowsAffected, err = updateUserSettings(user, updates)

	if err != nil {
		api.MakeCallResults(&callResults, "Railway: updateUserSettings()", nil, http.StatusInternalServerError, err)
		ginCtx.JSON(http.StatusInternalServerError, gin.H{
			"callResults": callResults,
		})
		return
	}

	api.MakeCallResults(&callResults, "Railway: updateUserSettings()", gin.H{"Rows Affected": *rowsAffected}, http.StatusOK, nil)

	ginCtx.JSON(http.StatusOK, gin.H{
		"callResults": callResults,
	})
}

type PatchUserPasswordRequestBody struct {
	CurrentPassword string `json:"currentPassword" binding:"required"`
	Password        string `json:"password" binding:"required"`
	ConfirmPassword string `json:"confirmPassword" binding:"required,min=3"`
}

func PatchUserPassword(ginCtx *gin.Context) {
	user, err := userinfo.GetUserContext(ginCtx.Request.Context())

	if err != nil {
		errorfuncs.UnauthorizedError(ginCtx)
		return
	}

	var requestBody PatchUserPasswordRequestBody

	err = ginCtx.ShouldBindJSON(&requestBody)

	if err != nil {
		ginCtx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	if requestBody.Password != requestBody.ConfirmPassword {
		ginCtx.JSON(http.StatusBadRequest, gin.H{
			"error": "Paswords Do Not Match",
		})
		return
	}

	var callResults []types.CallResult
	queriedUser, err := queryUserByID(user.ID)

	if err != nil {
		api.MakeCallResults(&callResults, "Railway queryUserByID()", nil, http.StatusInternalServerError, err)
		ginCtx.JSON(http.StatusInternalServerError, gin.H{
			"callResults": callResults,
		})
		return
	}
	api.MakeCallResults(&callResults, "Railway: queryUserByID()", *queriedUser, http.StatusOK, nil)

	match, err := argon.ComparePasswordAndHash(requestBody.CurrentPassword, queriedUser.Password)

	if err != nil {
		api.MakeCallResults(&callResults, "Argon: ComparePasswordAndHash()", nil, http.StatusInternalServerError, err)
		ginCtx.JSON(http.StatusInternalServerError, gin.H{
			"callResults": callResults,
		})
		return
	}

	if match == false {
		ginCtx.JSON(http.StatusUnauthorized, gin.H{
			"error":       "Unable to Update Password",
			"callResults": callResults,
		})
		return
	}

	hashedPassword, err := argon.HashPassword(requestBody.Password)

	if err != nil {
		api.MakeCallResults(&callResults, "Argon: HashPassword()", nil, http.StatusInternalServerError, err)
		ginCtx.JSON(http.StatusInternalServerError, gin.H{
			"callResults": callResults,
		})
		return
	}

	updates := map[string]interface{}{
		"password": hashedPassword,
	}

	rowsAffected, err := updateUserSettings(user, updates)

	if err != nil {
		api.MakeCallResults(&callResults, "Railway: updateUserSettings()", nil, http.StatusInternalServerError, err)
		ginCtx.JSON(http.StatusInternalServerError, gin.H{
			"callResults": callResults,
		})
		return
	}

	api.MakeCallResults(&callResults, "Railway: updateUserSettings()", gin.H{"Rows Affected": *rowsAffected}, http.StatusOK, nil)

	ginCtx.JSON(http.StatusOK, gin.H{
		"callResults": callResults,
	})
}
