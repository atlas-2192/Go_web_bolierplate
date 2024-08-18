package structx_test

import (
	"testing"

	"github.com/jackc/web-starter-app/lib/structx"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	value := struct {
		A string
		B int
	}{
		A: "hello",
		B: 42,
	}

	require.Equal(t, "hello", structx.Get(value, "A"))
	require.Equal(t, 42, structx.Get(value, "B"))
}
