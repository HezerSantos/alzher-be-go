package settings

import (
	"net/http"

	"github.com/HezerSantos/alzher-api/common/api"
	"github.com/HezerSantos/alzher-api/common/api/types"
	"github.com/HezerSantos/alzher-api/common/errorfuncs"
	"github.com/HezerSantos/alzher-api/common/userinfo"
	"github.com/HezerSantos/alzher-api/services/railway"
	"github.com/HezerSantos/alzher-api/services/railway/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
