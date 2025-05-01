package controller

import (
	"encoding/json"
	"errors"
	"reflect"

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

func GenerateSchema(typeDefinition interface{}) string {
	var modelName string
	if reflect.TypeOf(typeDefinition).Kind() == reflect.Ptr {
		modelName = reflect.TypeOf(typeDefinition).Elem().Name()
	} else {
		modelName = reflect.TypeOf(typeDefinition).Name()
	}
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

func ValidateSchema(schemaString string, data map[string]interface{}) error {
	schemaLoader := gojsonschema.NewStringLoader(schemaString)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return err
	}
	validResult, err := schema.Validate(gojsonschema.NewGoLoader(data))
	if err != nil {
		return err
	}
	if validResult.Valid() {
		return nil
	}
	return errors.New("invalid schema")
}
