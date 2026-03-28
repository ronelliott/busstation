package redis

import "encoding/json"

// Codec handles serialization and deserialization of bus message payloads.
// Implement this interface to use a custom encoding (e.g. protobuf, msgpack).
type Codec[T any] interface {
	Marshal(T) ([]byte, error)
	Unmarshal([]byte) (T, error)
}

// JSONCodec is the default Codec. It uses encoding/json.
type JSONCodec[T any] struct{}

func (JSONCodec[T]) Marshal(v T) ([]byte, error) {
	return json.Marshal(v)
}

func (JSONCodec[T]) Unmarshal(b []byte) (T, error) {
	var v T
	err := json.Unmarshal(b, &v)
	return v, err
}
