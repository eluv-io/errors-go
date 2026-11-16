package errors

import (
	"encoding/json"
	"sync/atomic"
)

// counter for global order of all unmarshalled orderedKeys
var keyPosCounter uint64

type orderedKey struct {
	key string
	pos uint64
}

func (p *orderedKey) UnmarshalText(text []byte) error {
	p.key = string(text)
	p.pos = atomic.AddUint64(&keyPosCounter, 1)
	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type ordereKeys []orderedKey

func (o ordereKeys) Len() int {
	return len(o)
}

func (o ordereKeys) Less(i, j int) bool {
	return o[i].pos < o[j].pos
}

func (o ordereKeys) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

type valOrMap struct {
	val interface{}
	m   map[orderedKey]valOrMap
	arr json.RawMessage // the JSON text of val if it is an array, decoded again for a cause, see causeArray
}

func (s *valOrMap) UnmarshalJSON(b []byte) error {
	err := json.Unmarshal(b, &s.m)
	if err == nil {
		return nil
	}
	if err = json.Unmarshal(b, &s.val); err != nil {
		return err
	}
	if _, ok := s.val.([]interface{}); ok {
		s.arr = append(json.RawMessage(nil), b...)
	}
	return nil
}

// causeArray returns the errors of an array, the form an ErrorArray marshals to. Decoding the array again from its
// JSON text keeps the order of the fields of the errors in it. It returns false if s holds no array.
func (s valOrMap) causeArray() (*ErrorArray, bool) {
	if s.arr == nil {
		return nil, false
	}
	arr := &ErrorArray{}
	if err := arr.UnmarshalJSON(s.arr); err != nil {
		return nil, false
	}
	return arr, true
}

func (s valOrMap) Get() interface{} {
	if s.m != nil {
		err := Error{}
		err.unmarshalFrom(s.m)
		return &err
	}
	return s.val
}

func (s valOrMap) AsError() error {
	if len(s.m) > 0 {
		err := Error{}
		err.unmarshalFrom(s.m)
		return &err
	}
	if s.val != nil {
		return Str(toString(s.val))
	}
	return nil
}
