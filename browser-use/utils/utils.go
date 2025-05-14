package utils

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/log"
	"github.com/joho/godotenv"
)

func LoadEnv(envPath string) {
	err := godotenv.Load(envPath)
	if err != nil {
		log.Print(err)
		log.Debug("Error loading .env file")
	}
}

// ConvertToStringMap converts a map[string]any to map[string]string.
// It iterates through the input map and includes only the key-value pairs
// where the value is of type string.
func ConvertToStringMap(inputMap map[string]any) map[string]string {
	resultMap := make(map[string]string)
	if inputMap == nil {
		return resultMap
	}
	for key, value := range inputMap {
		if strValue, ok := value.(string); ok {
			resultMap[key] = strValue
		}
	}
	return resultMap
}

// GetDefaultValue returns the default value if the key is not found in the data
// Example: GetDefaultValue(data, "headless", false)
func GetDefaultValue[T any](data map[string]any, key string, defaultValue T) T {
	if value, ok := data[key]; ok {
		if value, ok := value.(T); ok {
			return value
		}
	}
	return defaultValue
}

func ConvertToSliceOfInt(input any) ([]int, error) {
	if input == nil {
		return nil, fmt.Errorf("input cannot be nil")
	}
	slice, ok := input.([]any)
	if !ok {
		return nil, fmt.Errorf("input is not a slice of interface")
	}
	result := make([]int, 0, len(slice))

	for i, elem := range slice {
		if elem == nil {
			return nil, fmt.Errorf("element at index %d is nil", i)
		}
		// if typeof elem is string, convert to int using strconv.Atoi, if it's int, just use it
		if strValue, ok := elem.(string); ok {
			value, err := strconv.Atoi(strValue)
			if err != nil {
				return nil, fmt.Errorf("failed to convert string to int: %s", err)
			}
			result = append(result, value)
			continue
		}
		value, ok := elem.(int)
		if !ok {
			return nil, fmt.Errorf("element at index %d is not of type %T, but %T", i, value, elem)
		}
		result = append(result, value)
	}
	return result, nil
}

func ConvertToOptional[T any](value any) *T {
	if value, ok := value.(T); ok {
		return &value
	}
	return nil
}
