/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package secrets

import (
	"encoding/json"
	"fmt"
)

func ParseJSONToMap(jsonStr string) (map[string][]byte, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonStr), &result)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	secretData := make(map[string][]byte)
	for key, value := range result {
		switch v := value.(type) {
		case map[string]interface{}, []interface{}: // For nested structures, marshal back to JSON
			bytes, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("error marshalling value for key %s: %w", key, err)
			}
			secretData[key] = bytes
		default:
			bytes, err := json.Marshal(v)
			if err != nil {
				return nil, fmt.Errorf("error marshalling value for key %s: %w", key, err)
			}
			secretData[key] = bytes
		}
	}

	return secretData, nil
}

// mapToStructuredJSON takes a map[string]interface{} as input, where some values may be JSON strings,
// and returns an interface{} that represents the structured JSON object.
func MapToStructuredJSON(input map[string]interface{}) (interface{}, error) {
	// Create a result map to hold our structured data.
	result := make(map[string]interface{})

	for key, value := range input {
		switch v := value.(type) {
		case string:
			// Attempt to unmarshal the string into a map if it's JSON.
			// This covers cases where the string is actually a JSON object or array.
			var jsonVal interface{}
			err := json.Unmarshal([]byte(v), &jsonVal)
			if err == nil {
				// If successful, use the unmarshaled value.
				result[key] = jsonVal
			} else {
				// If the string is not JSON, remove surrounding quotes if present and use the string directly.
				if len(v) > 1 && v[0] == '"' && v[len(v)-1] == '"' {
					result[key] = v[1 : len(v)-1]
				} else {
					result[key] = v
				}
			}
		default:
			// For all other types, use the value directly.
			result[key] = value
		}
	}

	return result, nil
}
