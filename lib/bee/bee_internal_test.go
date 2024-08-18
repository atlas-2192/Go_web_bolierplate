package bee

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSplitParamName(t *testing.T) {
	for _, tc := range []struct {
		testName  string
		paramName string
		parts     []string
		errStr    string
	}{
		{
			testName:  "empty",
			paramName: "",
			parts:     []string{""},
		},
		{
			testName:  "top level key",
			paramName: "foo",
			parts:     []string{"foo"},
		},
		{
			testName:  "top level array key",
			paramName: "foo[]",
			parts:     []string{"foo", paramNameArrayPart},
		},
		{
			testName:  "nested attribute",
			paramName: "foo[bar]",
			parts:     []string{"foo", "bar"},
		},
		{
			testName:  "double nested attribute",
			paramName: "foo[bar][baz]",
			parts:     []string{"foo", "bar", "baz"},
		},
		{
			testName:  "double nested attribute array",
			paramName: "foo[bar][baz][]",
			parts:     []string{"foo", "bar", "baz", paramNameArrayPart},
		},
		{
			testName:  "nested array of object is invalid",
			paramName: "foo[bar][][baz]",
			parts:     []string{"foo[bar][][baz]"},
		},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			parts := splitParamName(tc.paramName)
			require.Equal(t, tc.parts, parts)
		})
	}
}
