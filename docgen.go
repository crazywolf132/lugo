package lugo

import (
	"fmt"
	"reflect"
	"strings"
)

type DocGenerator struct {
	Format           string
	TypeDescriptions map[string]string
	IncludeExamples  bool
}

func (c *Config) GenerateDocs(v interface{}, gen DocGenerator) (string, error) {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	var b strings.Builder
	b.WriteString("# Configuration Reference\n\n")

	if err := generateFieldDocs(&b, t, "", gen); err != nil {
		return "", err
	}

	return b.String(), nil
}

func generateFieldDocs(b *strings.Builder, t reflect.Type, prefix string, gen DocGenerator) error {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// Skip unexported fields
		if !field.IsExported() {
			continue
		}

		luaTag := field.Tag.Get("lua")
		if luaTag == "" {
			continue
		}

		path := luaTag
		if prefix != "" {
			path = prefix + "." + luaTag
		}

		// Write field header
		fmt.Fprintf(b, "## %s\n\n", path)

		// Write type information
		typeDesc := getTypeDescription(field.Type, gen.TypeDescriptions)
		fmt.Fprintf(b, "**Type:** `%s`\n\n", typeDesc)

		// Write documentation
		if doc := field.Tag.Get("doc"); doc != "" {
			fmt.Fprintf(b, "%s\n\n", doc)
		}

		// Write validation rules
		if validate := field.Tag.Get("validate"); validate != "" {
			fmt.Fprintf(b, "**Validation:**\n")
			rules := strings.Split(validate, ",")
			for _, rule := range rules {
				fmt.Fprintf(b, "- %s\n", rule)
			}
			fmt.Fprintln(b)
		}

		// Write example if available
		if gen.IncludeExamples {
			if example := field.Tag.Get("example"); example != "" {
				fmt.Fprintf(b, "**Example:** `%s`\n\n", example)
			}
		}

		// Handle nested structs
		if field.Type.Kind() == reflect.Struct {
			if err := generateFieldDocs(b, field.Type, path, gen); err != nil {
				return err
			}
		}
	}

	return nil
}

func getTypeDescription(t reflect.Type, descriptions map[string]string) string {
	// Check for custom type description
	if desc, ok := descriptions[t.String()]; ok {
		return desc
	}

	// Default type descriptions
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int64:
		return "integer"
	case reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice:
		return "array"
	case reflect.Map:
		return "map"
	case reflect.Struct:
		return "table"
	default:
		return t.String()
	}
}
