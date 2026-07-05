package panel

import "encoding/json"

func (r *APIResponse) UnmarshalObj(v any) error {
	if len(r.Obj) == 0 || string(r.Obj) == "null" {
		return nil
	}
	return json.Unmarshal(r.Obj, v)
}
