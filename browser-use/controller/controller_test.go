package controller_test

import (
	"encoding/json"
	"nerdface-ai/browser-use-go/browser-use/browser"
	"nerdface-ai/browser-use-go/browser-use/controller"
	"testing"
	"time"

	"github.com/moznion/go-optional"
)

type tempFunctionResult struct {
	jsonString string
	typeName   string
}

func tempFunction(arg1 interface{}, arg2 map[string]interface{}) (*controller.ActionResult, error) {
	b, _ := json.Marshal(arg1)
	return &controller.ActionResult{
		IsDone:           optional.Some(true),
		ExtractedContent: optional.Some(string(b)),
		IncludeInMemory:  true,
		Success:          optional.Some(true),
	}, nil
}

func TestNewController(t *testing.T) {
	c := controller.NewController()
	t.Log(c)
}

func TestRegisterAction(t *testing.T) {
	c := controller.NewController()
	t.Log(c)
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

func TestExecuteClickElement(t *testing.T) {
	c := controller.NewController()
	b := browser.NewBrowser(browser.BrowserConfig{
		"headless": true,
	})
	defer b.Close()
	bc := b.NewContext()
	defer bc.Close()

	bc.NavigateTo("https://www.naver.com")
	time.Sleep(1 * time.Second)

	// ------------------ buildDomTree.js -> set SelectorMap --------------------------
	// this will be done in Agent.Step() later
	currentState := bc.GetState(false)
	time.Sleep(1 * time.Second)

	session := bc.GetSession()
	session.CachedState = currentState
	// ------------------ buildDomTree.js -> set SelectorMap --------------------------

	// register controller service ahead of execution
	c.RegisterAction(
		"ClickElementAction",
		"click element",
		controller.ClickElementAction{},
		c.ClickElementByIndex,
		[]string{},
		nil,
	)

	actionModel := &controller.ActionModel{
		Actions: map[string]interface{}{
			"ClickElementAction": map[string]interface{}{
				"index": 8, // 0
			},
		},
	}

	result, err := c.ExecuteAction(actionModel, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	json, _ := json.Marshal(result)
	t.Log(string(json))
}

func TestExecuteInputText(t *testing.T) {
	c := controller.NewController()
	b := browser.NewBrowser(browser.BrowserConfig{
		"headless": true,
	})
	defer b.Close()
	bc := b.NewContext()
	defer bc.Close()

	bc.NavigateTo("https://www.google.com")
	time.Sleep(1 * time.Second)

	// ------------------ buildDomTree.js -> set SelectorMap --------------------------
	// this will be done in Agent.Step() later
	currentState := bc.GetState(false)
	time.Sleep(1 * time.Second)

	session := bc.GetSession()
	session.CachedState = currentState
	// ------------------ buildDomTree.js -> set SelectorMap --------------------------

	// register controller service ahead of execution
	c.RegisterAction(
		"InputTextAction",
		"input text",
		controller.InputTextAction{},
		c.InputText,
		[]string{},
		nil,
	)
	selectorMap := bc.GetSelectorMap()
	key := -1
	for k, v := range *selectorMap {
		if v.TagName == "textarea" {
			key = k
			break
		}
	}
	if key == -1 {
		t.Error("textarea not found")
		return
	}

	actionModel := &controller.ActionModel{
		Actions: map[string]interface{}{
			"InputTextAction": map[string]interface{}{
				"index": key,
				"text":  "Seoul weather",
			},
		},
	}

	result, err := c.ExecuteAction(actionModel, bc, nil, nil, nil)
	if err != nil {
		t.Error(err)
	}
	json, _ := json.Marshal(result)
	t.Log(string(json))
}
