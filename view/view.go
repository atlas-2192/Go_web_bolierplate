package view

import (
	"context"
	"errors"
)

func csrfToken(ctx context.Context) (string, error) {
	token, ok := ctx.Value("gorilla.csrf.Token").(string)
	if !ok {
		return "", errors.New("missing gorilla.csrf.Token in context")
	}
	return token, nil
}
