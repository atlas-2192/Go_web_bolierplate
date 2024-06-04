package browser_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/go-rod/rod"
	"github.com/jackc/testdb"
	"github.com/jackc/web-starter-app/db"
	"github.com/jackc/web-starter-app/server"
	"github.com/jackc/web-starter-app/test/testbrowser"
	"github.com/jackc/web-starter-app/test/testutil"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

var concurrentChan chan struct{}
var TestDBManager *testdb.Manager
var baseBrowser *rod.Browser
var TestBrowserManager *testbrowser.Manager

func TestMain(m *testing.M) {
	maxConcurrent := 1
	if n, err := strconv.ParseInt(os.Getenv("MAX_CONCURRENT_BROWSER_TESTS"), 10, 32); err == nil {
		maxConcurrent = int(n)
	}
	if maxConcurrent < 1 {
		fmt.Println("MAX_CONCURRENT_BROWSER_TESTS must be greater than 0")
		os.Exit(1)
	}
	concurrentChan = make(chan struct{}, maxConcurrent)

	TestDBManager = testutil.InitTestDBManager(m)

	var err error
	TestBrowserManager, err = testbrowser.NewManager(testbrowser.ManagerConfig{})
	if err != nil {
		fmt.Println("Failed to initialize TestBrowserManager")
		os.Exit(1)
	}

	baseBrowser = rod.New().MustConnect()

	os.Exit(m.Run())
}

type serverInstanceT struct {
	Server *httptest.Server
	DB     *testdb.DB
}

func startServer(t *testing.T) *serverInstanceT {
	ctx := context.Background()
	tdb := TestDBManager.AcquireDB(t, ctx)

	dbpool := tdb.PoolConnect(t, ctx)
	dbsess := db.NewSession(dbpool)
	handler, err := server.NewServer(
		"127.0.0.1:0",
		dbsess,
		zerolog.Ctx(ctx),
	)
	require.NoError(t, err)

	server := httptest.NewServer(handler)
	t.Cleanup(func() {
		server.Close()
	})

	instance := &serverInstanceT{
		Server: server,
		DB:     tdb,
	}

	return instance
}
