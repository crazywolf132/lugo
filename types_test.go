package lugo

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTypeConverter_ToString(t *testing.T) {
	tc := &TypeConverter{}
	tests := []struct {
		name     string
		input    interface{}
		expected string
		wantErr  bool
	}{
		{"nil value", nil, "", false},
		{"string value", "test", "test", false},
		{"int value", 42, "42", false},
		{"uint value", uint64(42), "42", false},
		{"float value", 3.14, "3.14", false},
		{"bool true", true, "true", false},
		{"bool false", false, "false", false},
		{"time value", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), "2024-01-01T00:00:00Z", false},
		{"complex type", complex(1, 2), "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tc.ToString(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.True(t, IsErrorCode(err, ErrInvalidType))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestTypeConverter_ToInt(t *testing.T) {
	tc := &TypeConverter{}
	tests := []struct {
		name     string
		input    interface{}
		expected int64
		wantErr  bool
	}{
		{"nil value", nil, 0, false},
		{"int value", 42, 42, false},
		{"int8 value", int8(42), 42, false},
		{"int16 value", int16(42), 42, false},
		{"int32 value", int32(42), 42, false},
		{"int64 value", int64(42), 42, false},
		{"uint value", uint(42), 42, false},
		{"uint64 max value", uint64(math.MaxInt64), math.MaxInt64, false},
		{"uint64 overflow", uint64(math.MaxInt64 + 1), 0, true},
		{"float32 value", float32(42.0), 42, false},
		{"float64 value", float64(42.0), 42, false},
		{"string valid", "42", 42, false},
		{"string invalid", "not a number", 0, true},
		{"bool true", true, 1, false},
		{"bool false", false, 0, false},
		{"complex type", complex(1, 2), 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tc.ToInt(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.True(t, IsErrorCode(err, ErrInvalidType))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestTypeConverter_ToFloat(t *testing.T) {
	tc := &TypeConverter{}
	tests := []struct {
		name     string
		input    interface{}
		expected float64
		wantErr  bool
	}{
		{"nil value", nil, 0, false},
		{"float32 value", float32(3.14), 3.14, false},
		{"float64 value", float64(3.14), 3.14, false},
		{"int value", 42, 42.0, false},
		{"int64 value", int64(42), 42.0, false},
		{"uint value", uint(42), 42.0, false},
		{"uint64 value", uint64(42), 42.0, false},
		{"string valid", "3.14", 3.14, false},
		{"string invalid", "not a number", 0, true},
		{"bool true", true, 1.0, false},
		{"bool false", false, 0.0, false},
		{"complex type", complex(1, 2), 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tc.ToFloat(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.True(t, IsErrorCode(err, ErrInvalidType))
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tt.expected, result, 0.0001)
			}
		})
	}
}

func TestTypeConverter_ToBool(t *testing.T) {
	tc := &TypeConverter{}
	tests := []struct {
		name     string
		input    interface{}
		expected bool
		wantErr  bool
	}{
		{"nil value", nil, false, false},
		{"bool true", true, true, false},
		{"bool false", false, false, false},
		{"int zero", 0, false, false},
		{"int non-zero", 1, true, false},
		{"int64 zero", int64(0), false, false},
		{"int64 non-zero", int64(1), true, false},
		{"uint zero", uint(0), false, false},
		{"uint non-zero", uint(1), true, false},
		{"float32 zero", float32(0), false, false},
		{"float32 non-zero", float32(1), true, false},
		{"float64 zero", float64(0), false, false},
		{"float64 non-zero", float64(1), true, false},
		{"string true", "true", true, false},
		{"string false", "false", false, false},
		{"string yes", "yes", true, false},
		{"string no", "no", false, false},
		{"string y", "y", true, false},
		{"string n", "n", false, false},
		{"string 1", "1", true, false},
		{"string 0", "0", false, false},
		{"string on", "on", true, false},
		{"string off", "off", false, false},
		{"string invalid", "not a bool", false, true},
		{"complex type", complex(1, 2), false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := tc.ToBool(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.True(t, IsErrorCode(err, ErrInvalidType))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
