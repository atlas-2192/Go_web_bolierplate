package view

import (
	"context"
	"errors"
	"fmt"
)

const EnvironmentCtxKey = "view.Environment"

type Environment struct {
	AssetManifest map[string]string
	ViteHotReload bool
}

func assetPath(ctx context.Context, name string) (string, error) {
	env := ctx.Value(EnvironmentCtxKey).(*Environment)

	if env.AssetManifest == nil {
		return fmt.Sprintf("/assets/%s", name), nil
	}

	path, ok := env.AssetManifest[name]
	if !ok {
		return "", fmt.Errorf("asset not found: %s", name)
	}

	return path, nil
}

func viteHotReload(ctx context.Context) bool {
	env := ctx.Value(EnvironmentCtxKey).(*Environment)
	return env.ViteHotReload
}

func csrfToken(ctx context.Context) (string, error) {
	token, ok := ctx.Value("gorilla.csrf.Token").(string)
	if !ok {
		return "", errors.New("missing gorilla.csrf.Token in context")
	}
	return token, nil
}
