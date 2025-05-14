package controller

import (
	"reflect"
	"testing"
)

func TestTypeName(t *testing.T) {
	modelName := reflect.TypeOf(InputTextAction{}).Name()
	t.Log("type of modelName", modelName)
	modelName = reflect.TypeOf(&InputTextAction{}).Name()
	t.Log("type of pointer modelName", modelName)
}

func TestGenerateSchema(t *testing.T) {
	t.Log(GenerateSchema(&InputTextAction{}))
	t.Log(GenerateSchema(&NoParamsAction{}))
}

func TestValidateSchema(t *testing.T) {
	schemaString := GenerateSchema(&InputTextAction{})
	err := ValidateSchema(schemaString, map[string]interface{}{"index": 3, "text": "GPU compiler"})
	if err != nil {
		t.Error(err)
		t.Log("InputTextAction validate test failed")
	} else {
		t.Log("InputTextAction validate test success")
	}

	schemaString = GenerateSchema(&DoneAction{})
	err = ValidateSchema(schemaString, map[string]interface{}{"text": "Here are recommended books related to GPU compilers that you can read on Amazon Kindle:\n\n1. 'Die CUDA-Programmierung mit C++ meistern: Eine umfassende Einführung (German Edition)' by Jamie Flux – A comprehensive introduction to CUDA programming with C++. (German language, Kindle Edition available)\n\n2. 'Languages and Compilers for Parallel Computing: 28th International Workshop, LCPC 2015' by Xipeng Shen, Frank Mueller et al. – Research collection on languages and compilers for parallel computing, including topics relevant to GPU compilation. (English, Kindle Edition available)\n\n3. 'Languages and Compilers for Parallel Computing: 29th International Workshop, LCPC 2016' by Chen Ding, John Criswell et al. – Proceedings from a major parallel computing workshop, with research applicable to GPU and compiler technology. (English, Kindle Edition available)\n\n4. 'Evolving OpenMP for Evolving Architectures: 14th International Workshop on OpenMP, IWOMP 2018' by Bronis R. de Supinski, Pedro Valero-Lara et al. – Papers on OpenMP and parallel architectures, including content of interest to GPU programming and compilers. (English, Kindle Edition available)\n\nThese titles are relevant for learning about GPU compilers, parallel languages, and related compiler technology. Some more technical or research-focused books are also available in paperback only, but not for Kindle.\n\nTask complete. Success!", "success": true})
	if err != nil {
		t.Error(err)
		t.Log("DoneAction validate test failed")
	} else {
		t.Log("DoneAction validate test success")
	}

	schemaString = GenerateSchema(&GoToUrlAction{})
	err = ValidateSchema(schemaString, map[string]interface{}{"url": "https://www.google.com"})
	if err != nil {
		t.Error(err)
		t.Log("GoToUrlAction validate test failed")
	} else {
		t.Log("GoToUrlAction validate test success")
	}

	schemaString = GenerateSchema(&ClickElementAction{})
	err = ValidateSchema(schemaString, map[string]interface{}{"index": 5})
	if err != nil {
		t.Error(err)
		t.Log("ClickElementAction validate test failed")
	} else {
		t.Log("ClickElementAction validate test success")
	}

	// should be error
	schemaString = GenerateSchema(&GoToUrlAction{})
	err = ValidateSchema(schemaString, map[string]interface{}{"index": 5})
	if err != nil {
		t.Log("GoToUrlAction & ClickElementAction validate test success")
	} else {
		t.Error("it should be error")
		t.Log("GoToUrlAction & ClickElementAction validate test failed")
	}
}
