package lugo

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"
)

// TypeConverter provides methods for safe type conversion
type TypeConverter struct{}

// ToString converts any supported value to string
func (tc *TypeConverter) ToString(v interface{}) (string, error) {
	if v == nil {
		return "", nil
	}

	switch val := v.(type) {
	case string:
		return val, nil
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", val), nil
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", val), nil
	case float32, float64:
		return fmt.Sprintf("%g", val), nil
	case bool:
		return strconv.FormatBool(val), nil
	case time.Time:
		return val.Format(time.RFC3339), nil
	case fmt.Stringer:
		return val.String(), nil
	default:
		return "", &Error{
			Code:    ErrInvalidType,
			Message: fmt.Sprintf("cannot convert type %T to string", v),
		}
	}
}

// ToInt converts any supported value to int64
func (tc *TypeConverter) ToInt(v interface{}) (int64, error) {
	if v == nil {
		return 0, nil
	}

	switch val := v.(type) {
	case int:
		return int64(val), nil
	case int8:
		return int64(val), nil
	case int16:
		return int64(val), nil
	case int32:
		return int64(val), nil
	case int64:
		return val, nil
	case uint:
		return int64(val), nil
	case uint8:
		return int64(val), nil
	case uint16:
		return int64(val), nil
	case uint32:
		return int64(val), nil
	case uint64:
		if val > math.MaxInt64 {
			return 0, &Error{
				Code:    ErrInvalidType,
				Message: "uint64 value overflows int64",
			}
		}
		return int64(val), nil
	case float32:
		return int64(val), nil
	case float64:
		return int64(val), nil
	case string:
		i, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return 0, &Error{
				Code:    ErrInvalidType,
				Message: fmt.Sprintf("cannot convert string '%s' to int", val),
				Cause:   err,
			}
		}
		return i, nil
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, &Error{
			Code:    ErrInvalidType,
			Message: fmt.Sprintf("cannot convert type %T to int", v),
		}
	}
}

// ToFloat converts any supported value to float64
func (tc *TypeConverter) ToFloat(v interface{}) (float64, error) {
	if v == nil {
		return 0, nil
	}

	switch val := v.(type) {
	case float32:
		return float64(val), nil
	case float64:
		return val, nil
	case int, int8, int16, int32, int64:
		return float64(reflect.ValueOf(val).Int()), nil
	case uint, uint8, uint16, uint32, uint64:
		return float64(reflect.ValueOf(val).Uint()), nil
	case string:
		f, err := strconv.ParseFloat(val, 64)
		if err != nil {
			return 0, &Error{
				Code:    ErrInvalidType,
				Message: fmt.Sprintf("cannot convert string '%s' to float", val),
				Cause:   err,
			}
		}
		return f, nil
	case bool:
		if val {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, &Error{
			Code:    ErrInvalidType,
			Message: fmt.Sprintf("cannot convert type %T to float", v),
		}
	}
}

// ToBool converts any supported value to bool
func (tc *TypeConverter) ToBool(v interface{}) (bool, error) {
	if v == nil {
		return false, nil
	}

	switch val := v.(type) {
	case bool:
		return val, nil
	case int, int8, int16, int32, int64:
		return reflect.ValueOf(val).Int() != 0, nil
	case uint, uint8, uint16, uint32, uint64:
		return reflect.ValueOf(val).Uint() != 0, nil
	case float32:
		return val != 0, nil
	case float64:
		return val != 0, nil
	case string:
		b, err := strconv.ParseBool(val)
		if err != nil {
			// Handle special string cases
			switch val {
			case "yes", "y", "1", "on":
				return true, nil
			case "no", "n", "0", "off":
				return false, nil
			default:
				return false, &Error{
					Code:    ErrInvalidType,
					Message: fmt.Sprintf("cannot convert string '%s' to bool", val),
					Cause:   err,
				}
			}
		}
		return b, nil
	default:
		return false, &Error{
			Code:    ErrInvalidType,
			Message: fmt.Sprintf("cannot convert type %T to bool", v),
		}
	}
}
