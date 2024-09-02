package browser_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/jackc/pgxutil"
	"github.com/jackc/web-starter-app/db"
	"github.com/stretchr/testify/require"
)

func TestPasswordLoginSuccess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	serverInstance := startServer(t)
	dbconn := serverInstance.DB.Connect(t, ctx)
	userID := uuid.Must(uuid.NewV7())
	err := pgxutil.InsertRow(ctx, dbconn, "users", map[string]any{"id": userID, "username": "testuser"})
	require.NoError(t, err)

	err = db.SetUserPassword(ctx, dbconn, userID, "password")
	require.NoError(t, err)

	page := TestBrowserManager.Acquire(t).Page()

	page.MustNavigate(fmt.Sprintf("%s/login", serverInstance.Server.URL))

	page.FillIn("input[name=username]", "testuser")
	page.FillIn("input[name=password]", "password")
	page.ClickOn("Login")

	page.HasContent("div", "Hello, testuser!")
}

func TestPasswordLoginFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	serverInstance := startServer(t)
	dbconn := serverInstance.DB.Connect(t, ctx)
	userID := uuid.Must(uuid.NewV7())
	err := pgxutil.InsertRow(ctx, dbconn, "users", map[string]any{"id": userID, "username": "testuser"})
	require.NoError(t, err)

	err = db.SetUserPassword(ctx, dbconn, userID, "password")
	require.NoError(t, err)

	page := TestBrowserManager.Acquire(t).Page()

	page.MustNavigate(fmt.Sprintf("%s/login", serverInstance.Server.URL))

	page.FillIn("input[name=username]", "testuser")
	page.FillIn("input[name=password]", "wrongpassword")
	page.ClickOn("Login")

	page.HasContent("body", "invalid")
}
