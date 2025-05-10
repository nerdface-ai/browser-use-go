package agent

import (
	"encoding/json"
	"fmt"
	"math"
	"nerdface-ai/browser-use-go/browser-use/browser"
	"nerdface-ai/browser-use-go/browser-use/controller"
	"nerdface-ai/browser-use-go/browser-use/utils"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/moznion/go-optional"
	"github.com/tmc/langchaingo/llms"
)

type MessageManagerSettings struct {
	MaxInputTokens              int                     `json:"max_input_tokens"`
	EstimatedCharactersPerToken int                     `json:"estimated_characters_per_token"`
	ImageTokens                 int                     `json:"image_tokens"`
	IncludeAttributes           []string                `json:"include_attributes"`
	MessageContext              optional.Option[string] `json:"message_context"`
	SensitiveData               map[string]string       `json:"sensitive_data"`
	AvailableFilePaths          []string                `json:"available_file_paths"`
}

type MessageManagerConfig map[string]interface{}

func NewMessageManagerSettings(config MessageManagerConfig) *MessageManagerSettings {
	return &MessageManagerSettings{
		MaxInputTokens:              utils.GetDefaultValue[int](config, "max_input_tokens", 128000),
		EstimatedCharactersPerToken: utils.GetDefaultValue[int](config, "estimated_characters_per_token", 3),
		ImageTokens:                 utils.GetDefaultValue[int](config, "image_tokens", 800),
		IncludeAttributes:           utils.GetDefaultValue[[]string](config, "include_attributes", []string{}),
		MessageContext:              utils.GetDefaultValue[optional.Option[string]](config, "message_context", nil),
		SensitiveData:               utils.GetDefaultValue[map[string]string](config, "sensitive_data", nil),
		AvailableFilePaths:          utils.GetDefaultValue[[]string](config, "available_file_paths", nil),
	}
}

type MessageManager struct {
	Task         string
	SystemPrompt llms.SystemChatMessage
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
	Action       []map[string]interface{} `json:"action"`
}

func NewMessageManager(
	task string,
	systemPrompt llms.SystemChatMessage,
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
	// Initialize message history with system message, context, task, and other initial messages
	m.addMessageWithTokens(m.SystemPrompt, nil, optional.Some("init"))

	if m.Settings.MessageContext != nil {
		contextMessage := llms.HumanChatMessage{
			Content: "Context for the task" + m.Settings.MessageContext.Unwrap(),
		}
		m.addMessageWithTokens(contextMessage, nil, optional.Some("init"))
	}

	taskMessage := llms.HumanChatMessage{
		Content: fmt.Sprintf(
			`Your ultimate task is: "%s". 
			If you achieved your ultimate task, stop everything and use the done action in the next step to complete the task. 
			If not, continue as usual.`,
			m.Task,
		),
	}
	m.addMessageWithTokens(taskMessage, nil, optional.Some("init"))

	if m.Settings.SensitiveData != nil {
		data := m.Settings.SensitiveData
		keys := make([]string, 0, len(data))
		for k := range data {
			keys = append(keys, k)
		}
		info := fmt.Sprintf("Here are placeholders for sensitive data: %s", strings.Join(keys, ", "))
		info += "To use them, write <secret>the placeholder name</secret>"
		infoMessage := llms.HumanChatMessage{
			Content: info,
		}
		m.addMessageWithTokens(infoMessage, nil, optional.Some("init"))
	}

	placeHolderMessage := llms.HumanChatMessage{
		Content: "Example output:",
	}
	m.addMessageWithTokens(placeHolderMessage, nil, optional.Some("init"))

	args := AIMessageArguments{
		CurrentState: CurrentState{
			EvaluationPreviousGoal: `Success - I successfully clicked on the 'Apple' link from the Google Search results page, 
				which directed me to the 'Apple' company homepage. This is a good start toward finding 
				the best place to buy a new iPhone as the Apple website often list iPhones for sale.`,
			Memory: `I searched for 'iPhone retailers' on Google. From the Google Search results page, 
				I used the 'click_element' tool to click on a element labelled 'Best Buy' but calling 
				the tool did not direct me to a new page. I then used the 'click_element' tool to click 
				on a element labelled 'Apple' which redirected me to the 'Apple' company homepage. 
				Currently at step 3/15.`,
			NextGoal: `Looking at reported structure of the current page, I can see the item '[127]<h3 iPhone/>' 
				in the content. I think this button will lead to more information and potentially prices 
				for iPhones. I'll click on the link to 'iPhone' at index [127] using the 'click_element' 
				tool and hope to see prices on the next page.`,
		},
		Action: []map[string]interface{}{
			{
				"click_element": map[string]interface{}{
					"index": 127,
				},
			},
		},
	}
	argsBytes, err := json.Marshal(args)
	if err != nil {
		panic(err)
	}

	exampleToolCall := llms.AIChatMessage{
		Content: "",
		ToolCalls: []llms.ToolCall{
			{
				ID:   strconv.Itoa(m.State.ToolId),
				Type: "tool_call",
				FunctionCall: &llms.FunctionCall{
					Name:      "AgentOutput",
					Arguments: string(argsBytes),
				},
			},
		},
	}
	m.addMessageWithTokens(exampleToolCall, nil, optional.Some("init"))
	m.addToolMessage("Browser started", optional.Some("init"))

	// Clarify that below is about task history
	placeHolderMessage = llms.HumanChatMessage{
		Content: "[Your task history memory starts here]",
	}
	m.addMessageWithTokens(placeHolderMessage, nil, nil)

	if m.Settings.AvailableFilePaths != nil {
		filePathsMsg := llms.HumanChatMessage{
			Content: fmt.Sprintf("Here are file paths you can use: %s", strings.Join(m.Settings.AvailableFilePaths, ", ")),
		}
		m.addMessageWithTokens(filePathsMsg, nil, optional.Some("init"))
	}
}

func (m *MessageManager) addMessageWithTokens(
	message llms.ChatMessage,
	position optional.Option[int],
	messageType optional.Option[string],
) {
	/*
		Add message with token count metadata
		position: None for last, -1 for second last, etc.
	*/

	// TODO: filter out sensitive data from the message
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

func (m *MessageManager) countTokens(message llms.ChatMessage) int {
	// Count tokens in a message using the model's tokenizer
	tokens := 0
	msg := message.GetContent()

	// TODO:
	// if hasattr(message, 'tool_calls'):
	// 	msg += str(message.tool_calls)  # type: ignore

	tokens += int(math.Round(float64(len(msg)) / float64(m.Settings.EstimatedCharactersPerToken)))
	return tokens
}

func (m *MessageManager) addToolMessage(content string, messageType optional.Option[string]) {
	// Add tool message to history
	msg := llms.ToolChatMessage{
		Content: content,
		ID:      strconv.Itoa(m.State.ToolId),
	}
	m.State.ToolId++
	m.addMessageWithTokens(msg, nil, messageType)
}

func (m *MessageManager) GetMessages() []llms.ChatMessage {
	// Get current message list, potentially trimmed to max tokens

	msg := make([]llms.ChatMessage, len(m.State.History.Messages))
	// debug which messages are in history with token count # log
	totalInputTokens := 0
	for i, mm := range m.State.History.Messages {
		msg[i] = mm.Message
		totalInputTokens += mm.Metadata.Tokens
		log.Debug(fmt.Sprintf("%T - Token count: %d", mm.Message.GetType(), mm.Metadata.Tokens))
	}
	log.Debug(fmt.Sprintf("Total input tokens: %d", totalInputTokens))

	return msg
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
				msg := llms.HumanChatMessage{
					Content: "Action result: " + r.ExtractedContent.Unwrap(),
				}
				m.addMessageWithTokens(msg, nil, nil)
			}
			if r.Error != nil {
				// if endswith \n, remove it
				errStr := r.Error.Unwrap()
				errStr = strings.TrimSuffix(errStr, "\n")
				r.Error = optional.Some(errStr)
				// get only last line of error
				splitted := strings.Split(errStr, "\n")
				lastLine := splitted[len(splitted)-1]
				msg := llms.HumanChatMessage{
					Content: "Action error: " + lastLine,
				}
				m.addMessageWithTokens(msg, nil, nil)
			}
			// if result in history, we dont want to add it again (add to memory only first one in the result)
			result = nil
		}
	}

	// otherwise add state message and result to next message (which will not stay in memory)
	stateMessage := NewAgentMessagePrompt(state, result, m.Settings.IncludeAttributes, stepInfo).
		GetUserMessage(useVision)
	m.addMessageWithTokens(stateMessage, nil, nil)
}

func (m *MessageManager) SaveConversation(
	inputMessages []llms.ChatMessage,
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

func writeMessagesToFile(f *os.File, messages []llms.ChatMessage) error {
	for _, msg := range messages {
		fmt.Fprintf(f, " %s \n", msg.GetType())

		var js map[string]interface{}
		if err := json.Unmarshal([]byte(msg.GetContent()), &js); err == nil {
			pretty, _ := json.MarshalIndent(js, "", "  ")
			if _, err := f.WriteString(string(pretty) + "\n"); err != nil {
				return err
			}
		} else {
			if _, err := f.WriteString(msg.GetContent() + "\n"); err != nil {
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

func (m *MessageManager) RemoveLastStateMessage() error {
	return nil
}

func (m *MessageManager) AddModelOutput(modelOutput *AgentOutput) error {
	return nil
}
