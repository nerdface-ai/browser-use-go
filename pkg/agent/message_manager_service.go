package agent

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nerdface-ai/browser-use-go/internals/controller"
	"github.com/nerdface-ai/browser-use-go/internals/utils"
	"github.com/nerdface-ai/browser-use-go/pkg/browser"

	"github.com/charmbracelet/log"
	"github.com/cloudwego/eino/schema"
	"github.com/playwright-community/playwright-go"
)

type MessageManagerSettings struct {
	MaxInputTokens              int               `json:"max_input_tokens"`
	EstimatedCharactersPerToken int               `json:"estimated_characters_per_token"`
	ImageTokens                 int               `json:"image_tokens"`
	IncludeAttributes           []string          `json:"include_attributes"`
	MessageContext              *string           `json:"message_context,omitempty"`
	SensitiveData               map[string]string `json:"sensitive_data"`
	AvailableFilePaths          []string          `json:"available_file_paths"`
}

type MessageManagerConfig map[string]interface{}

func NewMessageManagerSettings(config MessageManagerConfig) *MessageManagerSettings {
	return &MessageManagerSettings{
		MaxInputTokens:              utils.GetDefaultValue[int](config, "max_input_tokens", 128000),
		EstimatedCharactersPerToken: utils.GetDefaultValue[int](config, "estimated_characters_per_token", 3),
		ImageTokens:                 utils.GetDefaultValue[int](config, "image_tokens", 800),
		IncludeAttributes:           utils.GetDefaultValue[[]string](config, "include_attributes", []string{}),
		MessageContext:              utils.GetDefaultValue[*string](config, "message_context", nil),
		SensitiveData:               utils.GetDefaultValue[map[string]string](config, "sensitive_data", nil),
		AvailableFilePaths:          utils.GetDefaultValue[[]string](config, "available_file_paths", nil),
	}
}

type MessageManager struct {
	Task         string
	SystemPrompt *schema.Message
	Settings     *MessageManagerSettings
	State        *MessageManagerState
}

// AgentBrain
type CurrentState struct {
	EvaluationPreviousGoal string `json:"evaluation_previous_goal"`
	Memory                 string `json:"memory"`
	NextGoal               string `json:"next_goal"`
}

type AIMessageArguments struct {
	CurrentState CurrentState             `json:"current_state"`
	Actions      []map[string]interface{} `json:"actions"`
}

func NewMessageManager(
	task string,
	systemPrompt *schema.Message,
	settings *MessageManagerSettings,
	state *MessageManagerState,
) *MessageManager {
	if settings == nil {
		defaultSettings := NewMessageManagerSettings(MessageManagerConfig{})
		settings = defaultSettings
	}
	if state == nil {
		state = NewMessageManagerState()
	}

	manager := &MessageManager{
		Task:         task,
		SystemPrompt: systemPrompt,
		Settings:     settings,
		State:        state,
	}

	// Only initialize messages if state is empty
	if len(state.History.Messages) == 0 {
		manager.initMessages()
	}
	return manager
}

func (m *MessageManager) initMessages() {
	initStr := "init"
	// Initialize message history with system message, context, task, and other initial messages
	m.AddMessageWithTokens(m.SystemPrompt, nil, &initStr)

	if m.Settings.MessageContext != nil {
		contextMessage := &schema.Message{
			Role:    schema.User,
			Content: "Context for the task" + *m.Settings.MessageContext,
		}
		m.AddMessageWithTokens(contextMessage, nil, &initStr)
	}

	taskMessage := &schema.Message{
		Role: schema.User,
		Content: fmt.Sprintf(
			`Your ultimate task is: "%s". 
			If you achieved your ultimate task, stop everything and use the done action in the next step to complete the task. 
			If not, continue as usual.`,
			m.Task,
		),
	}
	m.AddMessageWithTokens(taskMessage, nil, &initStr)

	if m.Settings.SensitiveData != nil {
		data := m.Settings.SensitiveData
		keys := make([]string, 0, len(data))
		for k := range data {
			keys = append(keys, k)
		}
		info := fmt.Sprintf("Here are placeholders for sensitive data: %s", strings.Join(keys, ", "))
		info += "To use them, write <secret>the placeholder name</secret>"
		infoMessage := &schema.Message{
			Role:    schema.User,
			Content: info,
		}
		m.AddMessageWithTokens(infoMessage, nil, &initStr)
	}

	placeHolderMessage := &schema.Message{
		Role:    schema.User,
		Content: "Example output:",
	}
	m.AddMessageWithTokens(placeHolderMessage, nil, &initStr)

	args := AIMessageArguments{
		CurrentState: CurrentState{
			EvaluationPreviousGoal: `Success - I successfully clicked on the 'Apple' link from the Google Search results page, 
				which directed me to the 'Apple' company homepage. This is a good start toward finding 
				the best place to buy a new iPhone as the Apple website often list iPhones for sale.`,
			Memory: `I searched for 'iPhone retailers' on Google. From the Google Search results page, 
				I used the 'click_element_by_index' tool to click on a element labelled 'Best Buy' but calling 
				the tool did not direct me to a new page. I then used the 'click_element_by_index' tool to click 
				on a element labelled 'Apple' which redirected me to the 'Apple' company homepage. 
				Currently at step 3/15.`,
			NextGoal: `Looking at reported structure of the current page, I can see the item '[127]<h3 iPhone/>' 
				in the content. I think this button will lead to more information and potentially prices 
				for iPhones. I'll click on the link to 'iPhone' at index [127] using the 'click_element_by_index' 
				tool and hope to see prices on the next page.`,
		},
		Actions: []map[string]interface{}{
			{
				"click_element_by_index": map[string]interface{}{
					"index": 127,
				},
			},
		},
	}
	argsBytes, err := json.Marshal(args)
	if err != nil {
		panic(err)
	}

	exampleToolCall := &schema.Message{
		Role:    schema.Assistant,
		Content: "",
		ToolCalls: []schema.ToolCall{
			{
				ID:   strconv.Itoa(m.State.ToolId),
				Type: "tool_call",
				Function: schema.FunctionCall{
					Name:      "AgentOutput",
					Arguments: string(argsBytes),
				},
			},
		},
	}
	m.AddMessageWithTokens(exampleToolCall, nil, &initStr)
	m.addToolMessage("Browser started", &initStr)

	// Clarify that below is about task history
	placeHolderMessage = &schema.Message{
		Role:    schema.User,
		Content: "[Your task history memory starts here]",
	}
	m.AddMessageWithTokens(placeHolderMessage, nil, nil)

	if m.Settings.AvailableFilePaths != nil {
		filePathsMsg := &schema.Message{
			Role:    schema.User,
			Content: fmt.Sprintf("Here are file paths you can use: %s", strings.Join(m.Settings.AvailableFilePaths, ", ")),
		}
		m.AddMessageWithTokens(filePathsMsg, nil, &initStr)
	}
}

func (m *MessageManager) AddNewTask(newTask string) {
	content := fmt.Sprintf("Your new ultimate task is: \"%s\". Take the previous context into account and finish your new ultimate task. ", newTask)
	msg := &schema.Message{
		Role:    schema.User,
		Content: content,
	}
	m.AddMessageWithTokens(msg, nil, nil)
	m.Task = newTask
}

func (m *MessageManager) AddStateMessage(
	state *browser.BrowserState,
	result []*controller.ActionResult,
	stepInfo *AgentStepInfo,
	useVision bool,
) {
	// Add browser state as human message
	// if keep in memory, add to directly to history and add state without result
	for _, r := range result {
		if r.IncludeInMemory {
			if r.ExtractedContent != nil {
				msg := &schema.Message{
					Role:    schema.User,
					Content: "Action result: " + *r.ExtractedContent,
				}
				m.AddMessageWithTokens(msg, nil, nil)
			}
			if r.Error != nil {
				// if endswith \n, remove it
				errStr := *r.Error
				errStr = strings.TrimSuffix(errStr, "\n")
				r.Error = playwright.String(errStr)
				// get only last line of error
				splitted := strings.Split(errStr, "\n")
				lastLine := splitted[len(splitted)-1]
				msg := &schema.Message{
					Role:    schema.User,
					Content: "Action error: " + lastLine,
				}
				m.AddMessageWithTokens(msg, nil, nil)
			}
			// if result in history, we dont want to add it again (add to memory only first one in the result)
			result = nil
		}
	}

	// otherwise add state message and result to next message (which will not stay in memory)
	stateMessage := NewAgentMessagePrompt(state, result, m.Settings.IncludeAttributes, stepInfo).
		GetUserMessage(useVision)
	m.AddMessageWithTokens(stateMessage, nil, nil)
}

func (m *MessageManager) AddModelOutput(output *AgentOutput) {
	// Add model output as AI message
	toolCalls := []schema.ToolCall{
		{
			ID:   strconv.Itoa(m.State.ToolId),
			Type: "tool_call",
			Function: schema.FunctionCall{
				Name:      "AgentOutput",
				Arguments: output.ToString(),
			},
		},
	}

	msg := &schema.Message{
		Role:      schema.Assistant,
		Content:   "",
		ToolCalls: toolCalls,
	}
	m.AddMessageWithTokens(msg, nil, nil)
	// empty tool response
	m.addToolMessage("tool executed", nil)
}

func (m *MessageManager) AddPlan(plan *string, position *int) error {
	if plan != nil && *plan != "" {
		msg := &schema.Message{
			Role:    schema.Assistant,
			Content: *plan,
		}
		m.AddMessageWithTokens(msg, position, nil)
	}
	return nil
}

func (m *MessageManager) GetMessages() []*schema.Message {
	// Get current message list, potentially trimmed to max tokens

	msg := make([]*schema.Message, len(m.State.History.Messages))
	// debug which messages are in history with token count # log
	totalInputTokens := 0
	for i, mm := range m.State.History.Messages {
		msg[i] = mm.Message
		totalInputTokens += mm.Metadata.Tokens
		log.Debugf("%s - Token count: %d", mm.Message.Role, mm.Metadata.Tokens)
	}
	log.Debugf("Total input tokens: %d", totalInputTokens)

	return msg
}

func (m *MessageManager) AddMessageWithTokens(
	message *schema.Message,
	position *int,
	messageType *string,
) {
	/*
		Add message with token count metadata
		position: None for last, -1 for second last, etc.
	*/

	// TODO(HIGH): filter out sensitive data from the message
	// if m.Settings.SensitiveData != nil {
	// 	message = filterSensitiveData(message)
	// }

	tokenCount := m.countTokens(message)
	metadata := &MessageMetadata{
		Tokens:      tokenCount,
		MessageType: messageType,
	}
	m.State.History.AddMessage(message, metadata, position)
}

func (m *MessageManager) countTokens(message *schema.Message) int {
	// Count tokens in a message using the model's tokenizer
	tokens := 0
	var msg string
	if len(message.MultiContent) > 0 {
		for _, part := range message.MultiContent {
			if part.Type == schema.ChatMessagePartTypeImageURL {
				tokens += m.Settings.ImageTokens
			} else if part.Type == schema.ChatMessagePartTypeText {
				tokens += m.countTextTokens(part.Text)
			}
		}
	} else {
		msg = message.Content
		if message.ToolCalls != nil {
			argsBytes, err := json.Marshal(message.ToolCalls)
			if err != nil {
				panic(err)
			}
			msg += string(argsBytes)
		}
		tokens += m.countTextTokens(msg)
	}
	return tokens
}

func (m *MessageManager) countTextTokens(text string) int {
	return int(math.Round(float64(len(text)) / float64(m.Settings.EstimatedCharactersPerToken)))
}

func (m *MessageManager) CutMessages() error {
	// Get current message list, potentially trimmed to max tokens
	diff := m.State.History.CurrentTokens - m.Settings.MaxInputTokens
	if diff <= 0 {
		return nil
	}

	msg := m.State.History.Messages[len(m.State.History.Messages)-1]

	// if list with image remove image
	if len(msg.Message.MultiContent) > 0 {
		text := ""
		for _, item := range msg.Message.MultiContent {
			if item.Type == schema.ChatMessagePartTypeImageURL {
				diff -= m.Settings.ImageTokens
				msg.Metadata.Tokens -= m.Settings.ImageTokens
				m.State.History.CurrentTokens -= m.Settings.ImageTokens
				log.Debugf("Removed image with %d tokens - total tokens now: %d/%d", m.Settings.ImageTokens, m.State.History.CurrentTokens, m.Settings.MaxInputTokens)
			} else if item.Type == schema.ChatMessagePartTypeText {
				text += item.Text
			}
		}
		// leave only text content
		msg.Message.Content = text
		msg.Message.MultiContent = nil
		m.State.History.Messages[len(m.State.History.Messages)-1] = msg
	}

	if diff <= 0 {
		return nil
	}

	// if still over, remove text from state message proportionally to the number of tokens needed with buffer
	// Calculate the proportion of content to remove
	proportionToRemove := float64(diff) / float64(msg.Metadata.Tokens)
	if proportionToRemove > 0.99 {
		return fmt.Errorf(
			"max token limit reached - history is too long - reduce the system prompt or task. "+
				"proportion_to_remove: %f.2f",
			proportionToRemove)
	}
	log.Debug("Removing %f.2f of the last message (%f.2f / %f.2f tokens)",
		proportionToRemove*100,
		proportionToRemove*float64(msg.Metadata.Tokens),
		float64(msg.Metadata.Tokens),
	)

	content := msg.Message.Content
	charactersToRemove := len(content) * int(proportionToRemove)
	content = content[:len(content)-charactersToRemove]

	// remove tokens and old long message
	m.State.History.RemoveLastStateMessage()

	// add new message with updated content
	newMsg := &schema.Message{
		Role:    schema.User,
		Content: content,
	}
	m.AddMessageWithTokens(newMsg, nil, nil)

	lastMsg := m.State.History.Messages[len(m.State.History.Messages)-1]

	log.Debug("Added message with %d tokens - total tokens now: %d / %d - total messages: %d",
		lastMsg.Metadata.Tokens,
		m.State.History.CurrentTokens,
		m.Settings.MaxInputTokens,
		len(m.State.History.Messages),
	)
	return nil
}

func (m *MessageManager) RemoveLastStateMessage() error {
	// remove last state nessage from history
	m.State.History.RemoveLastStateMessage()
	return nil
}

func (m *MessageManager) addToolMessage(content string, messageType *string) {
	// Add tool message to history
	msg := &schema.Message{
		Role:       schema.Tool,
		Content:    content,
		ToolCallID: strconv.Itoa(m.State.ToolId),
	}
	m.State.ToolId++
	m.AddMessageWithTokens(msg, nil, messageType)
}

func (m *MessageManager) SaveConversation(
	inputMessages []*schema.Message,
	modelOutput *AgentOutput,
	target string,
) error {
	// Save conversation history to file

	// create folders if not exists
	dirname := filepath.Dir(target)
	if _, err := os.Stat(dirname); os.IsNotExist(err) {
		os.MkdirAll(dirname, 0755)
	}

	f, err := os.Create(target)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := writeMessagesToFile(f, inputMessages); err != nil {
		return err
	}
	if err := writeAgentOutputToFile(f, modelOutput); err != nil {
		return err
	}

	return nil
}

func writeMessagesToFile(f *os.File, messages []*schema.Message) error {
	for _, msg := range messages {
		fmt.Fprintf(f, " %s \n", msg.Role)

		var js map[string]interface{}
		if err := json.Unmarshal([]byte(msg.Content), &js); err == nil {
			pretty, _ := json.MarshalIndent(js, "", "  ")
			if _, err := f.WriteString(string(pretty) + "\n"); err != nil {
				return err
			}
		} else {
			if _, err := f.WriteString(msg.Content + "\n"); err != nil {
				return err
			}
		}
		f.WriteString("\n")
	}
	return nil
}

func writeAgentOutputToFile(f *os.File, modelOutput *AgentOutput) error {
	if modelOutput == nil {
		return nil
	}
	fmt.Fprintf(f, " AgentOutput \n")

	js, err := json.MarshalIndent(modelOutput, "", "  ")
	if err == nil {
		if _, err := f.WriteString(string(js) + "\n"); err != nil {
			return err
		}
	} else {
		if _, err := f.WriteString(fmt.Sprintf("%+v\n", modelOutput)); err != nil {
			return err
		}
	}
	f.WriteString("\n")
	return nil
}
