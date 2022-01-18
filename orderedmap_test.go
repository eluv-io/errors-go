package errors

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderedMap_basic(t *testing.T) {
	am := orderedMap{}
	assertFields(t, &am)

	am.Append("key1", 1, "key2", "2", "key3", io.EOF)
	assertFields(t, &am, "key1", 1, "key2", "2", "key3", io.EOF)

	am.Append("key4", 4)
	assertFields(t, &am, "key1", 1, "key2", "2", "key3", io.EOF, "key4", 4)

	am.Delete("key2")
	am.Delete("key4")
	assertFields(t, &am, "key1", 1, "key3", io.EOF)

	am.Set("key3", 3)
	assertFields(t, &am, "key1", 1, "key3", 3)

	am.Set("key4", 4)
	assertFields(t, &am, "key1", 1, "key3", 3, "key4", 4)
}

func TestOrderedMap_basicPtr(t *testing.T) {
	am := new(orderedMap)
	assertFields(t, am)

	am.Append("key1", 1, "key2", "2", "key3", io.EOF)
	assertFields(t, am, "key1", 1, "key2", "2", "key3", io.EOF)

	am.Append("key4", 4)
	assertFields(t, am, "key1", 1, "key2", "2", "key3", io.EOF, "key4", 4)

	am.Delete("key2")
	am.Delete("key4")
	assertFields(t, am, "key1", 1, "key3", io.EOF)

	am.Set("key3", 3)
	assertFields(t, am, "key1", 1, "key3", 3)

	am.Set("key4", 4)
	assertFields(t, am, "key1", 1, "key3", 3, "key4", 4)
}

func TestOrderedMap_Append(t *testing.T) {
	am := new(orderedMap)

	am.Append("a single value")
	assertFields(t, am, "a single value", "<missing>")

	am.Clear()
	am.Append("k1", "v1", "k2", "v2", "k3", "v3", "k4", "v4", "k5")
	assertFields(t, am, "k1", "v1", "k2", "v2", "k3", "v3", "k4", "v4", "k5", "<missing>")
}

func TestOrderedMap_String(t *testing.T) {
	am := new(orderedMap)

	am.Append("k1", "v1", "k2", "v2")
	assert.Equal(t, "{k1:v1, k2:v2}", am.String())

	am.Append(3, "v3")
	assert.Equal(t, "{k1:v1, k2:v2, 3:v3}", am.String())

	am.Append(nil, "v4")
	assert.Equal(t, "{k1:v1, k2:v2, 3:v3, :v4}", am.String())

}
func assertFields(t *testing.T, am *orderedMap, values ...interface{}) {
	require.Len(t, *am, len(values))
	for idx, val := range values {
		require.Equal(t, val, (*am)[idx])
	}
}
