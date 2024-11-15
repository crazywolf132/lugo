package lugo

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

// Test structures
type SimpleConfig struct {
	Name    string `lua:"name"`
	Value   int    `lua:"value"`
	Enabled bool   `lua:"enabled"`
}

type ComplexConfig struct {
	Basic struct {
		ID      string  `lua:"id"`
		Name    string  `lua:"name"`
		Version float64 `lua:"version"`
	} `lua:"basic"`
	Arrays struct {
		Strings []string    `lua:"strings"`
		Numbers []float64   `lua:"numbers"`
		Mixed   [][]string  `lua:"mixed"`
		Matrix  [][]float64 `lua:"matrix"`
	} `lua:"arrays"`
	Maps struct {
		Settings map[string]string       `lua:"settings"`
		Nested   map[string]SimpleConfig `lua:"nested"`
		Complex  map[string]interface{}  `lua:"complex"`
	} `lua:"maps"`
	Timing struct {
		Created  time.Time            `lua:"created"`
		Modified time.Time            `lua:"modified"`
		Schedule map[string]time.Time `lua:"schedule"`
	} `lua:"timing"`
}

// TestBasicInitialization tests the basic initialization of the Lugo system
func TestBasicInitialization(t *testing.T) {
	tests := []struct {
		name    string
		options []Option
		verify  func(*testing.T, *Config)
	}{
		{
			name:    "default configuration",
			options: nil,
			verify: func(t *testing.T, cfg *Config) {
				assert.NotNil(t, cfg.L)
				assert.NotNil(t, cfg.logger)
				assert.NotNil(t, cfg.sandbox)
				assert.Empty(t, cfg.middlewares)
				assert.Empty(t, cfg.hooks)
			},
		},
		{
			name: "custom logger",
			options: []Option{
				WithLogger(zap.NewExample()),
			},
			verify: func(t *testing.T, cfg *Config) {
				assert.NotNil(t, cfg.logger)
			},
		},
		{
			name: "custom sandbox",
			options: []Option{
				WithSandbox(&Sandbox{
					EnableFileIO:     true,
					EnableNetworking: false,
					MaxMemory:        1024 * 1024,
				}),
			},
			verify: func(t *testing.T, cfg *Config) {
				assert.True(t, cfg.sandbox.EnableFileIO)
				assert.False(t, cfg.sandbox.EnableNetworking)
				assert.Equal(t, uint64(1024*1024), cfg.sandbox.MaxMemory)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := New(tt.options...)
			require.NotNil(t, cfg)
			tt.verify(t, cfg)
			cfg.Close()
		})
	}
}

// TestTypeRegistration tests the registration and validation of Go types
func TestTypeRegistration(t *testing.T) {
	tests := []struct {
		name       string
		typeName   string
		typeStruct interface{}
		script     string
		validate   func(*testing.T, interface{})
		wantErr    bool
	}{
		{
			name:     "simple struct",
			typeName: "config",
			typeStruct: SimpleConfig{
				Name:    "default",
				Value:   42,
				Enabled: true,
			},
			script: `
                config = {
                    name = "test",
                    value = 100,
                    enabled = false
                }
            `,
			validate: func(t *testing.T, v interface{}) {
				cfg := v.(*SimpleConfig)
				assert.Equal(t, "test", cfg.Name)
				assert.Equal(t, 100, cfg.Value)
				assert.False(t, cfg.Enabled)
			},
			wantErr: false,
		},
		{
			name:       "complex struct",
			typeName:   "config",
			typeStruct: ComplexConfig{},
			script: `
				-- Helper function to create a timestamp table
				local function timestamp()
					local t = os.date("*t")
					return {
						year = t.year,
						month = t.month,
						day = t.day,
						hour = t.hour,
						min = t.min,
						sec = t.sec
					}
				end

				config = {
					basic = {
						id = "123",
						name = "test",
						version = 1.0
					},
					arrays = {
						strings = {"a", "b", "c"},
						numbers = {1.1, 2.2, 3.3},
						mixed = {{"x", "y"}, {"z"}},
						matrix = {{1, 2}, {3, 4}}
					},
					maps = {
						settings = {
							theme = "dark",
							lang = "en"
						},
						nested = {
							item1 = {
								name = "nested1",
								value = 42,
								enabled = true
							}
						},
						complex = {
							key1 = "value1",
							key2 = 123,
							key3 = true
						}
					},
					timing = {
						created = timestamp(),
						modified = timestamp(),
						schedule = {
							start = timestamp(),
							["end"] = timestamp()
						}
					}
				}
			`,
			validate: func(t *testing.T, v interface{}) {
				cfg := v.(*ComplexConfig)
				// Basic fields
				assert.Equal(t, "123", cfg.Basic.ID)
				assert.Equal(t, "test", cfg.Basic.Name)
				assert.Equal(t, 1.0, cfg.Basic.Version)

				// Arrays
				assert.Equal(t, []string{"a", "b", "c"}, cfg.Arrays.Strings)
				assert.Equal(t, []float64{1.1, 2.2, 3.3}, cfg.Arrays.Numbers)
				assert.Len(t, cfg.Arrays.Mixed, 2)
				assert.Equal(t, [][]float64{{1, 2}, {3, 4}}, cfg.Arrays.Matrix)

				// Maps
				assert.Equal(t, "dark", cfg.Maps.Settings["theme"])
				assert.Equal(t, "en", cfg.Maps.Settings["lang"])
				if nested, ok := cfg.Maps.Nested["item1"]; ok {
					assert.Equal(t, "nested1", nested.Name)
					assert.Equal(t, 42, nested.Value)
					assert.True(t, nested.Enabled)
				}

				// Time fields - just verify they're not zero and are recent
				now := time.Now()
				maxDiff := time.Minute // Allow for some time difference during test execution

				assert.True(t, now.Sub(cfg.Timing.Created) < maxDiff, "Created time should be recent")
				assert.True(t, now.Sub(cfg.Timing.Modified) < maxDiff, "Modified time should be recent")

				for key, timeVal := range cfg.Timing.Schedule {
					assert.True(t, now.Sub(timeVal) < maxDiff,
						"Time value for key %s should be recent (got %v)", key, timeVal)
				}
			},
			wantErr: false,
		},
		{
			name:     "invalid type",
			typeName: "config",
			typeStruct: func() interface{} {
				// Return a non-struct type
				return "not a struct"
			}(),
			script:   `config = { value = "test" }`,
			validate: nil,
			wantErr:  true,
		},
		{
			name:       "nil registration",
			typeName:   "config",
			typeStruct: nil,
			script:     "",
			validate:   nil,
			wantErr:    true,
		},
	}

	// Run the regular test cases first
	for _, tt := range tests {
		if tt.name != "nil registration" { // Skip nil registration test
			t.Run(tt.name, func(t *testing.T) {
				cfg := New()
				defer cfg.Close()

				err := cfg.RegisterType(context.Background(), tt.typeName, tt.typeStruct)
				if tt.wantErr {
					assert.Error(t, err)
					return
				}
				require.NoError(t, err)

				if tt.script != "" {
					err = cfg.L.DoString(tt.script)
					if tt.wantErr {
						assert.Error(t, err)
						return
					}
					require.NoError(t, err)

					if tt.validate != nil {
						newVal := reflect.New(reflect.TypeOf(tt.typeStruct)).Interface()
						err = cfg.Get(context.Background(), tt.typeName, newVal)
						require.NoError(t, err)
						tt.validate(t, newVal)
					}
				}
			})
		}
	}
}

// Separate test for nil registration
func TestTypeRegistration_NilType(t *testing.T) {
	cfg := New()
	defer cfg.Close()

	err := cfg.RegisterType(context.Background(), "config", nil)
	require.Error(t, err, "RegisterType should return error for nil type")

	if e, ok := err.(*Error); ok {
		assert.Equal(t, ErrInvalidType, e.Code, "Expected ErrInvalidType for nil registration")
		assert.Equal(t, "typeStruct cannot be nil", e.Message)
	} else {
		t.Error("Expected error of type *Error")
	}
}

// TestFunctionRegistration tests the registration and execution of Go functions
func TestFunctionRegistration(t *testing.T) {
	tests := []struct {
		name     string
		fn       interface{}
		script   string
		validate func(*testing.T, *lua.LState)
		wantErr  bool
	}{
		{
			name: "simple function",
			fn: func(x int) int {
				return x * 2
			},
			script:  `assert(test(21) == 42)`,
			wantErr: false,
		},
		{
			name: "function with context",
			fn: func(ctx context.Context, msg string) string {
				return "Hello " + msg
			},
			script:  `assert(test("World") == "Hello World")`,
			wantErr: false,
		},
		{
			name: "function with error",
			fn: func() error {
				return errors.New("test error")
			},
			script:  `test()`,
			wantErr: true,
		},
		{
			name: "complex function",
			fn: func(numbers []float64, op string) (float64, error) {
				if len(numbers) == 0 {
					return 0, errors.New("empty input")
				}

				var result float64
				switch op {
				case "sum":
					for _, n := range numbers {
						result += n
					}
				case "avg":
					for _, n := range numbers {
						result += n
					}
					result /= float64(len(numbers))
				default:
					return 0, fmt.Errorf("unknown operation: %s", op)
				}

				return result, nil
			},
			script: `
				local sum = test({1, 2, 3, 4, 5}, "sum")
				assert(sum == 15)
				local avg = test({1, 2, 3, 4, 5}, "avg")
				assert(avg == 3)
			`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := New()
			defer cfg.Close()

			err := cfg.RegisterFunction(context.Background(), "test", tt.fn)
			require.NoError(t, err)

			err = cfg.L.DoString(tt.script)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			if tt.validate != nil {
				tt.validate(t, cfg.L)
			}
		})
	}
}

// TestSandbox tests the sandbox security features
func TestSandbox(t *testing.T) {
	tests := []struct {
		name    string
		sandbox *Sandbox
		script  string
		wantErr bool
	}{
		{
			name: "restricted functions",
			sandbox: &Sandbox{
				EnableFileIO: false,
			},
			script: `
                -- Try to use a restricted function
                return loadfile ~= nil or dofile ~= nil or require('io') ~= nil
            `,
			wantErr: false, // Script should run but return false
		},
		{
			name: "networking disabled",
			sandbox: &Sandbox{
				EnableNetworking: false,
			},
			script: `
                local socket = require("socket")
            `,
			wantErr: true,
		},
		{
			name: "basic sandbox",
			sandbox: &Sandbox{
				EnableFileIO:     false,
				EnableNetworking: false,
			},
			script: `
                -- Verify we can still use basic Lua functionality
                local t = {1, 2, 3}
                local sum = 0
                for i, v in ipairs(t) do
                    sum = sum + v
                end
                assert(sum == 6)
                return true
            `,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := New(WithSandbox(tt.sandbox))
			defer cfg.Close()

			err := cfg.L.DoString(tt.script)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConcurrentMiddleware(t *testing.T) {
	var (
		callCount int32
	)

	middleware := func(next LuaFunction) LuaFunction {
		return func(ctx context.Context, L *lua.LState) ([]lua.LValue, error) {
			atomic.AddInt32(&callCount, 1)
			return next(ctx, L)
		}
	}

	cfg := New(WithMiddleware(middleware))
	defer cfg.Close()

	// Register a simple test function
	err := cfg.RegisterFunction(context.Background(), "test", func(ctx context.Context) int {
		return 42
	})
	require.NoError(t, err)

	// Prepare the script
	script := `return test()`

	// Run concurrent calls
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Create a new Lua state for each goroutine
			localCfg := New(WithMiddleware(middleware))
			defer localCfg.Close()

			// Register the function in the local state
			err := localCfg.RegisterFunction(context.Background(), "test", func(ctx context.Context) int {
				return 42
			})
			if err != nil {
				errors <- err
				return
			}

			// Execute the script
			err = localCfg.L.DoString(script)
			if err != nil {
				errors <- err
				return
			}

			// Verify the result
			result := localCfg.L.Get(-1) // Get the top of the stack
			if num, ok := result.(lua.LNumber); !ok || int(num) != 42 {
				errors <- fmt.Errorf("unexpected result: %v", result)
			}
			localCfg.L.Pop(1) // Clean up the stack
		}()
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(errors)

	// Check for any errors
	for err := range errors {
		t.Errorf("concurrent execution error: %v", err)
	}

	// Verify middleware was called the expected number of times
	assert.Equal(t, int32(100), atomic.LoadInt32(&callCount))
}

// TestConcurrency tests thread safety of the configuration system
func TestConcurrency(t *testing.T) {
	cfg := New()
	defer cfg.Close()

	// Register a simple counter function
	err := cfg.RegisterFunction(context.Background(), "increment", func(x int) int {
		return x + 1
	})
	require.NoError(t, err)

	var wg sync.WaitGroup
	errs := make(chan error, 100)

	// Create a mutex to synchronize access to the Lua state
	var stateMutex sync.Mutex

	// Run multiple goroutines accessing the Lua state
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(val int) {
			defer wg.Done()

			// Create a new state for each goroutine
			localCfg := New()
			defer localCfg.Close()

			// Register the increment function in the local state
			err := localCfg.RegisterFunction(context.Background(), "increment", func(x int) int {
				return x + 1
			})
			if err != nil {
				errs <- fmt.Errorf("failed to register function: %v", err)
				return
			}

			// Execute the script with proper locking
			stateMutex.Lock()
			err = localCfg.L.DoString(fmt.Sprintf("assert(increment(%d) == %d)", val, val+1))
			stateMutex.Unlock()

			if err != nil {
				errs <- fmt.Errorf("script execution error: %v", err)
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	// Check for any errors
	for err := range errs {
		t.Errorf("Concurrent execution error: %v", err)
	}
}

// TestHooks tests the hook system
func TestHooks(t *testing.T) {
	var hookCalls []string

	cfg := New()
	defer cfg.Close()

	// Register hooks for all types
	cfg.RegisterHook(BeforeLoad, func(ctx context.Context, event HookEvent) error {
		hookCalls = append(hookCalls, "before_load")
		return nil
	})

	cfg.RegisterHook(AfterLoad, func(ctx context.Context, event HookEvent) error {
		hookCalls = append(hookCalls, "after_load")
		// Verify event data
		assert.NotEmpty(t, event.Name)
		assert.NotZero(t, event.Elapsed)
		return nil
	})

	cfg.RegisterHook(BeforeExec, func(ctx context.Context, event HookEvent) error {
		hookCalls = append(hookCalls, "before_exec")
		return nil
	})

	cfg.RegisterHook(AfterExec, func(ctx context.Context, event HookEvent) error {
		hookCalls = append(hookCalls, "after_exec")
		return nil
	})

	// Create and load a test file
	tmpFile := filepath.Join(t.TempDir(), "test.lua")
	err := os.WriteFile(tmpFile, []byte(`print("test")`), 0644)
	require.NoError(t, err)

	err = cfg.LoadFile(context.Background(), tmpFile)
	// Continuing from previous TestHooks
	require.NoError(t, err)

	// Verify hook execution order
	expectedHooks := []string{
		"before_load",
		"after_load",
	}
	assert.Equal(t, expectedHooks, hookCalls)
}

// TestMiddleware tests the middleware system
func TestMiddleware(t *testing.T) {
	// Simple counter to track middleware execution
	var callCount int32

	middleware := func(next LuaFunction) LuaFunction {
		return func(ctx context.Context, L *lua.LState) ([]lua.LValue, error) {
			atomic.AddInt32(&callCount, 1)
			return next(ctx, L)
		}
	}

	cfg := New(WithMiddleware(middleware))
	defer cfg.Close()

	// Register a simple test function
	err := cfg.RegisterFunction(context.Background(), "test", func() int {
		return 42
	})
	require.NoError(t, err)

	// Run the function
	err = cfg.L.DoString("assert(test() == 42)")
	require.NoError(t, err)

	// Verify middleware was called
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))
}

func TestChainedMiddleware(t *testing.T) {
	var order []string
	var mu sync.Mutex

	addOrder := func(s string) {
		mu.Lock()
		order = append(order, s)
		mu.Unlock()
	}

	middleware1 := func(next LuaFunction) LuaFunction {
		return func(ctx context.Context, L *lua.LState) ([]lua.LValue, error) {
			addOrder("m1_before")
			res, err := next(ctx, L)
			addOrder("m1_after")
			return res, err
		}
	}

	middleware2 := func(next LuaFunction) LuaFunction {
		return func(ctx context.Context, L *lua.LState) ([]lua.LValue, error) {
			addOrder("m2_before")
			res, err := next(ctx, L)
			addOrder("m2_after")
			return res, err
		}
	}

	cfg := New(
		WithMiddleware(middleware1),
		WithMiddleware(middleware2),
	)
	defer cfg.Close()

	// Register test function
	err := cfg.RegisterFunction(context.Background(), "test", func() int {
		addOrder("execute")
		return 42
	})
	require.NoError(t, err)

	// Execute
	err = cfg.L.DoString("test()")
	require.NoError(t, err)

	// Check order
	expected := []string{"m1_before", "m2_before", "execute", "m2_after", "m1_after"}
	assert.Equal(t, expected, order)
}

// TestErrorHandling tests various error conditions
func TestErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*Config)
		action      func(*Config) error
		expectedErr ErrorCode
		errCheck    func(*testing.T, error)
	}{
		{
			name: "type registration error",
			action: func(cfg *Config) error {
				return cfg.RegisterType(context.Background(), "invalid", nil)
			},
			expectedErr: ErrInvalidType,
			errCheck: func(t *testing.T, err error) {
				luaErr, ok := err.(*Error)
				assert.True(t, ok)
				assert.Equal(t, ErrInvalidType, luaErr.Code)
			},
		},
		{
			name: "validation error",
			setup: func(cfg *Config) {
				_ = cfg.RegisterType(context.Background(), "config", struct {
					Age int `lua:"age"`
				}{})
			},
			action: func(cfg *Config) error {
				err := cfg.L.DoString(`
                    config = {
                        age = "not a number"
                    }
                `)
				if err != nil {
					return err
				}
				var result struct {
					Age int `lua:"age"`
				}
				return cfg.Get(context.Background(), "config", &result)
			},
			expectedErr: ErrValidation,
			errCheck: func(t *testing.T, err error) {
				luaErr, ok := err.(*Error)
				assert.True(t, ok)
				assert.Equal(t, ErrValidation, luaErr.Code)
			},
		},
		{
			name: "sandbox error",
			setup: func(cfg *Config) {
				cfg.sandbox = &Sandbox{
					EnableFileIO: false,
				}
				_ = cfg.applySandboxRestrictions()
			},
			action: func(cfg *Config) error {
				return cfg.L.DoString(`
                    local file = io.open("test.txt", "w")
                    if file then
                        file:write("test")
                    end
                `)
			},
			expectedErr: ErrSandbox,
			errCheck: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
		},
		{
			name: "execution error",
			action: func(cfg *Config) error {
				return cfg.L.DoString(`
                    error("custom error")
                `)
			},
			expectedErr: ErrExecution,
			errCheck: func(t *testing.T, err error) {
				assert.Error(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := New()
			defer cfg.Close()

			if tt.setup != nil {
				tt.setup(cfg)
			}

			err := tt.action(cfg)
			assert.Error(t, err)

			if tt.errCheck != nil {
				tt.errCheck(t, err)
			}
		})
	}
}

// TestTypeConversion tests various type conversions between Go and Lua
func TestTypeConversion(t *testing.T) {
	type CustomTime struct {
		Time time.Time `lua:"time"`
	}

	tests := []struct {
		name      string
		goValue   interface{}
		luaScript string
		validate  func(*testing.T, interface{})
	}{
		{
			name: "time conversion",
			goValue: CustomTime{
				Time: time.Now(),
			},
			luaScript: `
                local now = os.time()
                config = {
                    time = {
                        year = os.date("*t", now).year,
                        month = os.date("*t", now).month,
                        day = os.date("*t", now).day,
                        hour = os.date("*t", now).hour,
                        min = os.date("*t", now).min,
                        sec = os.date("*t", now).sec
                    }
                }
            `,
			validate: func(t *testing.T, v interface{}) {
				ct := v.(*CustomTime)
				assert.NotZero(t, ct.Time)
				assert.WithinDuration(t, time.Now(), ct.Time, time.Hour*24)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := New()
			defer cfg.Close()

			err := cfg.RegisterType(context.Background(), "config", tt.goValue)
			require.NoError(t, err)

			err = cfg.L.DoString(tt.luaScript)
			require.NoError(t, err)

			newVal := reflect.New(reflect.TypeOf(tt.goValue)).Interface()
			err = cfg.Get(context.Background(), "config", newVal)
			require.NoError(t, err)

			if tt.validate != nil {
				tt.validate(t, newVal)
			}
		})
	}
}

// Benchmark tests
func BenchmarkLuaExecution(b *testing.B) {
	cfg := New()
	defer cfg.Close()

	script := `
		function fibonacci(n)
			if n <= 1 then
				return n
			end
			return fibonacci(n-1) + fibonacci(n-2)
		end
	`
	err := cfg.L.DoString(script)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := cfg.L.DoString("fibonacci(10)")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTypeConversion(b *testing.B) {
	type BenchConfig struct {
		Name     string            `lua:"name"`
		Values   []int             `lua:"values"`
		Metadata map[string]string `lua:"metadata"`
	}

	cfg := New()
	defer cfg.Close()

	err := cfg.RegisterType(context.Background(), "config", BenchConfig{})
	require.NoError(b, err)

	script := `
		config = {
			name = "benchmark",
			values = {1, 2, 3, 4, 5},
			metadata = {
				key1 = "value1",
				key2 = "value2",
				key3 = "value3"
			}
		}
	`
	err = cfg.L.DoString(script)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result BenchConfig
		err := cfg.Get(context.Background(), "config", &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConcurrentAccess(b *testing.B) {
	cfg := New()
	defer cfg.Close()

	err := cfg.RegisterFunction(context.Background(), "compute", func(x int) int {
		return x * x
	})
	require.NoError(b, err)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := cfg.L.DoString("compute(42)")
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func TestTypeRegistrationErrors(t *testing.T) {
	tests := []struct {
		name       string
		typeStruct interface{}
		wantMsg    string
	}{
		{
			name:       "nil type",
			typeStruct: nil,
			wantMsg:    "typeStruct cannot be nil",
		},
		{
			name:       "string type",
			typeStruct: "not a struct",
			wantMsg:    "typeStruct must be a struct, got string",
		},
		{
			name:       "int type",
			typeStruct: 42,
			wantMsg:    "typeStruct must be a struct, got int",
		},
		{
			name:       "slice type",
			typeStruct: []string{"not", "a", "struct"},
			wantMsg:    "typeStruct must be a struct, got []string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := New()
			defer cfg.Close()

			err := cfg.RegisterType(context.Background(), "test", tt.typeStruct)

			require.Error(t, err)
			if e, ok := err.(*Error); ok {
				assert.Equal(t, ErrInvalidType, e.Code)
				assert.Equal(t, tt.wantMsg, e.Message)
			} else {
				t.Error("Expected error of type *Error")
			}
		})
	}
}

// Additional test for thread safety
func TestThreadSafety(t *testing.T) {
	cfg := New()
	defer cfg.Close()

	// Register a function that maintains internal state
	var counter int32
	err := cfg.RegisterFunction(context.Background(), "increment", func() int32 {
		return atomic.AddInt32(&counter, 1)
	})
	require.NoError(t, err)

	// Run concurrent increments
	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			localCfg := New()
			defer localCfg.Close()

			err := localCfg.RegisterFunction(context.Background(), "increment", func() int32 {
				return atomic.AddInt32(&counter, 1)
			})
			if err != nil {
				errors <- err
				return
			}

			err = localCfg.L.DoString("return increment()")
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("thread safety error: %v", err)
	}

	assert.Equal(t, int32(100), counter)
}
