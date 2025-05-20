package agent

import (
	"embed"
	"fmt"
	"strings"
	"time"

	"github.com/nerdface-ai/browser-use-go/internals/controller"
	"github.com/nerdface-ai/browser-use-go/pkg/browser"

	"github.com/cloudwego/eino/schema"
)

//go:embed system_prompt.md
var template embed.FS

type SystemPrompt struct {
	SystemMessage            *schema.Message
	DefaultActionDescription string
	MaxActionsPerStep        int
}

func NewSystemPrompt(
	actionDescription string,
	maxActionsPerStep int,
	overrideSystemMessage *string,
	extendSystemMessage *string,
) *SystemPrompt {
	sp := &SystemPrompt{
		DefaultActionDescription: actionDescription,
		MaxActionsPerStep:        maxActionsPerStep,
	}
	var prompt string
	if overrideSystemMessage != nil {
		prompt = *overrideSystemMessage
	} else {
		loaded := sp.loadPromptTemplate()
		prompt = strings.Replace(loaded, "{max_actions}", fmt.Sprintf("%d", sp.MaxActionsPerStep), -1)
	}

	if extendSystemMessage != nil {
		prompt += fmt.Sprintf("\n%s", *extendSystemMessage)
	}

	sp.SystemMessage = &schema.Message{
		Role:    schema.System,
		Content: prompt,
	}
	return sp
}

func (sp *SystemPrompt) loadPromptTemplate() string {
	// Load the prompt template from the markdown file
	data, err := template.ReadFile("system_prompt.md")
	if err != nil {
		panic(err)
	}
	return string(data)
}

type AgentMessagePrompt struct {
	State             *browser.BrowserState
	Result            []*controller.ActionResult
	IncludeAttributes []string
	StepInfo          *AgentStepInfo
}

func NewAgentMessagePrompt(
	state *browser.BrowserState,
	result []*controller.ActionResult,
	includeAttributes []string,
	stepInfo *AgentStepInfo,
) *AgentMessagePrompt {
	return &AgentMessagePrompt{
		State:             state,
		Result:            result,
		IncludeAttributes: includeAttributes,
		StepInfo:          stepInfo,
	}
}

func (amp *AgentMessagePrompt) GetUserMessage(useVision bool) *schema.Message {
	// get specific attribute clickable elements in DomTree as string
	elementText := amp.State.ElementTree.ClickableElementsToString(amp.IncludeAttributes)

	hasContentAbove := amp.State.PixelAbove > 0
	hasContentBelow := amp.State.PixelBelow > 0

	if elementText != "" {
		if hasContentAbove {
			elementText = fmt.Sprintf("... %d pixels above - scroll or extract content to see more ...\n%s", amp.State.PixelAbove, elementText)
		} else {
			elementText = fmt.Sprintf("[Start of page]\n%s", elementText)
		}
		// Update elementText by appending the new info to the existing value
		if hasContentBelow {
			elementText = fmt.Sprintf("%s\n... %d pixels below - scroll or extract content to see more ...", elementText, amp.State.PixelBelow)
		} else {
			elementText = fmt.Sprintf("%s\n[End of page]", elementText)
		}
	} else {
		elementText = "empty page"
	}

	var stepInfoDescription string
	if amp.StepInfo != nil {
		current := int(amp.StepInfo.StepNumber) + 1
		max := int(amp.StepInfo.MaxSteps)
		stepInfoDescription = fmt.Sprintf("Current step: %d/%d", current, max)
	} else {
		stepInfoDescription = ""
	}
	timeStr := time.Now().Format("2006-01-02 15:04")
	stepInfoDescription += fmt.Sprintf("Current date and time: %s", timeStr)

	stateDescription := fmt.Sprintf(`
[Task history memory ends]
[Current state starts here]
The following is one-time information - if you need to remember it write it to memory:
Current url: %s
Available tabs:
%s
Interactive elements from top layer of the current page inside the viewport:
%s
%s`,
		amp.State.Url,
		browser.TabsToString(amp.State.Tabs),
		elementText,
		stepInfoDescription,
	)

	if amp.Result != nil {
		for i, result := range amp.Result {
			if result.ExtractedContent != nil {
				stateDescription += fmt.Sprintf("\nAction result %d/%d: %s", i+1, len(amp.Result), *result.ExtractedContent)
			}
			if result.Error != nil {
				// only use last line of error
				errStr := *result.Error
				splitted := strings.Split(errStr, "\n")
				lastLine := splitted[len(splitted)-1]
				stateDescription += fmt.Sprintf("\nAction error %d/%d: ...%s", i+1, len(amp.Result), lastLine)
			}
		}
	}

	if amp.State.Screenshot != nil && useVision {
		// Format message for vision model
		return &schema.Message{
			Role: schema.User,
			MultiContent: []schema.ChatMessagePart{
				{
					Type: schema.ChatMessagePartTypeText,
					Text: stateDescription,
				},
				{
					Type: schema.ChatMessagePartTypeImageURL,
					ImageURL: &schema.ChatMessageImageURL{
						URL: "data:image/png;base64," + *amp.State.Screenshot,
					},
				},
			},
		}
	}

	return &schema.Message{
		Role:    schema.User,
		Content: stateDescription,
	}
}

func getPlannerPromptMessage(isPlannerReasoning bool) *schema.Message {
	plannerPromptText := `You are a planning agent that helps break down tasks into smaller steps and reason about the current state.
Your role is to:
1. Analyze the current state and history
2. Evaluate progress towards the ultimate goal
3. Identify potential challenges or roadblocks
4. Suggest the next high-level steps to take

Inside your messages, there will be AI messages from different agents with different formats.

Your output format should be always a JSON object with the following fields:
{
    "state_analysis": "Brief analysis of the current state and what has been done so far",
    "progress_evaluation": "Evaluation of progress towards the ultimate goal (as percentage and description)",
    "challenges": "List any potential challenges or roadblocks",
    "next_steps": "List 2-3 concrete next steps to take",
    "reasoning": "Explain your reasoning for the suggested next steps"
}

Ignore the other AI messages output structures.

Keep your responses concise and focused on actionable insights.`

	if isPlannerReasoning {
		return &schema.Message{
			Role:    schema.User,
			Content: plannerPromptText,
		}
	}

	return &schema.Message{
		Role:    schema.System,
		Content: plannerPromptText,
	}
}
