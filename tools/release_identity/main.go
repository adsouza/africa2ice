// Command release_identity prints the save and algorithm compatibility
// identity compiled into the exact source tree being released.
package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/adsouza/africa2ice/internal/application"
)

func main() {
	schema, algorithms := application.SupportedCompatibility()
	fmt.Printf("save_schema=%d\n", schema)
	value := reflect.ValueOf(algorithms)
	typeOf := value.Type()
	for index := 0; index < typeOf.NumField(); index++ {
		field := typeOf.Field(index)
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		fmt.Printf("algorithm.%s=%s\n", name, value.Field(index).String())
	}
}
