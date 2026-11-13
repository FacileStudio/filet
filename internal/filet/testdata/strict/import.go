package fixture

import "C" // want

import (
	"fmt"
	"reflect" // want
	"strings"
	"unsafe" // want
)

func describe(v any) string {
	return strings.ToLower(fmt.Sprint(reflect.TypeOf(v), unsafe.Sizeof(v)))
}
