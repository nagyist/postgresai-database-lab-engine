package srv

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomOptions(t *testing.T) {
	testCases := []struct {
		customOptions  []interface{}
		expectedResult error
	}{
		{
			customOptions:  []interface{}{"--verbose"},
			expectedResult: nil,
		},
		{
			customOptions:  []interface{}{"--exclude-scheme=test_scheme"},
			expectedResult: nil,
		},
		{
			customOptions:  []interface{}{`--exclude-scheme="test_scheme"`},
			expectedResult: nil,
		},
		{
			customOptions:  []interface{}{"--table=$(echo 'test')"},
			expectedResult: errInvalidOption,
		},
		{
			customOptions:  []interface{}{"--table=test&table"},
			expectedResult: errInvalidOption,
		},
		{
			customOptions:  []interface{}{5},
			expectedResult: errInvalidOptionType,
		},
	}

	for _, tc := range testCases {
		validationResult := validateCustomOptions(tc.customOptions)

		require.ErrorIs(t, validationResult, tc.expectedResult)
	}
}

func TestValidateCustomOptions_AdditionalCases(t *testing.T) {
	t.Run("empty options list is valid", func(t *testing.T) {
		assert.NoError(t, validateCustomOptions([]interface{}{}))
	})

	t.Run("nil options list is valid", func(t *testing.T) {
		assert.NoError(t, validateCustomOptions(nil))
	})

	t.Run("multiple valid options", func(t *testing.T) {
		opts := []interface{}{"--verbose", "--jobs=4", "--format=custom"}
		assert.NoError(t, validateCustomOptions(opts))
	})

	t.Run("first valid second invalid stops with error", func(t *testing.T) {
		opts := []interface{}{"--verbose", "--table=$(injection)"}
		err := validateCustomOptions(opts)
		require.Error(t, err)
		assert.ErrorIs(t, err, errInvalidOption)
	})

	t.Run("option with spaces is rejected", func(t *testing.T) {
		opts := []interface{}{"--table=test table"}
		err := validateCustomOptions(opts)
		require.Error(t, err)
		assert.ErrorIs(t, err, errInvalidOption)
	})

	t.Run("option with semicolon is rejected", func(t *testing.T) {
		opts := []interface{}{"--table=test;drop"}
		err := validateCustomOptions(opts)
		require.Error(t, err)
		assert.ErrorIs(t, err, errInvalidOption)
	})

	t.Run("option with pipe is rejected", func(t *testing.T) {
		opts := []interface{}{"--table=test|cat"}
		err := validateCustomOptions(opts)
		require.Error(t, err)
		assert.ErrorIs(t, err, errInvalidOption)
	})

	t.Run("non-string types are rejected", func(t *testing.T) {
		testCases := []struct {
			name string
			opt  interface{}
		}{
			{name: "float", opt: 3.14},
			{name: "bool", opt: true},
			{name: "slice", opt: []string{"a"}},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := validateCustomOptions([]interface{}{tc.opt})
				assert.ErrorIs(t, err, errInvalidOptionType)
			})
		}
	})

	t.Run("valid option characters", func(t *testing.T) {
		validOpts := []interface{}{
			"simple",
			"With-Hyphens",
			"under_scores",
			"equal=sign",
			`with"quotes"`,
			"MixedCase123",
		}
		for _, opt := range validOpts {
			assert.NoError(t, validateCustomOptions([]interface{}{opt}), "option %q should be valid", opt)
		}
	})
}
