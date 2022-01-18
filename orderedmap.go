package errors

import (
	"fmt"
	"strings"
)

// orderedMap is a data structure that behaves like a regular map, but stores its key-value pairs in a flat array. KV
// pairs are stored in insertion order.
//
// Warning: this type should only be used to store a small number of KV pairs with infrequent modifications and lookups.
type orderedMap []interface{}

func (a *orderedMap) Append(kvs ...interface{}) {
	l2 := (len(kvs) + 1) / 2 * 2
	a.Grow(l2)
	for i := 0; i+1 < len(kvs); i += 2 {
		a.Set(toString(kvs[i]), kvs[i+1])
	}
	if l2 > len(kvs) {
		a.Set(toString(kvs[len(kvs)-1]), "<missing>")
	}
}

func (a *orderedMap) Set(key string, val interface{}) {
	for i := 0; i+1 < len(*a); i += 2 {
		if (*a)[i] == key {
			(*a)[i+1] = val
			return
		}
	}
	*a = append(*a, key, val)
}

func (a *orderedMap) Delete(key string) {
	for i := 0; i+1 < len(*a); i += 2 {
		if (*a)[i] == key {
			copy((*a)[i:], (*a)[i+2:])
			*a = (*a)[:len(*a)-2]
		}
	}
}

func (a *orderedMap) Clear() {
	*a = (*a)[:0]
}

func (a orderedMap) Get(key string) (interface{}, bool) {
	for i := 0; i+1 < len(a); i += 2 {
		if a[i] == key {
			return a[i+1], true
		}
	}
	return nil, false
}

func (a orderedMap) String() string {
	sb := strings.Builder{}
	sb.WriteString("{")
	for i := 0; i+1 < len(a); i += 2 {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(a[i].(string))
		sb.WriteString(":")
		sb.WriteString(fmt.Sprint(a[i+1]))
	}
	sb.WriteString("}")
	return sb.String()
}

func toString(val interface{}) string {
	if val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprint(val)
}

func (a *orderedMap) Grow(i int) {
	newCap := len(*a) + i
	if newCap > cap(*a) {
		n := make([]interface{}, len(*a), newCap)
		copy(n, *a)
		*a = n
	}
}
