// Package luaconfig provides a production-ready Lua configuration system for Go
package lugo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	lua "github.com/yuin/gopher-lua"
	"go.uber.org/zap"
)

// Config represents the configuration manager
type Config struct {
	L           *lua.LState
	logger      *zap.Logger
	sandbox     *Sandbox
	middlewares []Middleware
	mu          sync.RWMutex
	hooks       map[HookType][]Hook
}

// Option represents a configuration option
type Option func(*Config)

// Hook represents a function that can be called at various points
type Hook func(ctx context.Context, event HookEvent) error

// HookType represents different hook points
type HookType int

const (
	BeforeLoad HookType = iota
	AfterLoad
	BeforeExec
	AfterExec
)

// HookEvent contains information about the hook execution
type HookEvent struct {
	Type    HookType
	Name    string
	Args    []interface{}
	Result  interface{}
	Error   error
	Elapsed time.Duration
}

// Middleware represents a function that can modify behavior
type Middleware func(next LuaFunction) LuaFunction

// LuaFunction represents a function that can be called from Lua
type LuaFunction func(ctx context.Context, L *lua.LState) ([]lua.LValue, error)

// Sandbox provides security restrictions
type Sandbox struct {
	EnableFileIO     bool
	EnableNetworking bool
	EnableSyscalls   bool
	MaxMemory        uint64 // in bytes
	MaxExecutionTime time.Duration
	AllowedPaths     []string
	BlockedPaths     []string
}

// Error handling
type ErrorCode int

const (
	ErrInvalidType ErrorCode = iota
	ErrNotFound
	ErrValidation
	ErrSandbox
	ErrExecution
)

type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// New creates a new Config instance with options
func New(opts ...Option) *Config {
	cfg := &Config{
		L:      lua.NewState(),
		hooks:  make(map[HookType][]Hook),
		logger: zap.NewNop(),
		sandbox: &Sandbox{
			MaxMemory:        100 * 1024 * 1024, // 100MB
			MaxExecutionTime: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

// WithLogger sets the logger
func WithLogger(logger *zap.Logger) Option {
	return func(c *Config) {
		c.logger = logger
	}
}

// WithSandbox sets sandbox options
func WithSandbox(sandbox *Sandbox) Option {
	return func(c *Config) {
		c.sandbox = sandbox
	}
}

// WithMiddleware adds middleware
func WithMiddleware(middleware Middleware) Option {
	return func(c *Config) {
		c.middlewares = append(c.middlewares, middleware)
	}
}

// Close closes the Lua state
func (c *Config) Close() {
	c.L.Close()
}

// RegisterHook registers a hook for a specific point
func (c *Config) RegisterHook(hookType HookType, hook Hook) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hooks[hookType] = append(c.hooks[hookType], hook)
}

// RegisterType registers a Go struct as a Lua type with optional default values
func (c *Config) RegisterType(ctx context.Context, name string, typeStruct interface{}, defaultValue ...interface{}) error {
	if typeStruct == nil {
		return &Error{
			Code:    ErrInvalidType,
			Message: "typeStruct cannot be nil",
		}
	}

	c.logger.Debug("registering type",
		zap.String("name", name),
		zap.String("type", fmt.Sprintf("%T", typeStruct)),
	)

	val := reflect.ValueOf(typeStruct)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return &Error{
			Code:    ErrInvalidType,
			Message: fmt.Sprintf("typeStruct must be a struct, got %T", typeStruct),
		}
	}

	table := c.L.NewTable()

	if len(defaultValue) > 0 {
		defaultTable, err := c.structToTable(defaultValue[0])
		if err != nil {
			return &Error{
				Code:    ErrInvalidType,
				Message: "failed to convert default value",
				Cause:   err,
			}
		}
		table = defaultTable
	}

	c.L.SetGlobal(name, table)
	return nil
}

// RegisterFunction registers a Go function in the Lua environment with middlewares
func (c *Config) RegisterFunction(ctx context.Context, name string, fn interface{}) error {
	wrapped, err := c.wrapGoFunction(fn)
	if err != nil {
		return &Error{
			Code:    ErrInvalidType,
			Message: "failed to wrap function",
			Cause:   err,
		}
	}

	// Apply middlewares in reverse order
	final := wrapped
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		final = c.middlewares[i](final)
	}

	luaFn := c.createLuaFunction(name, final)
	c.L.SetGlobal(name, c.L.NewFunction(luaFn))

	return nil
}

// LoadFile loads and executes a Lua file with context and hooks
func (c *Config) LoadFile(ctx context.Context, filename string) error {
	start := time.Now()
	event := HookEvent{
		Type: BeforeLoad,
		Name: filename,
	}

	if err := c.runHooks(ctx, BeforeLoad, event); err != nil {
		return &Error{
			Code:    ErrExecution,
			Message: "before load hook failed",
			Cause:   err,
		}
	}

	// Apply sandbox restrictions
	if err := c.applySandboxRestrictions(); err != nil {
		return &Error{
			Code:    ErrSandbox,
			Message: "failed to apply sandbox restrictions",
			Cause:   err,
		}
	}

	err := c.L.DoFile(filename)
	elapsed := time.Since(start)

	event.Elapsed = elapsed
	event.Error = err

	if err != nil {
		return &Error{
			Code:    ErrExecution,
			Message: "failed to load file",
			Cause:   err,
		}
	}

	return c.runHooks(ctx, AfterLoad, event)
}

// Get retrieves the configuration into the provided struct with validation
func (c *Config) Get(ctx context.Context, name string, target interface{}) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	lv := c.L.GetGlobal(name)
	if lv == lua.LNil {
		return &Error{
			Code:    ErrNotFound,
			Message: fmt.Sprintf("configuration '%s' not found", name),
		}
	}

	if err := c.validateValue(lv, reflect.TypeOf(target).Elem()); err != nil {
		return &Error{
			Code:    ErrValidation,
			Message: "validation failed",
			Cause:   err,
		}
	}

	return c.luaToStruct(lv, target)
}

// Helper functions

func (c *Config) createLuaFunction(name string, fn LuaFunction) lua.LGFunction {
	return func(L *lua.LState) int {
		defer func() {
			if r := recover(); r != nil {
				c.logger.Error("function panic",
					zap.String("function", name),
					zap.Any("panic", r),
				)
				L.RaiseError("function execution failed: %v", r)
			}
		}()

		ctx := context.Background()
		results, err := fn(ctx, L)
		if err != nil {
			L.RaiseError("%v", err) // Changed to properly raise Lua error
			return 0
		}

		for _, result := range results {
			L.Push(result)
		}

		return len(results)
	}
}

func (c *Config) applySandboxRestrictions() error {
	// Create a restricted environment
	restricted := c.L.NewTable()

	// Add safe basic functions
	safeFuncs := map[string]lua.LGFunction{
		"assert":   c.L.GetGlobal("assert").(*lua.LFunction).GFunction,
		"error":    c.L.GetGlobal("error").(*lua.LFunction).GFunction,
		"ipairs":   c.L.GetGlobal("ipairs").(*lua.LFunction).GFunction,
		"next":     c.L.GetGlobal("next").(*lua.LFunction).GFunction,
		"pairs":    c.L.GetGlobal("pairs").(*lua.LFunction).GFunction,
		"select":   c.L.GetGlobal("select").(*lua.LFunction).GFunction,
		"tonumber": c.L.GetGlobal("tonumber").(*lua.LFunction).GFunction,
		"tostring": c.L.GetGlobal("tostring").(*lua.LFunction).GFunction,
		"type":     c.L.GetGlobal("type").(*lua.LFunction).GFunction,
		"unpack":   c.L.GetGlobal("unpack").(*lua.LFunction).GFunction,
	}

	// Add safe standard libraries
	safeLibs := map[string]*lua.LTable{
		"string": c.L.GetGlobal("string").(*lua.LTable),
		"table":  c.L.GetGlobal("table").(*lua.LTable),
		"math":   c.L.GetGlobal("math").(*lua.LTable),
	}

	// Set up safe functions
	c.L.SetFuncs(restricted, safeFuncs)

	// Add safe libraries to restricted environment
	for name, lib := range safeLibs {
		restricted.RawSetString(name, lib)
	}

	if !c.sandbox.EnableFileIO {
		// Remove file-related capabilities
		c.L.SetGlobal("io", lua.LNil)
		c.L.SetGlobal("dofile", lua.LNil)
		c.L.SetGlobal("loadfile", lua.LNil)
		c.L.SetGlobal("load", lua.LNil)

		// Restrict os library to non-file operations
		osTable := c.L.NewTable()
		if c.sandbox.EnableSyscalls {
			baseOS := c.L.GetGlobal("os").(*lua.LTable)
			safeOSFuncs := []string{"clock", "date", "difftime", "time"}
			for _, fname := range safeOSFuncs {
				c.L.SetField(osTable, fname, baseOS.RawGetString(fname))
			}
		}
		restricted.RawSetString("os", osTable)
	}

	if !c.sandbox.EnableNetworking {
		c.L.PreloadModule("socket", nil)
	}

	// Custom require function that respects restrictions
	requireFn := c.L.NewFunction(func(L *lua.LState) int {
		modname := L.CheckString(1)

		// Block access to disabled modules
		if !c.sandbox.EnableFileIO && (modname == "io" || modname == "os") {
			L.Push(lua.LNil)
			L.Push(lua.LString(fmt.Sprintf("module '%s' is disabled", modname)))
			return 2
		}
		if !c.sandbox.EnableNetworking && modname == "socket" {
			L.Push(lua.LNil)
			L.Push(lua.LString("networking is disabled"))
			return 2
		}

		// Call original require for allowed modules
		L.Push(L.GetGlobal("require"))
		L.Push(lua.LString(modname))
		L.Call(1, 1)
		return 1
	})
	restricted.RawSetString("require", requireFn)

	// Set memory limit (note: this is best-effort as Lua doesn't provide fine-grained control)
	if c.sandbox.MaxMemory > 0 {
		limitInK := int(c.sandbox.MaxMemory / 1024)
		if limitInK < 100 {
			return fmt.Errorf("memory limit too small (minimum 100KB)")
		}
		c.L.SetMx(limitInK)
	}

	// Replace the global environment
	c.L.SetGlobal("_G", restricted)

	return nil
}

func (c *Config) runHooks(ctx context.Context, hookType HookType, event HookEvent) error {
	c.mu.RLock()
	hooks := c.hooks[hookType]
	c.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook(ctx, event); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) structToTable(v interface{}) (*lua.LTable, error) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %T", v)
	}

	table := c.L.NewTable()
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" { // Skip unexported fields
			continue
		}

		// Get field name from tag or use field name
		name := field.Tag.Get("lua")
		if name == "" {
			name = strings.ToLower(field.Name)
		}

		fv := val.Field(i)
		lv, err := c.goToLua(fv.Interface())
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", name, err)
		}

		table.RawSetString(name, lv)
	}

	return table, nil
}

func (c *Config) wrapGoFunction(fn interface{}) (LuaFunction, error) {
	val := reflect.ValueOf(fn)
	if val.Kind() != reflect.Func {
		return nil, fmt.Errorf("expected function, got %T", fn)
	}

	return func(ctx context.Context, L *lua.LState) ([]lua.LValue, error) {
		ft := val.Type()
		totalArgs := ft.NumIn()
		contextOffset := 0

		// Check if first parameter is context
		hasContext := totalArgs > 0 && ft.In(0).Implements(reflect.TypeOf((*context.Context)(nil)).Elem())
		if hasContext {
			contextOffset = 1
		}

		// Prepare arguments
		args := make([]reflect.Value, totalArgs)
		luaIndex := 1 // Lua stack index starts at 1

		// Set context if needed
		if hasContext {
			args[0] = reflect.ValueOf(ctx)
		}

		// Convert arguments
		for i := contextOffset; i < totalArgs; i++ {
			paramType := ft.In(i)
			luaArg := L.Get(luaIndex)

			// Handle nil values
			if luaArg == lua.LNil {
				args[i] = reflect.Zero(paramType)
				luaIndex++
				continue
			}

			goArg, err := c.luaToGo(luaArg, paramType)
			if err != nil {
				return nil, fmt.Errorf("argument %d: %w", i+1, err)
			}
			args[i] = reflect.ValueOf(goArg)
			luaIndex++
		}

		// Call function
		results := val.Call(args)

		// Convert results
		luaResults := make([]lua.LValue, 0, len(results))
		for _, result := range results {
			// Special handling for error type
			if result.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
				if !result.IsNil() {
					return nil, result.Interface().(error)
				}
				continue
			}

			lv, err := c.goToLua(result.Interface())
			if err != nil {
				return nil, fmt.Errorf("failed to convert return value: %w", err)
			}
			luaResults = append(luaResults, lv)
		}

		return luaResults, nil
	}, nil
}

func (c *Config) validateValue(lv lua.LValue, t reflect.Type) error {

	// Handle nil values
	if lv == lua.LNil {
		switch t.Kind() {
		case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Map:
			return nil // These types can be nil
		default:
			// For other types, check if it's a required field
			// The required check should be handled at a higher level
			return nil
		}
	}

	switch t.Kind() {
	case reflect.Struct:
		if lv.Type() != lua.LTTable {
			return fmt.Errorf("expected table for struct, got %s", lv.Type())
		}
		table := lv.(*lua.LTable)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" { // Skip unexported fields
				continue
			}
			name := field.Tag.Get("lua")
			if name == "" {
				name = strings.ToLower(field.Name)
			}
			fieldValue := table.RawGetString(name)
			if err := c.validateValue(fieldValue, field.Type); err != nil {
				return fmt.Errorf("field %s: %w", name, err)
			}
		}
	case reflect.Slice:
		if lv == lua.LNil {
			return nil // Allow nil slices
		}
		if lv.Type() != lua.LTTable {
			return fmt.Errorf("expected table for slice, got %s", lv.Type())
		}
	case reflect.Map:
		if lv == lua.LNil {
			return nil // Allow nil maps
		}
		if lv.Type() != lua.LTTable {
			return fmt.Errorf("expected table for map, got %s", lv.Type())
		}
	case reflect.String:
		if lv.Type() != lua.LTString && lv.Type() != lua.LTNil {
			return fmt.Errorf("expected string, got %s", lv.Type())
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if lv.Type() != lua.LTNumber && lv.Type() != lua.LTNil {
			return fmt.Errorf("expected number, got %s", lv.Type())
		}
	case reflect.Float32, reflect.Float64:
		if lv.Type() != lua.LTNumber && lv.Type() != lua.LTNil {
			return fmt.Errorf("expected number, got %s", lv.Type())
		}
	case reflect.Bool:
		if lv.Type() != lua.LTBool && lv.Type() != lua.LTNil {
			return fmt.Errorf("expected boolean, got %s", lv.Type())
		}
	}
	return nil
}

func (c *Config) luaToStruct(lv lua.LValue, target interface{}) error {
	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}
	val = val.Elem()

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to struct")
	}

	table, ok := lv.(*lua.LTable)
	if !ok {
		return fmt.Errorf("expected table, got %T", lv)
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" { // Skip unexported fields
			continue
		}

		// Get field name from tag or use field name
		name := field.Tag.Get("lua")
		if name == "" {
			name = strings.ToLower(field.Name)
		}

		lval := table.RawGetString(name)
		if lval == lua.LNil {
			continue
		}

		goval, err := c.luaToGo(lval, field.Type)
		if err != nil {
			return fmt.Errorf("field %s: %w", name, err)
		}

		val.Field(i).Set(reflect.ValueOf(goval))
	}

	return nil
}

func (c *Config) goToLua(v interface{}) (lua.LValue, error) {
	if v == nil {
		return lua.LNil, nil
	}

	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.String:
		return lua.LString(val.String()), nil
	case reflect.Bool:
		return lua.LBool(val.Bool()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return lua.LNumber(val.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return lua.LNumber(val.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return lua.LNumber(val.Float()), nil
	case reflect.Slice, reflect.Array:
		table := c.L.NewTable()
		for i := 0; i < val.Len(); i++ {
			lv, err := c.goToLua(val.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			table.Append(lv)
		}
		return table, nil
	case reflect.Map:
		table := c.L.NewTable()
		iter := val.MapRange()
		for iter.Next() {
			k := iter.Key()
			v := iter.Value()
			lv, err := c.goToLua(v.Interface())
			if err != nil {
				return nil, err
			}
			table.RawSetString(k.String(), lv)
		}
		return table, nil
	case reflect.Struct:
		table, err := c.structToTable(v)
		if err != nil {
			return nil, err
		}
		return table, nil
	case reflect.Ptr:
		if val.IsNil() {
			return lua.LNil, nil
		}
		return c.goToLua(val.Elem().Interface())
	default:
		return nil, fmt.Errorf("unsupported type: %T", v)
	}
}

func (c *Config) luaToGo(lv lua.LValue, t reflect.Type) (interface{}, error) {
	if lv == lua.LNil {
		return reflect.Zero(t).Interface(), nil
	}

	switch lv.Type() {
	case lua.LTBool:
		if t.Kind() == reflect.Bool {
			return bool(lv.(lua.LBool)), nil
		}
		if t.Kind() == reflect.Interface {
			return bool(lv.(lua.LBool)), nil
		}
		return nil, fmt.Errorf("cannot convert boolean to %v", t)

	case lua.LTNumber:
		n := float64(lv.(lua.LNumber))
		switch t.Kind() {
		case reflect.Float64:
			return n, nil
		case reflect.Float32:
			return float32(n), nil
		case reflect.Int:
			return int(n), nil
		case reflect.Int64:
			return int64(n), nil
		case reflect.Int32:
			return int32(n), nil
		case reflect.Interface:
			return n, nil
		default:
			return nil, fmt.Errorf("cannot convert number to %v", t)
		}

	case lua.LTString:
		if t.Kind() != reflect.String && t.Kind() != reflect.Interface {
			return nil, fmt.Errorf("cannot convert string to %v", t)
		}
		return string(lv.(lua.LString)), nil

	case lua.LTTable:
		table := lv.(*lua.LTable)

		// Check if table is array-like (sequential numeric keys starting from 1)
		isArray := true
		maxn := table.MaxN()
		if maxn > 0 {
			for i := 1; i <= maxn; i++ {
				if table.RawGetInt(i) == lua.LNil {
					isArray = false
					break
				}
			}
		} else {
			isArray = false
		}

		switch t.Kind() {
		case reflect.Slice:
			slice := reflect.MakeSlice(t, 0, table.Len())
			if isArray {
				for i := 1; i <= maxn; i++ {
					val, err := c.luaToGo(table.RawGetInt(i), t.Elem())
					if err == nil {
						slice = reflect.Append(slice, reflect.ValueOf(val))
					}
				}
			} else {
				table.ForEach(func(_ lua.LValue, v lua.LValue) {
					val, err := c.luaToGo(v, t.Elem())
					if err == nil {
						slice = reflect.Append(slice, reflect.ValueOf(val))
					}
				})
			}
			return slice.Interface(), nil

		case reflect.Map:
			m := reflect.MakeMap(t)
			table.ForEach(func(k, v lua.LValue) {
				key, err := c.luaToGo(k, t.Key())
				if err != nil {
					return
				}
				val, err := c.luaToGo(v, t.Elem())
				if err != nil {
					return
				}
				m.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(val))
			})
			return m.Interface(), nil

		case reflect.Struct:
			if t == reflect.TypeOf(time.Time{}) {
				return c.luaTableToTime(table)
			}
			ptr := reflect.New(t)
			if err := c.luaToStruct(table, ptr.Interface()); err != nil {
				return nil, err
			}
			return ptr.Elem().Interface(), nil

		case reflect.Interface:
			// If it's an array-like table, convert to slice
			if isArray {
				result := make([]interface{}, 0, maxn)
				for i := 1; i <= maxn; i++ {
					val, err := c.luaToGo(table.RawGetInt(i), reflect.TypeOf((*interface{})(nil)).Elem())
					if err == nil {
						result = append(result, val)
					}
				}
				return result, nil
			}

			// Otherwise convert to map
			result := make(map[string]interface{})
			table.ForEach(func(k, v lua.LValue) {
				key := k.String()
				val, err := c.luaToGo(v, reflect.TypeOf((*interface{})(nil)).Elem())
				if err == nil {
					result[key] = val
				}
			})
			return result, nil

		default:
			return nil, fmt.Errorf("cannot convert table to %v", t)
		}
	default:
		return nil, fmt.Errorf("unsupported Lua type: %s", lv.Type())
	}
}

// Simple helper methods for common operations
func (c *Config) DoString(script string) error {
	return c.L.DoString(script)
}

func (c *Config) DoFile(filename string) error {
	return c.L.DoFile(filename)
}

// GetGlobal retrieves a global variable with type conversion
func (c *Config) GetGlobal(name string, target interface{}) error {
	lv := c.L.GetGlobal(name)
	if lv == lua.LNil {
		return &Error{
			Code:    ErrNotFound,
			Message: fmt.Sprintf("global variable '%s' not found", name),
		}
	}

	val := reflect.ValueOf(target)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("target must be a pointer")
	}

	converted, err := c.luaToGo(lv, val.Elem().Type())
	if err != nil {
		return err
	}

	val.Elem().Set(reflect.ValueOf(converted))
	return nil
}

// SetGlobal sets a global variable with type conversion
func (c *Config) SetGlobal(name string, value interface{}) error {
	lv, err := c.goToLua(value)
	if err != nil {
		return err
	}
	c.L.SetGlobal(name, lv)
	return nil
}

// Call invokes a Lua function with automatic type conversion
func (c *Config) Call(funcName string, args ...interface{}) ([]interface{}, error) {
	fn := c.L.GetGlobal(funcName)
	if fn == lua.LNil {
		return nil, fmt.Errorf("function '%s' not found", funcName)
	}

	luaArgs := make([]lua.LValue, len(args))
	for i, arg := range args {
		lv, err := c.goToLua(arg)
		if err != nil {
			return nil, err
		}
		luaArgs[i] = lv
	}

	err := c.L.CallByParam(lua.P{
		Fn:      fn,
		NRet:    lua.MultRet,
		Protect: true,
	}, luaArgs...)

	if err != nil {
		return nil, err
	}

	// Get all return values
	top := c.L.GetTop()
	result := make([]interface{}, top)

	for i := 1; i <= top; i++ {
		val := c.L.Get(i)

		if val.Type() == lua.LTTable {
			table := val.(*lua.LTable)
			maxn := table.MaxN()

			if maxn > 0 {
				// It's an array-like table
				slice := make([]interface{}, maxn)
				for j := 1; j <= maxn; j++ {
					elem := table.RawGetInt(j)
					if elem != lua.LNil {
						converted, err := c.luaToGo(elem, reflect.TypeOf((*interface{})(nil)).Elem())
						if err == nil {
							slice[j-1] = converted
						}
					}
				}
				result[i-1] = slice
			} else {
				// It's a regular table
				converted, err := c.luaToGo(val, reflect.TypeOf((*interface{})(nil)).Elem())
				if err != nil {
					return nil, err
				}
				result[i-1] = converted
			}
		} else {
			converted, err := c.luaToGo(val, reflect.TypeOf((*interface{})(nil)).Elem())
			if err != nil {
				return nil, err
			}
			result[i-1] = converted
		}
	}

	c.L.SetTop(0) // Clear the stack
	return result, nil
}

// RegisterConstants registers multiple constants at once
func (c *Config) RegisterConstants(constants map[string]interface{}) error {
	for name, value := range constants {
		if err := c.SetGlobal(name, value); err != nil {
			return err
		}
	}
	return nil
}

// LoadDirectory loads all .lua files from a directory
func (c *Config) LoadDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".lua") {
			path := filepath.Join(dir, entry.Name())
			if err := c.L.DoFile(path); err != nil {
				return &Error{
					Code:    ErrExecution,
					Message: fmt.Sprintf("failed to load %s", path),
					Cause:   err,
				}
			}
		}
	}
	return nil
}

// Eval evaluates a Lua expression and returns the result
func (c *Config) Eval(expr string) (interface{}, error) {
	err := c.L.DoString(fmt.Sprintf("__eval_result = %s", expr))
	if err != nil {
		return nil, err
	}

	result := c.L.GetGlobal("__eval_result")
	c.L.SetGlobal("__eval_result", lua.LNil) // Clean up

	return c.luaToGo(result, reflect.TypeOf((*interface{})(nil)).Elem())
}

// Helper function to convert Lua table to time.Time
func (c *Config) luaTableToTime(table *lua.LTable) (time.Time, error) {
	year := int(table.RawGetString("year").(lua.LNumber))
	month := int(table.RawGetString("month").(lua.LNumber))
	day := int(table.RawGetString("day").(lua.LNumber))
	hour := int(table.RawGetString("hour").(lua.LNumber))
	min := int(table.RawGetString("min").(lua.LNumber))
	sec := int(table.RawGetString("sec").(lua.LNumber))

	return time.Date(year, time.Month(month), day, hour, min, sec, 0, time.Local), nil
}
