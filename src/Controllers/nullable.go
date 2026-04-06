package Controllers

import "encoding/json"

// NullableString distinguishes between a field being absent from JSON
// and a field being explicitly set to null.
// IsSet=false → field not present in request (skip update)
// IsSet=true, Value=nil → field explicitly set to null
// IsSet=true, Value=&"..." → field set to a string value
type NullableString struct {
	Value *string
	IsSet bool
}

func (n *NullableString) UnmarshalJSON(data []byte) error {
	n.IsSet = true
	if string(data) == "null" {
		n.Value = nil
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	n.Value = &s
	return nil
}
