package lugo

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
)

// Generator provides a fluent API for generating Lua code
type Generator struct {
	buffer bytes.Buffer
	indent int
}

// NewGenerator creates a new Lua code generator
func NewGenerator() *Generator {
	return &Generator{
		indent: 0,
	}
}

// String returns the generated Lua code as a string
func (g *Generator) String() string {
	return g.buffer.String()
}

// Reset clears the generator buffer
func (g *Generator) Reset() {
	g.buffer.Reset()
	g.indent = 0
}

// writeIndent writes the current indentation
func (g *Generator) writeIndent() {
	g.buffer.WriteString(strings.Repeat("    ", g.indent))
}

// Table starts a new table declaration
func (g *Generator) Table(name string) *Generator {
	g.writeIndent()
	if name != "" {
		g.buffer.WriteString(name)
		g.buffer.WriteString(" = ")
	}
	g.buffer.WriteString("{\n")
	g.indent++
	return g
}

// EndTable closes a table declaration
func (g *Generator) EndTable() *Generator {
	g.indent--
	g.writeIndent()
	g.buffer.WriteString("}")
	if g.indent == 0 {
		g.buffer.WriteString("\n")
	} else {
		g.buffer.WriteString(",\n")
	}
	return g
}

// Field adds a field to the current table
func (g *Generator) Field(name string, value interface{}) *Generator {
	g.writeIndent()
	if strings.Contains(name, " ") || strings.Contains(name, "-") {
		g.buffer.WriteString(fmt.Sprintf("[\"%s\"]", name))
	} else {
		g.buffer.WriteString(name)
	}
	g.buffer.WriteString(" = ")
	g.writeValue(value)
	g.buffer.WriteString(",\n")
	return g
}

// Array adds an array to the current table
func (g *Generator) Array(values ...interface{}) *Generator {
	g.writeIndent()
	g.buffer.WriteString("{ ")
	for i, v := range values {
		if i > 0 {
			g.buffer.WriteString(", ")
		}
		g.writeValue(v)
	}
	g.buffer.WriteString(" },\n")
	return g
}

// Comment adds a comment
func (g *Generator) Comment(text string) *Generator {
	g.writeIndent()
	g.buffer.WriteString("-- ")
	g.buffer.WriteString(text)
	g.buffer.WriteString("\n")
	return g
}

// Raw adds raw Lua code
func (g *Generator) Raw(code string) *Generator {
	g.writeIndent()
	// Remove any existing indentation from the code
	code = strings.TrimSpace(code)
	g.buffer.WriteString(code)
	g.buffer.WriteString("\n")
	return g
}

// Function adds a function declaration
func (g *Generator) Function(name string, params ...string) *Generator {
	g.writeIndent()
	if name != "" {
		g.buffer.WriteString("function ")
		g.buffer.WriteString(name)
	}
	g.buffer.WriteString("(")
	g.buffer.WriteString(strings.Join(params, ", "))
	g.buffer.WriteString(")\n")
	g.indent++
	return g
}

// EndFunction closes a function declaration
func (g *Generator) EndFunction() *Generator {
	g.indent--
	g.writeIndent()
	g.buffer.WriteString("end\n")
	return g
}

// writeValue writes a value in Lua format
func (g *Generator) writeValue(v interface{}) {
	if v == nil {
		g.buffer.WriteString("nil")
		return
	}

	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.String:
		g.buffer.WriteString(fmt.Sprintf("%q", v))
	case reflect.Bool:
		g.buffer.WriteString(fmt.Sprintf("%v", v))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		g.buffer.WriteString(fmt.Sprintf("%v", v))
	case reflect.Slice, reflect.Array:
		g.buffer.WriteString("{ ")
		for i := 0; i < val.Len(); i++ {
			if i > 0 {
				g.buffer.WriteString(", ")
			}
			g.writeValue(val.Index(i).Interface())
		}
		g.buffer.WriteString(" }")
	case reflect.Map:
		g.buffer.WriteString("{\n")
		g.indent++
		iter := val.MapRange()
		for iter.Next() {
			g.writeIndent()
			k := iter.Key()
			if k.Kind() == reflect.String {
				if strings.Contains(k.String(), " ") || strings.Contains(k.String(), "-") {
					g.buffer.WriteString(fmt.Sprintf("[\"%s\"]", k))
				} else {
					g.buffer.WriteString(k.String())
				}
			} else {
				g.buffer.WriteString(fmt.Sprintf("[%v]", k))
			}
			g.buffer.WriteString(" = ")
			g.writeValue(iter.Value().Interface())
			g.buffer.WriteString(",\n")
		}
		g.indent--
		g.writeIndent()
		g.buffer.WriteString("}")
	default:
		g.buffer.WriteString(fmt.Sprintf("%v", v))
	}
}
