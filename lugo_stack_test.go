package lugo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func TestStackPushPop(t *testing.T) {
	cfg := New()
	defer cfg.Close()

	// Initially stack is empty
	assert.Equal(t, 0, cfg.GetStackSize())

	// Push a string
	err := cfg.PushValue("hello")
	require.NoError(t, err)
	assert.Equal(t, 1, cfg.GetStackSize())

	// Push a number
	err = cfg.PushValue(42)
	require.NoError(t, err)
	assert.Equal(t, 2, cfg.GetStackSize())

	// Push a boolean
	err = cfg.PushValue(true)
	require.NoError(t, err)
	assert.Equal(t, 3, cfg.GetStackSize())

	// Pop each value and verify correctness
	val, err := cfg.PopValue()
	require.NoError(t, err)
	assert.Equal(t, true, val, "Expected boolean true")

	val, err = cfg.PopValue()
	require.NoError(t, err)
	assert.Equal(t, float64(42), val, "Expected number 42 as float64")

	val, err = cfg.PopValue()
	require.NoError(t, err)
	assert.Equal(t, "hello", val, "Expected string 'hello'")

	// Now stack should be empty
	assert.Equal(t, 0, cfg.GetStackSize())

	// Popping from empty stack should error
	_, err = cfg.PopValue()
	assert.Error(t, err, "expected error when popping from empty stack")
}

func TestStackPeek(t *testing.T) {
	cfg := New()
	defer cfg.Close()

	// Push multiple values
	require.NoError(t, cfg.PushValue("alpha"))
	require.NoError(t, cfg.PushValue("beta"))
	require.NoError(t, cfg.PushValue("gamma"))
	assert.Equal(t, 3, cfg.GetStackSize())

	// Peek at top value
	val, err := cfg.PeekValue(1)
	require.NoError(t, err)
	assert.Equal(t, "gamma", val, "Top value should be 'gamma'")

	// Peek one below the top
	val, err = cfg.PeekValue(2)
	require.NoError(t, err)
	assert.Equal(t, "beta", val, "Second from top should be 'beta'")

	// Peek at the bottom value
	val, err = cfg.PeekValue(3)
	require.NoError(t, err)
	assert.Equal(t, "alpha", val, "Bottom value should be 'alpha'")

	// Attempt to peek out of range
	_, err = cfg.PeekValue(4)
	assert.Error(t, err, "Expected error peeking beyond stack size")

	// Verify we haven't modified the stack
	assert.Equal(t, 3, cfg.GetStackSize())
}

func TestClearStack(t *testing.T) {
	cfg := New()
	defer cfg.Close()

	require.NoError(t, cfg.PushValue(100))
	require.NoError(t, cfg.PushValue("data"))
	assert.Equal(t, 2, cfg.GetStackSize())

	// Clear the stack
	cfg.ClearStack()
	assert.Equal(t, 0, cfg.GetStackSize())

	// Push again after clearing
	require.NoError(t, cfg.PushValue("reset"))
	assert.Equal(t, 1, cfg.GetStackSize())
}

func TestGetRawLuaValue(t *testing.T) {
	cfg := New()
	defer cfg.Close()

	// Push raw lua values directly
	cfg.L.Push(lua.LString("test"))
	cfg.L.Push(lua.LNumber(3.14))
	cfg.L.Push(lua.LBool(true))

	assert.Equal(t, 3, cfg.GetStackSize())

	// Get raw value at top (pos = 1)
	lv, err := cfg.GetRawLuaValue(1)
	require.NoError(t, err)
	assert.Equal(t, lua.LTBool, lv.Type(), "Expected top value to be boolean (lua.LBool)")

	// Get raw value one below top (pos = 2)
	lv, err = cfg.GetRawLuaValue(2)
	require.NoError(t, err)
	assert.Equal(t, lua.LTNumber, lv.Type(), "Second from top should be a number")

	// Get raw value at bottom (pos = 3)
	lv, err = cfg.GetRawLuaValue(3)
	require.NoError(t, err)
	assert.Equal(t, lua.LTString, lv.Type(), "Third from top (bottom) should be a string")

	// Out of range
	_, err = cfg.GetRawLuaValue(4)
	assert.Error(t, err, "Expected error when getting raw value out of range")
}

func TestPushLuaValue(t *testing.T) {
	cfg := New()
	defer cfg.Close()

	// Create a Lua table and push it
	table := cfg.L.NewTable()
	table.RawSetString("key", lua.LString("value"))
	cfg.PushLuaValue(table)

	assert.Equal(t, 1, cfg.GetStackSize())

	// Retrieve and check the table
	val, err := cfg.PopValue()
	require.NoError(t, err)

	// val should be a map[string]interface{} after conversion
	valMap, ok := val.(map[string]interface{})
	require.True(t, ok, "Expected table to convert to map[string]interface{}")
	assert.Equal(t, "value", valMap["key"], "Expected key='value'")
}

func TestMixedOperations(t *testing.T) {
	cfg := New()
	defer cfg.Close()

	// Push a variety of values
	require.NoError(t, cfg.PushValue("first"))
	require.NoError(t, cfg.PushValue("second"))
	require.NoError(t, cfg.PushValue(123))
	require.NoError(t, cfg.PushValue(true))

	assert.Equal(t, 4, cfg.GetStackSize())

	// Peek at various positions
	topVal, err := cfg.PeekValue(1)
	require.NoError(t, err)
	assert.Equal(t, true, topVal, "Top should be true")

	secondVal, err := cfg.PeekValue(2)
	require.NoError(t, err)
	assert.Equal(t, float64(123), secondVal, "Second should be number 123")

	// Pop from top
	poppedVal, err := cfg.PopValue()
	require.NoError(t, err)
	assert.Equal(t, true, poppedVal)
	assert.Equal(t, 3, cfg.GetStackSize())

	// Clear the stack
	cfg.ClearStack()
	assert.Equal(t, 0, cfg.GetStackSize())

	// Push after clearing
	require.NoError(t, cfg.PushValue("restart"))
	assert.Equal(t, 1, cfg.GetStackSize())

	// Pop again
	val, err := cfg.PopValue()
	require.NoError(t, err)
	assert.Equal(t, "restart", val)
	assert.Equal(t, 0, cfg.GetStackSize())
}

func TestStackErrorHandling(t *testing.T) {
	cfg := New()
	defer cfg.Close()

	// Trying to peek into empty stack
	_, err := cfg.PeekValue(1)
	assert.Error(t, err, "Expected error peeking empty stack")

	// Trying to get raw value from empty stack
	_, err = cfg.GetRawLuaValue(1)
	assert.Error(t, err, "Expected error getting raw value from empty stack")

	// Trying to pop from empty stack
	_, err = cfg.PopValue()
	assert.Error(t, err, "Expected error popping from empty stack")

	// Push an unsupported type to test goToLua error handling
	type unsupported struct{}
	err = cfg.PushValue(unsupported{})
	assert.Error(t, err, "Expected error pushing unsupported type")
}

func TestPushMethods(t *testing.T) {
	cfg := New()
	defer cfg.Close()

	// Test PushString
	err := cfg.PushString("hello")
	require.NoError(t, err)
	val, err := cfg.PopValue()
	require.NoError(t, err)
	assert.Equal(t, "hello", val)

	// Test PushNumber
	err = cfg.PushNumber(3.14)
	require.NoError(t, err)
	val, err = cfg.PopValue()
	require.NoError(t, err)
	assert.Equal(t, float64(3.14), val.(float64))

	// Test PushBool
	err = cfg.PushBool(true)
	require.NoError(t, err)
	val, err = cfg.PopValue()
	require.NoError(t, err)
	assert.Equal(t, true, val.(bool))

	// Test PushNil
	err = cfg.PushNil()
	require.NoError(t, err)
	val, err = cfg.PopValue()
	require.NoError(t, err)
	assert.Nil(t, val)

	// Test Push generic
	complexValue := map[string]interface{}{
		"key": "value",
		"num": 42,
	}
	err = cfg.Push(complexValue)
	require.NoError(t, err)
	val, err = cfg.PopValue()
	require.NoError(t, err)
	mapped, ok := val.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "value", mapped["key"])
	assert.Equal(t, float64(42), mapped["num"])
}
