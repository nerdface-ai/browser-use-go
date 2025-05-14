package agent

import (
	"context"
	"encoding/json"
	"nerdface-ai/browser-use-go/browser-use/controller"
	"nerdface-ai/browser-use-go/browser-use/utils"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"
)

func TestOpenAIChatModel(t *testing.T) {
	// task := "do google search to find images of Elon Musk's wife"
	utils.LoadEnv("../../.env")
	ctx := context.Background()
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   "gpt-4o-mini",
		Timeout: 30 * time.Second,
		APIKey:  os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		t.Fatal(err)
	}

	response, err := model.Generate(ctx, []*schema.Message{
		{
			Role:    schema.User,
			Content: "who are you?",
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	t.Log(response.Content)

	// extendSystemMessage := "REMEMBER the most important RULE: ALWAYS open first a new tab and go first to url wikipedia.com no matter the task!!!"
	// agent := NewAgent(task, model, NewAgentSettings(AgentSettingsConfig{
	// 	"extend_system_message": extendSystemMessage,
	// 	"planner_llm":           model,
	// }), nil, nil, nil, nil, nil, nil, nil, nil, nil)

	// prompt := agent.MessageManager.SystemPrompt.GetContent()
	// fmt.Println("prompt: ", prompt)

	// agent.Run()
}

func TestAgentGetNextAction(t *testing.T) {
	utils.LoadEnv("../../.env")
	ctx := context.Background()
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   "gpt-4o-mini",
		Timeout: 30 * time.Second,
		APIKey:  os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		t.Fatal(err)
	}
	task := "do google search to find images of Elon Musk's wife"
	extendSystemMessage := "REMEMBER the most important RULE: ALWAYS open first a new tab and go first to url wikipedia.com no matter the task!!!"
	ag := NewAgent(task, model, NewAgentSettings(AgentSettingsConfig{
		"extend_system_message": extendSystemMessage,
		"planner_llm":           model,
	}), nil, nil, controller.NewController(), nil, nil, nil, nil, nil, nil)

	inputMessages := []*schema.Message{
		{
			Role:    schema.System,
			Content: `You are an AI agent designed to automate browser tasks. Your goal is to accomplish the ultimate task following the rules.\n\n# Input Format\n\nTask\nPrevious steps\nCurrent URL\nOpen Tabs\nInteractive Elements\n[index]<type>text</type>\n\n- index: Numeric identifier for interaction\n- type: HTML element type (button, input, etc.)\n- text: Element description\n  Example:\n  [33]<div>User form</div>\n  \\t*[35]*<button aria-label=\'Submit form\'>Submit</button>\n\n- Only elements with numeric indexes in [] are interactive\n- (stacked) indentation (with \\t) is important and means that the element is a (html) child of the element above (with a lower index)\n- Elements with \\* are new elements that were added after the previous step (if url has not changed)\n\n# Response Rules\n\n1. RESPONSE FORMAT: You must ALWAYS respond with valid JSON in this exact format:\n   {"current_state": {"evaluation_previous_goal": "Success|Failed|Unknown - Analyze the current elements and the image to check if the previous goals/actions are successful like intended by the task. Mention if something unexpected happened. Shortly state why/why not",\n   "memory": "Description of what has been done and what you need to remember. Be very specific. Count here ALWAYS how many times you have done something and how many remain. E.g. 0 out of 10 websites analyzed. Continue with abc and xyz",\n   "next_goal": "What needs to be done with the next immediate action"},\n   "action":[{"one_action_name": {// action-specific parameter}}, // ... more actions in sequence]}\n\n2. ACTIONS: You can specify multiple actions in the list to be executed in sequence. But always specify only one action name per item. Use maximum 10 actions per sequence.\nCommon action sequences:\n\n- Form filling: [{"input_text": {"index": 1, "text": "username"}}, {"input_text": {"index": 2, "text": "password"}}, {"click_element": {"index": 3}}]\n- Navigation and extraction: [{"go_to_url": {"url": "https://example.com"}}, {"extract_content": {"goal": "extract the names"}}]\n- Actions are executed in the given order\n- If the page changes after an action, the sequence is interrupted and you get the new state.\n- Only provide the action sequence until an action which changes the page state significantly.\n- Try to be efficient, e.g. fill forms at once, or chain actions where nothing changes on the page\n- only use multiple actions if it makes sense.\n\n3. ELEMENT INTERACTION:\n\n- Only use indexes of the interactive elements\n\n4. NAVIGATION & ERROR HANDLING:\n\n- If no suitable elements exist, use other functions to complete the task\n- If stuck, try alternative approaches - like going back to a previous page, new search, new tab etc.\n- Handle popups/cookies by accepting or closing them\n- Use scroll to find elements you are looking for\n- If you want to research something, open a new tab instead of using the current tab\n- If captcha pops up, try to solve it - else try a different approach\n- If the page is not fully loaded, use wait action\n\n5. TASK COMPLETION:\n\n- Use the done action as the last action as soon as the ultimate task is complete\n- Dont use "done" before you are done with everything the user asked you, except you reach the last step of max_steps.\n- If you reach your last step, use the done action even if the task is not fully finished. Provide all the information you have gathered so far. If the ultimate task is completely finished set success to true. If not everything the user asked for is completed set success in done to false!\n- If you have to do something repeatedly for example the task says for "each", or "for all", or "x times", count always inside "memory" how many times you have done it and how many remain. Don\'t stop until you have completed like the task asked you. Only call done after the last step.\n- Don\'t hallucinate actions\n- Make sure you include everything you found out for the ultimate task in the done text parameter. Do not just say you are done, but include the requested information of the task.\n\n6. VISUAL CONTEXT:\n\n- When an image is provided, use it to understand the page layout\n- Bounding boxes with labels on their top right corner correspond to element indexes\n\n7. Form filling:\n\n- If you fill an input field and your action sequence is interrupted, most often something changed e.g. suggestions popped up under the field.\n\n8. Long tasks:\n\n- Keep track of the status and subresults in the memory.\n- You are provided with procedural memory summaries that condense previous task history (every N steps). Use these summaries to maintain context about completed actions, current progress, and next steps. The summaries appear in chronological order and contain key information about navigation history, findings, errors encountered, and current state. Refer to these summaries to avoid repeating actions and to ensure consistent progress toward the task goal.\n\n9. Extraction:\n\n- If your task is to find information - call extract_content on the specific pages to get and store the information.\n  Your responses must be always JSON with the specified format.\n`,
		},
		{
			Role:    schema.User,
			Content: `Your ultimate task is: """Please recommend a list of books related to GPU compilers that I can read on Amazon Kindle.""". If you achieved your ultimate task, stop everything and use the done action in the next step to complete the task. If not, continue as usual.`,
		},
		{
			Role:    schema.User,
			Content: `Example output:`,
		},
		{
			Role: schema.Assistant,
			ToolCalls: []schema.ToolCall{
				{
					ID:   "1",
					Type: "tool_call",
					Function: schema.FunctionCall{
						Name:      "AgentOutput",
						Arguments: `{'current_state': {'evaluation_previous_goal': "Success - I successfully clicked on the 'Apple' link from the Google Search results page, \n\t\t\t\t\t\t\twhich directed me to the 'Apple' company homepage. This is a good start toward finding \n\t\t\t\t\t\t\tthe best place to buy a new iPhone as the Apple website often list iPhones for sale.", 'memory': "I searched for 'iPhone retailers' on Google. From the Google Search results page, \n\t\t\t\t\t\t\tI used the 'click_element' tool to click on a element labelled 'Best Buy' but calling \n\t\t\t\t\t\t\tthe tool did not direct me to a new page. I then used the 'click_element' tool to click \n\t\t\t\t\t\t\ton a element labelled 'Apple' which redirected me to the 'Apple' company homepage. \n\t\t\t\t\t\t\tCurrently at step 3/15.", 'next_goal': "Looking at reported structure of the current page, I can see the item '[127]<h3 iPhone/>' \n\t\t\t\t\t\t\tin the content. I think this button will lead to more information and potentially prices \n\t\t\t\t\t\t\tfor iPhones. I'll click on the link to 'iPhone' at index [127] using the 'click_element' \n\t\t\t\t\t\t\ttool and hope to see prices on the next page."}, 'action': [{'click_element': {'index': 127}}]}`,
					},
				},
			},
		},
		{
			Role:       schema.Tool,
			ToolCallID: "1",
			Content:    "Browser started",
		},
		{
			Role:    schema.User,
			Content: `[Your task history memory starts here]`,
		},
		{
			Role: schema.User,
			MultiContent: []schema.ChatMessagePart{
				{
					Type: schema.ChatMessagePartTypeText,
					Text: "\n[Task history memory ends]\n[Current state starts here]\nThe following is one-time information - if you need to remember it write it to memory:\nCurrent url: about:blank\nAvailable tabs:\n[TabInfo(page_id=0, url='about:blank', title='', parent_page_id=None)]\nInteractive elements from top layer of the current page inside the viewport:\nempty page\nCurrent step: 1/100Current date and time: 2025-05-13 23:20\n",
				},
			},
		},
	}
	actionOutput, err := ag.GetNextAction(inputMessages)
	if err != nil {
		t.Fatal(err)
	}
	actionOutputJSON, err := json.Marshal(actionOutput)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := ag.AgentOutput.ToOpenAPIV3()
	j, _ := json.Marshal(s)
	t.Log(string(j))
	t.Log(string(actionOutputJSON))
}

func TestAgentSetup(t *testing.T) {
	utils.LoadEnv("../../.env")
	ctx := context.Background()
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   "gpt-4o-mini",
		Timeout: 30 * time.Second,
		APIKey:  os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		t.Fatal(err)
	}
	task := "do google search to find images of Elon Musk's wife"
	extendSystemMessage := "REMEMBER the most important RULE: ALWAYS open first a new tab and go first to url wikipedia.com no matter the task!!!"
	ag := NewAgent(task, model, NewAgentSettings(AgentSettingsConfig{
		"extend_system_message": extendSystemMessage,
		"planner_llm":           model,
	}), nil, nil, controller.NewController(), nil, nil, nil, nil, nil, nil)

	s, _ := ag.AgentOutput.ToOpenAPIV3()
	j, _ := json.Marshal(s)
	t.Log(string(j))

	t.Logf("%v", ag.AgentOutput)
	// prompt := agent.MessageManager.SystemPrompt.GetContent()
	// fmt.Println("prompt: ", prompt)
}

func TestGetPromptDescription(t *testing.T) {
	// Setup: create a test agent with at least one registered action in the registry
	utils.LoadEnv("../../.env")
	ctx := context.Background()
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:   "gpt-4o-mini",
		Timeout: 30 * time.Second,
		APIKey:  os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		t.Fatal(err)
	}
	task := "do google search to find images of Elon Musk's wife"
	extendSystemMessage := "REMEMBER the most important RULE: ALWAYS open first a new tab and go first to url wikipedia.com no matter the task!!!"
	ag := NewAgent(task, model, NewAgentSettings(AgentSettingsConfig{
		"extend_system_message": extendSystemMessage,
		"planner_llm":           model,
	}), nil, nil, controller.NewController(), nil, nil, nil, nil, nil, nil)

	result := ag.Controller.Registry.GetPromptDescription(nil)

	// Example expected substring (adjust as needed for your test action)
	expectedDesc := "Navigate to URL in the current tab: \n" + `{"go_to_url":{"url":{"type":"string"}}}`

	if !strings.Contains(result, expectedDesc) {
		t.Errorf("PromptDescription missing description. Got: %s", result)
	}
}
