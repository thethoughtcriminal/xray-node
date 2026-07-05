package panel

import (
	"encoding/json"
	"fmt"
)

// JSONField stores JSON payload from 3x-ui as a string.
// Newer panel versions return settings/streamSettings/sniffing as objects;
// older versions use JSON-encoded strings. Both are accepted on write.
type JSONField string

func (f *JSONField) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*f = ""
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*f = JSONField(s)
		return nil
	}
	if !json.Valid(data) {
		return fmt.Errorf("invalid json field")
	}
	*f = JSONField(string(data))
	return nil
}

func (f JSONField) String() string {
	return string(f)
}

func (f JSONField) MarshalJSON() ([]byte, error) {
	if f == "" {
		return []byte(`""`), nil
	}
	return json.Marshal(string(f))
}
