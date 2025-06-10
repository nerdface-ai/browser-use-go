**📢 Notice:** This project currently aligns with the `0.1.x` version series of the `browser-use` library. Work to support the `0.2.x` version series is scheduled to begin in mid-June.

# browser-use-go

**⚠️ This project is currently under active development. Please note that the API is subject to change.**

`browser-use-go` is a Go library for browser automation. Built on top of the Playwright Go library, it enables interaction with web browsers and content extraction from web pages using natural language. The primary purpose of this library is to enhance the usability of `browser-use` as a library, and TUI (Text-based User Interface) support is not a priority at this time.

## Key Features

*   Natural language-based browser automation via Playwright.
*   Facilitates easy website access and usability for AI agents.

## Installation

### Prerequisites

Before using this library, you need to install the required browsers and dependencies:

```bash
go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5101.0 install --with-deps
# Or
# go install github.com/playwright-community/playwright-go/cmd/playwright@v0.5101.0
# playwright install --with-deps
```

### Package Installation

```bash
go get github.com/nerdface-ai/browser-use-go
```

## Usage

For environment variable setup, copy `.env.example` to `.env` and fill in the necessary values.

```go
package main

import (
	"context"
	"os"

	"github.com/charmbracelet/log"
	"github.com/cloudwego/eino-ext/components/model/openai" // Example: When using OpenAI model
	"github.com/nerdface-ai/browser-use-go/pkg/agent"
	"github.com/nerdface-ai/browser-use-go/pkg/dotenv"
)

func main() {
	// Load environment variables from .env file
	dotenv.LoadEnv(".env")

	apiKey := os.Getenv("OPENAI_API_KEY") // Example environment variable
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set.")
	}

	log.Debug("OPENAI_API_KEY loaded.")

	ctx := context.Background()
	// Can be replaced with the LLM model you intend to use
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:  "gpt-4o-mini", // The model name you intend to use
		APIKey: apiKey,
	})
	if err != nil {
		log.Fatal("Failed to create chat model:", "err", err)
	}

	task := "Do a Google search to find out who Elon Musk's wife is." // Task to perform
	ag := agent.NewAgent(task, model)
	historyResult, err := ag.Run()

	if err != nil {
		log.Fatal("Agent run failed:", "err", err)
	}

	if historyResult != nil && historyResult.LastResult() != nil && historyResult.LastResult().ExtractedContent != nil {
		log.Infof("Agent output: %s", *historyResult.LastResult().ExtractedContent)
	} else {
		log.Info("Agent did not produce an extractable result.")
	}
}
```

## Contributing

We welcome and appreciate contributions from the community! Whether it's bug reports, feature requests, or code contributions, all are welcome. Here's how you can contribute:

1.  **Report bugs** by opening an issue.
2.  **Suggest features** through the issue tracker.
3.  **Submit pull requests** with bug fixes or new features.
4.  **Improve documentation**.

Please make sure to read our [Contributing Guidelines](CONTRIBUTING.md) before making a pull request.

## License

This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.

**Notice:**
This project uses the [eino](https://github.com/cloudwego/eino) library, which is licensed under the Apache License 2.0.
See the [NOTICE](./NOTICE) file for details.

