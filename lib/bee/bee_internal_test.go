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
			parts:     nil,
			errStr:    "paramName must not be empty",
		},
		{
			testName:  "top level key",
			paramName: "foo",
			parts:     []string{"foo"},
			errStr:    "",
		},
		{
			testName:  "top level array key",
			paramName: "foo[]",
			parts:     []string{"foo", paramNameArrayPart},
			errStr:    "",
		},
		{
			testName:  "nested attribute",
			paramName: "foo[bar]",
			parts:     []string{"foo", "bar"},
			errStr:    "",
		},
		{
			testName:  "double nested attribute",
			paramName: "foo[bar][baz]",
			parts:     []string{"foo", "bar", "baz"},
			errStr:    "",
		},
		{
			testName:  "double nested attribute array",
			paramName: "foo[bar][baz][]",
			parts:     []string{"foo", "bar", "baz", paramNameArrayPart},
			errStr:    "",
		},
		{
			testName:  "nested array of object",
			paramName: "foo[bar][][baz]",
			parts:     nil,
			errStr:    "paramName array part must be last element",
		},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			parts, err := splitParamName(tc.paramName)
			if tc.errStr == "" {
				require.NoError(t, err)
				require.Equal(t, tc.parts, parts)
			} else {
				require.EqualError(t, err, tc.errStr)
			}
		})
	}
}
