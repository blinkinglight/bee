package bee

import "encoding/json"

func Unmarshal[T any](data []byte) (T, error) {
	var msg T
	if err := json.Unmarshal(data, &msg); err != nil {
		return msg, err
	}
	return msg, nil
}
