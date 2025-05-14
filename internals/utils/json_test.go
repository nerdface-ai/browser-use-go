package utils_test

import (
	"testing"

	"github.com/nerdface-ai/browser-use-go/internals/utils"
)

func TestParseJSON(t *testing.T) {
	var v map[string]interface{}
	err := utils.ParseJSON(`{"a": 1}`, &v)
	if err != nil {
		t.Fatal(err)
	}
	if va, ok := v["a"].(float64); !ok || va != 1 {
		t.Fatal("expected a = 1, got", v["a"])
	}
}

type TestModel struct {
	A string `json:"a"`
}

func TestStringifyJSON(t *testing.T) {
	v := TestModel{A: "1"}
	json, err := utils.StringifyJSON(v)
	if err != nil {
		t.Fatal(err)
	}
	if json != `{"a":"1"}` {
		t.Fatal("expected a = 1, got", json)
	}
}

func TestModelDump(t *testing.T) {
	v := TestModel{A: "1"}
	dump, err := utils.ModelDump(v)
	if err != nil {
		t.Fatal(err)
	}
	if dump["a"] != "1" {
		t.Fatal("expected a = 1, got", dump["a"])
	}
}
