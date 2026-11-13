package validator

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
)

func validateIn(t reflect.StructField, v reflect.Value, params string) error {
	list := strings.Split(params, "|")

	if slices.Contains(list, v.String()) {
		return nil
	}

	return fmt.Errorf("%s must be one of %s", t.Name, strings.Join(list, ", "))
}
