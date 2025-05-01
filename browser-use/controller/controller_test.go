package controller_test

import (
	"encoding/json"
	"nerdface-ai/browser-use-go/browser-use/controller"
	"testing"

	"github.com/moznion/go-optional"
)

type tempFunctionResult struct {
	jsonString string
	typeName   string
}

func tempFunction(arg1 interface{}, arg2 interface{}) interface{} {
	b, _ := json.Marshal(arg1)
	return &controller.ActionResult{
		IsDone:           optional.Some(true),
		ExtractedContent: optional.Some(string(b)),
		IncludeInMemory:  true,
		Success:          optional.Some(true),
	}
}

func TestNewController(t *testing.T) {
	c := controller.NewController()
	t.Log(c)
}

func TestRegisterAction(t *testing.T) {
	c := controller.NewController()
	if len(c.Registry.Registry.Actions) != 0 {
		t.Error("expected 0 actions, got", len(c.Registry.Registry.Actions))
	}
	c.RegisterAction("InputTextAction", "input text", controller.InputTextAction{}, tempFunction, []string{}, nil)
	if len(c.Registry.Registry.Actions) != 1 {
		t.Error("expected 1 action, got", len(c.Registry.Registry.Actions))
	}
	c.RegisterAction("DoneAction", "done action", controller.DoneAction{}, tempFunction, []string{}, nil)
	if len(c.Registry.Registry.Actions) != 2 {
		t.Error("expected 2 actions, got", len(c.Registry.Registry.Actions))
	}
}

func TestExecuteAction(t *testing.T) {
	c := controller.NewController()
	c.RegisterAction("InputTextAction", "input text", controller.InputTextAction{}, tempFunction, []string{}, nil)
	c.RegisterAction("DoneAction", "done action", controller.DoneAction{}, tempFunction, []string{}, nil)
	_, err := c.ExecuteAction(&controller.ActionModel{
		Actions: map[string]interface{}{
			"InputTextAction": map[string]interface{}{
				"text": "test",
			},
		},
	}, nil, nil, nil, nil)
	if err == nil || err.Error() != "invalid schema" {
		t.Error("this should be error with 'invalid schema', but get ", err)
	}
	result, err := c.ExecuteAction(&controller.ActionModel{
		Actions: map[string]interface{}{
			"InputTextAction": map[string]interface{}{
				"text":  "test",
				"index": 3,
			},
		},
	}, nil, nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	t.Log(result)
	b, _ := json.Marshal(result)
	t.Log(string(b))
}
