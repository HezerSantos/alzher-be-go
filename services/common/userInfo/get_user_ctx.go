package userinfo

import (
	"context"
	"fmt"

	"github.com/HezerSantos/alzher-api/middleware"
	"github.com/HezerSantos/alzher-api/services/railway/models"
)

func GetUserContext(ctx context.Context) (models.User, error) {
	user := ctx.Value(middleware.UserContextKey)

	assertedUser, ok := user.(models.User)

	if !ok {
		return models.User{}, fmt.Errorf("User Context is not of type User")
	}

	return assertedUser, nil
}
