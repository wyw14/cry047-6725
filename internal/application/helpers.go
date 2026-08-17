package application

import "encoding/json"

// jsonMarshal is the indirection used by recordAudit to avoid duplicate imports.
func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }
