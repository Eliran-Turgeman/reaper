package decision

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// EncodeState accepts exactly one representation. Structured values are
// restricted to the object/array variants supported by the decision endpoint.
func EncodeState(text string, structured json.RawMessage) (json.RawMessage, error) {
	if len(structured) == 0 {
		if strings.TrimSpace(text) == "" {
			return nil, fmt.Errorf("state is required")
		}
		return json.Marshal(text)
	}
	if text != "" {
		return nil, fmt.Errorf("text and structured state are mutually exclusive")
	}
	data := bytes.TrimSpace(structured)
	if !json.Valid(data) {
		return nil, fmt.Errorf("invalid structured state")
	}
	if data[0] != '{' && data[0] != '[' {
		return nil, fmt.Errorf("structured state must be an object or array")
	}
	return json.RawMessage(data), nil
}

func (c *NoulCriteria) Validate() error {
	if c != nil && (strings.TrimSpace(c.True) == "" || strings.TrimSpace(c.False) == "") {
		return fmt.Errorf("noul criteria require nonempty true and false descriptions")
	}
	return nil
}
