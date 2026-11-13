package fixture

import (
	"context"
	"fmt"
)

type key struct{}

func unchecked(ctx context.Context) string {
	return ctx.Value(key{}).(string) // want
}

func checked(ctx context.Context) string {
	if s, ok := ctx.Value(key{}).(string); ok {
		return s
	}
	return ""
}

func declared(v any) int {
	var n, ok = v.(int)
	if !ok {
		return 0
	}
	return n
}

func typeSwitch(v any) string {
	switch t := v.(type) {
	case fmt.Stringer:
		return t.String()
	}
	return ""
}

func stringer(v any) string {
	return v.(fmt.Stringer).String() // want
}
