## License

This project is licensed under the MIT License.

**Notice:**  
This project uses the [eino](https://github.com/cloudwego/eino) library, which is licensed under the Apache License 2.0.  
See the [NOTICE](./NOTICE) file for details.
# browser-use-go

**⚠️ This project is currently under active development. Please note that the API is subject to change.**

A Go implementation of the [browser-use](https://github.com/browser-use/browser-use) library, built using [playwright-go](https://github.com/playwright-community/playwright-go) and [eino](https://github.com/cloudwego/eino) for browser automation.

Following the [commit](https://github.com/browser-use/browser-use/tree/e280cab621afc4a1c900d8a905f6503602b6a6d9) and [deepwiki](https://deepwiki.com/browser-use/browser-use)

## Overview

This library provides a Go interface for browser automation, following the patterns and functionality of the original browser-use python library. It leverages the Playwright Go bindings for reliable and efficient browser control.

## Installation

### Prerequisites

Before using this library, you need to install the required browsers and dependencies:

```bash
go run github.com/playwright-community/playwright-go/cmd/playwright@v0.5101.0 install --with-deps
# Or
go install github.com/playwright-community/playwright-go/cmd/playwright@v0.5101.0
playwright install --with-deps
```

### Package Installation

```bash
go get github.com/nerdface-ai/browser-use-go
```

## Usage

copy .env.example to .env and fill in the values

```go
package main

import (
	"context"
	"os"

	"github.com/charmbracelet/log"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/nerdface-ai/browser-use-go/pkg/agent"
	"github.com/nerdface-ai/browser-use-go/pkg/dotenv"
)

func main() {
	dotenv.LoadEnv(".env")

	log.Debug(os.Getenv("OPENAI_API_KEY"))

	ctx := context.Background()
	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		Model:  "gpt-4o-mini",
		APIKey: os.Getenv("OPENAI_API_KEY"),
	})
	if err != nil {
		log.Fatal(err)
	}

	task := "do google search and find who is Elon Musk's wife"
	ag := agent.NewAgent(task, model)
	historyResult, err := ag.Run()

	if err != nil {
		log.Fatal(err.Error())
	}
	log.Infof("agent output: %s", *historyResult.LastResult().ExtractedContent)
}
```

## Features

- Browser automation using Playwright through natural language.

## Contributing

We welcome and appreciate contributions from the community! Whether it's bug reports, feature requests, or code contributions, all are welcome. Here's how you can contribute:

1. **Report bugs** by opening an issue
2. **Suggest features** through the issue tracker
3. **Submit pull requests** with bug fixes or new features
4. **Improve documentation**

Please make sure to read our [Contributing Guidelines](CONTRIBUTING.md) before making a pull request.

## License

MIT License - see LICENSE file for details

