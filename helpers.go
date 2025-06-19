package bee

import "encoding/json"

// Unmarshal unmarshals JSON data into a struct of type T.
// It returns the struct and an error if unmarshaling fails.

func Unmarshal[T any](data []byte) (T, error) {
	var msg T
	if err := json.Unmarshal(data, &msg); err != nil {
		return msg, err
	}
	return msg, nil
}

// MustUnmarshal unmarshals JSON data into a struct of type T.
// It panics if unmarshaling fails, so it should be used when you are sure
// that the data is valid and will not cause an error.
// This is useful for tests or when you want to ensure that the data is always valid.
func MustUnmarshal[T any](data []byte) T {
	t, err := Unmarshal[T](data)
	if err != nil {
		panic(err)
	}
	return t
}
