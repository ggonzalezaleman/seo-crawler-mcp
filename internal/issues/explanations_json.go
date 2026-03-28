package issues

import "encoding/json"

// ExplanationsJSON returns the Explanations map serialized as a JSON string.
// This is intended for embedding in HTML reports as a JavaScript object.
func ExplanationsJSON() string {
	b, err := json.Marshal(Explanations)
	if err != nil {
		return "{}"
	}
	return string(b)
}
