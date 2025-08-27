package utils //nolint:revive // This is a utility package.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUnmarshal   = errors.New("unmarshal failed")
	ErrKeyNotFound = errors.New("key not found")
)

// UnmarshalToSlice unmarshals a JSON response data to a slice.
func UnmarshalToSlice[T any](responseData json.RawMessage, key string) ([]T, error) {
	data, found := searchForKey(responseData, key)
	if !found {
		return nil, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}

	var slice []T

	if err := json.Unmarshal(data, &slice); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnmarshal, key)
	}

	return slice, nil
}

// UnmashalToPointer unmarshals a JSON response data to a pointer.
func UnmarshalToPointer[T any](responseData json.RawMessage, key string) (*T, error) {
	data, found := searchForKey(responseData, key)
	if !found {
		return nil, fmt.Errorf("%w: %s", ErrKeyNotFound, key)
	}

	var t T

	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnmarshal, key)
	}

	return &t, nil
}

// searchForKey searches for a key in a JSON object, including nested objects.
// If the key is found, the value is returned as json.RawMessage.
// The function recurses if the value is an object.
// If the key is not found, the function returns false.
// nolint: gocognit
func searchForKey(data json.RawMessage, targetKey string) (json.RawMessage, bool) {
	decoder := json.NewDecoder(bytes.NewReader(data))

	// Ensure we start processing as an object
	token, err := decoder.Token()
	if err != nil || token != json.Delim('{') {
		return nil, false
	}

	for decoder.More() {
		// Read the key
		key, ok := getKey(decoder)
		if !ok {
			return nil, false
		}

		// If the key matches, return the corresponding value
		if key == targetKey {
			return getValue(decoder)
		}

		// Check if the value is an object to recurse
		var rawValue json.RawMessage
		if err := decoder.Decode(&rawValue); err != nil {
			return nil, false
		}

		// Check if it's a nested object and recurse
		if isJSONObject(rawValue) {
			// Recursively search for the key in the nested object
			if result, found := searchForKey(rawValue, targetKey); found {
				return result, true
			}
		}
	}

	return nil, false
}

// getValue reads the value from a JSON decoder.
func getValue(decoder *json.Decoder) (json.RawMessage, bool) {
	var value json.RawMessage
	if err := decoder.Decode(&value); err != nil {
		return nil, false
	}

	return value, true
}

// getKey reads the key from a JSON decoder.
func getKey(decoder *json.Decoder) (string, bool) {
	keyToken, err := decoder.Token()
	if err != nil {
		return "", false
	}

	key, ok := keyToken.(string)
	if !ok {
		return "", false
	}

	return key, true
}

// isJSONObject checks if a JSON RawMessage is an object.
func isJSONObject(data json.RawMessage) bool {
	trimmed := strings.TrimSpace(string(data))

	return len(trimmed) > 0 && trimmed[0] == '{'
}
