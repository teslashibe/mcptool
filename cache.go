package mcptool

import (
	"reflect"
	"sync"
)

var (
	schemaCacheMu sync.Mutex
	schemaCache   = map[reflect.Type]map[string]any{}
)
