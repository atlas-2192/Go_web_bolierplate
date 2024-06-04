package browser_test

import (
	"fmt"
	"testing"
)

func TestHello(t *testing.T) {
	t.Parallel()

	// ctx := context.Background()
	serverInstance := startServer(t)
	// db := serverInstance.DB.Connect(t, ctx)
	page := TestBrowserManager.Acquire(t).Page()

	page.MustNavigate(fmt.Sprintf("%s/", serverInstance.Server.URL))

	page.HasContent("div", "Hello, world!")
}
