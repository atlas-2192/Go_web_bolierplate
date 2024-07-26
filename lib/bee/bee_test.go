package bee_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/jackc/web-starter-app/lib/bee"
	"github.com/stretchr/testify/require"
)

func TestParseParamsQueryParameters(t *testing.T) {
	queryArgs := url.Values{}
	queryArgs.Add("a", "1")
	queryArgs.Add("b", "2")
	queryArgs.Add("c[]", "3")
	queryArgs.Add("c[]", "4")
	queryArgs.Add("d[e]", "5")
	queryArgs.Add("d[f]", "6")
	queryArgs.Add("g[h][i]", "7")
	queryArgs.Add("g[h][j]", "8")
	queryArgs.Add("k[l][m][]", "9")
	queryArgs.Add("k[l][m][]", "10")
	queryArgs.Add("k[l][n][]", "11")
	queryArgs.Add("k[l][n][]", "12")

	r := httptest.NewRequest("GET", fmt.Sprintf("/somewhere?%s", queryArgs.Encode()), nil)

	params, err := bee.ParseParams(r)
	require.NoError(t, err)

	require.Equal(t,
		map[string]any{
			"a": "1",
			"b": "2",
			"c": []string{"3", "4"},
			"d": map[string]any{"e": "5", "f": "6"},
			"g": map[string]any{"h": map[string]any{"i": "7", "j": "8"}},
			"k": map[string]any{
				"l": map[string]any{
					"m": []string{"9", "10"},
					"n": []string{"11", "12"},
				},
			},
		},
		params,
	)
}

func TestParseParamsFormURLEncoded(t *testing.T) {
	queryArgs := url.Values{}
	queryArgs.Add("a", "1")
	queryArgs.Add("b", "2")
	queryArgs.Add("c[]", "3")
	queryArgs.Add("c[]", "4")
	queryArgs.Add("d[e]", "5")
	queryArgs.Add("d[f]", "6")
	queryArgs.Add("g[h][i]", "7")
	queryArgs.Add("g[h][j]", "8")
	queryArgs.Add("k[l][m][]", "9")
	queryArgs.Add("k[l][m][]", "10")
	queryArgs.Add("k[l][n][]", "11")
	queryArgs.Add("k[l][n][]", "12")

	r := httptest.NewRequest("POST", "/somewhere", strings.NewReader(queryArgs.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	params, err := bee.ParseParams(r)
	require.NoError(t, err)

	require.Equal(t,
		map[string]any{
			"a": "1",
			"b": "2",
			"c": []string{"3", "4"},
			"d": map[string]any{"e": "5", "f": "6"},
			"g": map[string]any{"h": map[string]any{"i": "7", "j": "8"}},
			"k": map[string]any{
				"l": map[string]any{
					"m": []string{"9", "10"},
					"n": []string{"11", "12"},
				},
			},
		},
		params,
	)
}

func TestParseParamsApplicationJSON(t *testing.T) {
	postData := map[string]any{"a": "1", "b": "2"}
	postBody, err := json.Marshal(postData)
	require.NoError(t, err)

	r := httptest.NewRequest("POST", "/somewhere", bytes.NewReader(postBody))
	r.Header.Set("Content-Type", "application/json")

	params, err := bee.ParseParams(r)
	require.NoError(t, err)

	require.Equal(t, postData, params)
}

// bee.Must(err, 500) ?

func TestHandlerBuilderHandlerSetsEtag(t *testing.T) {
	hb := &bee.HandlerBuilder[struct{}]{}
	handler := hb.New(func(ctx context.Context, w http.ResponseWriter, r *http.Request, _ struct{}, params map[string]any) error {
		w.Write([]byte("Hello, world"))
		return nil
	})

	r := httptest.NewRequest("GET", "/", nil)
	responseRecorder := httptest.NewRecorder()
	handler.ServeHTTP(responseRecorder, r)

	require.Equal(t, "Hello, world", responseRecorder.Body.String())
	require.Equal(t, `W/"SufDtqwL7_Zx76jPVzhhUcBuWMpTp42D82EHMWzsEl8="`, responseRecorder.Header().Get("ETag"))
}
