package hack

import (
	"reflect"
	"unsafe"
)

type MutableString string

func String(b []byte) MutableString {
	var s MutableString
	if len(b) == 0 {
		return ""
	}
	pbytes := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	pstring := (*reflect.StringHeader)(unsafe.Pointer(&s))
	pstring.Data = pbytes.Data
	pstring.Len = pbytes.Len
	return s
}
