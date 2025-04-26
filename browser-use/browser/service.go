package browser

import (
	"github.com/google/uuid"
	"github.com/playwright-community/playwright-go"
)

type BrowserConfig = map[string]interface{}

type Browser struct {
	Config  BrowserConfig
	Browser playwright.Browser
}

func (self *Browser) NewContext() *BrowserContext {
	context, err := self.Browser.NewContext()
	if err != nil {
		panic(err)
	}

	session := NewSession(context, nil)

	return &BrowserContext{
		ContextId: uuid.New().String(),
		Config:    self.Config,
		Browser:   self,
		Session:   session,
		State:     &BrowserContextState{},
	}
}

func (self *Browser) GetPlaywrightBrowser() playwright.Browser {
	return self.Browser
}

func (self *Browser) Close(options ...playwright.BrowserCloseOptions) error {
	return self.Browser.Close(options...)
}
