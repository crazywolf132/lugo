package lugo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func TestLuaError(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	// Define a Lua function that raises an error
	err := L.DoString(`function test() error("test") end`)
	require.NoError(t, err)

	// Call the function using protected call
	err = L.CallByParam(lua.P{
		Fn:      L.GetGlobal("test"),
		NRet:    0,
		Protect: true,
	})
	assert.Error(t, err)

	t.Logf("Error message: %v", err)

	luaErr := NewLuaError(L, ErrExecution, "Lua error", err)
	require.NotNil(t, luaErr, "LuaError should not be nil")
	require.NotNil(t, luaErr.BaseError, "BaseError should not be nil")

	assert.Equal(t, ErrExecution, luaErr.BaseError.Code)
	assert.Equal(t, "Lua error", luaErr.BaseError.Message)

	require.Len(t, luaErr.Stack, 1, "stack should have at least 1 frame")
	assert.Equal(t, "test", luaErr.Stack[0].Function)
}

func TestLuaErrorWithNestedCalls(t *testing.T) {
	L := lua.NewState()
	defer L.Close()

	// Define the Lua functions
	err := L.DoString(`
        function a()
            b()
        end

        function b()
            error("test")
        end
    `)
	require.NoError(t, err)

	// Call the top-level function using protected call
	err = L.CallByParam(lua.P{
		Fn:      L.GetGlobal("a"),
		NRet:    0,
		Protect: true,
	})
	assert.Error(t, err)

	t.Logf("Error message: %v", err)

	luaErr := NewLuaError(L, ErrExecution, "Lua error", err)
	require.NotNil(t, luaErr, "LuaError should not be nil")
	require.NotNil(t, luaErr.BaseError, "BaseError should not be nil")

	assert.Equal(t, ErrExecution, luaErr.BaseError.Code)
	assert.Equal(t, "Lua error", luaErr.BaseError.Message)

	require.True(t, len(luaErr.Stack) >= 2, "should have at least 2 stack frames")
	assert.Equal(t, "b", luaErr.Stack[0].Function)
	assert.Equal(t, "a", luaErr.Stack[1].Function)
}
