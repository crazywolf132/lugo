package lugo

import (
	"context"
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

func TestRegisterLuaFunction(t *testing.T) {
	cfg := New()
	defer cfg.Close()

	// Test registering a simple Lua function
	err := cfg.RegisterLuaFunction("add", func(L *lua.LState) int {
		a := L.CheckNumber(1)
		b := L.CheckNumber(2)
		L.Push(lua.LNumber(a + b))
		return 1
	})
	require.NoError(t, err)

	// Test the registered function
	err = cfg.L.DoString(`
		result = add(5, 3)
		assert(result == 8)
	`)
	require.NoError(t, err)

	// Test error cases
	t.Run("empty name", func(t *testing.T) {
		err := cfg.RegisterLuaFunction("", func(L *lua.LState) int { return 0 })
		require.Error(t, err)
		assert.Equal(t, "function name cannot be empty", err.(*Error).Message)
	})

	t.Run("nil function", func(t *testing.T) {
		err := cfg.RegisterLuaFunction("test", nil)
		require.Error(t, err)
		assert.Equal(t, "function cannot be nil", err.(*Error).Message)
	})
}

func TestRegisterLuaFunctionString(t *testing.T) {
	cfg := New()
	defer cfg.Close()

	// Test registering a Lua function from string
	err := cfg.RegisterLuaFunctionString("multiply", `
		local x, y = ...
		return x * y
	`)
	require.NoError(t, err)

	// Test the registered function
	err = cfg.L.DoString(`
		result = multiply(6, 7)
		assert(result == 42)
	`)
	require.NoError(t, err)

	// Test registering a pre-defined function
	err = cfg.RegisterLuaFunctionString("greet", `
		function greet(name)
			return "Hello, " .. name
		end
	`)
	require.NoError(t, err)

	// Test the pre-defined function
	err = cfg.L.DoString(`
		result = greet("World")
		assert(result == "Hello, World")
	`)
	require.NoError(t, err)

	// Test error cases
	t.Run("empty name", func(t *testing.T) {
		err := cfg.RegisterLuaFunctionString("", "return 42")
		require.Error(t, err)
		assert.Equal(t, "function name cannot be empty", err.(*Error).Message)
	})

	t.Run("empty code", func(t *testing.T) {
		err := cfg.RegisterLuaFunctionString("test", "")
		require.Error(t, err)
		assert.Equal(t, "function code cannot be empty", err.(*Error).Message)
	})

	t.Run("invalid code", func(t *testing.T) {
		err := cfg.RegisterLuaFunctionString("test", "this is not valid lua")
		require.Error(t, err)
		assert.Equal(t, "failed to register Lua function", err.(*Error).Message)
	})
}

func TestRegisterLuaFunctionWithOptions(t *testing.T) {
	cfg := New()
	defer cfg.Close()

	// Test namespace registration
	err := cfg.RegisterLuaFunctionWithOptions("get", func(L *lua.LState) int {
		L.Push(lua.LString("response"))
		return 1
	}, FunctionOptions{
		Namespace: "http.client",
		Metadata: &FunctionMetadata{
			Description: "Makes an HTTP GET request",
			Params: []ParamMetadata{
				{Name: "url", Type: "string", Description: "The URL to request"},
			},
			Returns: []ReturnMetadata{
				{Name: "response", Type: "string", Description: "The response body"},
			},
		},
	})
	require.NoError(t, err)

	// Test the namespaced function
	err = cfg.L.DoString(`
		result = http.client.get("https://example.com")
		assert(result == "response")
	`)
	require.NoError(t, err)

	// Test function composition
	err = cfg.RegisterLuaFunctionString("double", `
		local x = ...
		return x * 2
	`)
	require.NoError(t, err)

	err = cfg.RegisterLuaFunctionString("addOne", `
		local x = ...
		return x + 1
	`)
	require.NoError(t, err)

	err = cfg.ComposeFunctions("doubleAndAddOne", "double", "addOne")
	require.NoError(t, err)

	err = cfg.L.DoString(`
		result = doubleAndAddOne(5)
		print("Result:", result)
		assert(result == 11) -- (5 * 2) + 1
	`)
	require.NoError(t, err)

	// Test middleware
	called := false
	loggingCalled := false
	metricsCalled := false

	// Register test middleware
	cfg.middlewareMap = map[string]func(lua.LGFunction) lua.LGFunction{
		"logging": func(next lua.LGFunction) lua.LGFunction {
			return func(L *lua.LState) int {
				loggingCalled = true
				return next(L)
			}
		},
		"metrics": func(next lua.LGFunction) lua.LGFunction {
			return func(L *lua.LState) int {
				metricsCalled = true
				return next(L)
			}
		},
	}

	err = cfg.RegisterLuaFunctionWithOptions("secure", func(L *lua.LState) int {
		L.Push(lua.LString("secret"))
		return 1
	}, FunctionOptions{
		BeforeCall: func(L *lua.LState) error {
			called = true
			// Simulate auth check
			return nil
		},
		Middleware: []string{"logging", "metrics"},
	})
	require.NoError(t, err)

	err = cfg.L.DoString(`
		result = secure()
		assert(result == "secret")
	`)
	require.NoError(t, err)
	assert.True(t, called, "BeforeCall hook should have been called")
	assert.True(t, loggingCalled, "Logging middleware should have been called")
	assert.True(t, metricsCalled, "Metrics middleware should have been called")
}

func TestRegisterFunctionTable(t *testing.T) {
	cfg := New()
	defer cfg.Close()

	ctx := context.Background()
	ctxWithValue := context.WithValue(ctx, "key", "value")

	// Test registering a table of functions
	funcs := map[string]interface{}{
		"add": func(x, y int) int {
			return x + y
		},
		"multiply": func(x, y float64) float64 {
			return x * y
		},
		"greet": func(name string) string {
			return "Hello, " + name
		},
		"withContext": func(ctx context.Context, msg string) string {
			if val, ok := ctx.Value("key").(string); ok {
				return val + ": " + msg
			}
			return "Context: " + msg
		},
	}

	err := cfg.RegisterFunctionTable(ctxWithValue, "utils", funcs)
	require.NoError(t, err)

	// Test the registered functions
	err = cfg.L.DoString(`
		assert(utils.add(5, 3) == 8)
		assert(utils.multiply(6, 7) == 42)
		assert(utils.greet("World") == "Hello, World")
		assert(utils.withContext("test") == "value: test")
	`)
	require.NoError(t, err)

	// Test error cases
	t.Run("empty name", func(t *testing.T) {
		err := cfg.RegisterFunctionTable(ctxWithValue, "", funcs)
		require.Error(t, err)
		assert.Equal(t, "table name cannot be empty", err.(*Error).Message)
	})

	t.Run("empty funcs", func(t *testing.T) {
		err := cfg.RegisterFunctionTable(ctxWithValue, "empty", map[string]interface{}{})
		require.Error(t, err)
		assert.Equal(t, "functions map cannot be empty", err.(*Error).Message)
	})

	t.Run("invalid function", func(t *testing.T) {
		err := cfg.RegisterFunctionTable(ctxWithValue, "invalid", map[string]interface{}{
			"bad": "not a function",
		})
		require.Error(t, err)
		assert.Contains(t, err.(*Error).Message, "failed to wrap function")
	})

	// Test with hooks
	var hookCalled bool
	cfg.RegisterHook(BeforeExec, func(ctx context.Context, event HookEvent) error {
		hookCalled = true
		return nil
	})

	err = cfg.RegisterFunctionTable(ctxWithValue, "more", map[string]interface{}{
		"test": func() string { return "test" },
	})
	require.NoError(t, err)
	assert.True(t, hookCalled, "hook should have been called")
}
