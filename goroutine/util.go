package goroutine

import (
	"reflect"
)

// CheckAndGetters 检查Getter是否为nil
func CheckAndGetters(getters ...func() any) {
	errS := "Getter检查未通过，存在nil返回值: "
	for _, f := range getters {
		r := f()
		vl := reflect.ValueOf(r)
		if r == nil {
			panic(errS + vl.Type().String())
		}
		if vl.Kind() == reflect.Ptr && vl.IsNil() {
			panic(errS + vl.Type().String())
		}
	}
}
