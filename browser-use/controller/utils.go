package controller

import (
	"encoding/json"
	"errors"

	"github.com/invopop/jsonschema"
	"github.com/xeipuuv/gojsonschema"
)

func removeRefRecursive(v interface{}) interface{} {
	switch vv := v.(type) {
	case map[string]interface{}:
		delete(vv, "$ref")
		for k, val := range vv {
			vv[k] = removeRefRecursive(val)
		}
		return vv
	case []interface{}:
		for i, val := range vv {
			vv[i] = removeRefRecursive(val)
		}
		return vv
	default:
		return v
	}
}

func GenerateSchema(modelName string, typeDefinition interface{}) string {
	s := jsonschema.Reflect(typeDefinition)
	s.Definitions[modelName].Title = modelName
	b, err := json.Marshal(s.Definitions[modelName])
	if err != nil {
		panic(err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		panic(err)
	}
	// remove all $ref
	m = removeRefRecursive(m).(map[string]interface{})
	b2, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b2)
}

func ValidateSchema(schemaString string, dataString string) error {
	schemaLoader := gojsonschema.NewStringLoader(schemaString)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return err
	}
	validResult, err := schema.Validate(gojsonschema.NewStringLoader(dataString))
	if err != nil {
		return err
	}
	if validResult.Valid() {
		return nil
	}
	return errors.New("invalid schema")
}
