package utils

import (
	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/encoder"
)

func StringifyJSON(v any) (string, error) {
	b, e := encoder.Encode(v, encoder.EscapeHTML)
	if e != nil {
		return "", e
	}
	return string(b), nil
}

func ParseJSON(data string, v any) error {
	return sonic.UnmarshalString(data, v)
}

func ModelDump(v any) (map[string]interface{}, error) {
	b, e := encoder.Encode(v, encoder.EscapeHTML)
	if e != nil {
		return nil, e
	}
	var dict map[string]interface{}
	e = sonic.Unmarshal(b, &dict)
	if e != nil {
		return nil, e
	}
	return dict, nil
}
