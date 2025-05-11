package controller

import (
	"github.com/playwright-community/playwright-go"
)

// Action Input Models
type SearchGoogleAction struct {
	Query string `json:"query"`
}

type GoToUrlAction struct {
	Url string `json:"url"`
}

type ClickElementAction struct {
	Index int     `json:"index"`
	Xpath *string `json:"xpath,omitempty" jsonschema:"anyof_type=string;null,default=null"`
}

type InputTextAction struct {
	Index int     `json:"index"`
	Text  string  `json:"text"`
	Xpath *string `json:"xpath,omitempty" jsonschema:"anyof_type=string;null,default=null"`
}

type DoneAction struct {
	Text    string `json:"text"`
	Success bool   `json:"success"`
}

type WaitAction struct {
	Seconds int `json:"seconds"`
}

type GoBackAction struct {
}

type SavePdfAction struct {
}

type ExtractContentAction struct {
	Goal                string `json:"goal"`
	ShouldStripLinkUrls bool   `json:"should_strip_link_urls"`
}

type ScrollToTextAction struct {
	Text string `json:"text"`
}

type GetDropdownOptionsAction struct {
	Index int `json:"index"`
}

type SelectDropdownOptionAction struct {
	Index int    `json:"index"`
	Text  string `json:"text"`
}

type SwitchTabAction struct {
	PageId int `json:"page_id"`
}

type OpenTabAction struct {
	Url string `json:"url"`
}

type CloseTabAction struct {
	PageId int `json:"page_id"`
}

type ScrollDownAction struct {
	Amount *int `json:"amount,omitempty" jsonschema:"anyof_type=integer;null,default=null"`
}

type ScrollUpAction struct {
	Amount *int `json:"amount,omitempty" jsonschema:"anyof_type=integer;null,default=null"`
}

type SendKeysAction struct {
	Keys string `json:"keys"`
}

type GroupTabsAction struct {
	TabIds []int   `json:"tab_ids"`
	Title  string  `json:"title"`
	Color  *string `json:"color,omitempty" jsonschema:"anyof_type=string;null,default=null"`
}

type UngroupTabsAction struct {
	TabIds []int `json:"tab_ids"`
}

type ExtractPageContentAction struct {
	Value string `json:"value"`
}

type NoParamsAction struct {
	// Accepts absolutely anything in the incoming data
	// and discards it, so the final parsed model is empty.
}

func (NoParamsAction) IgnoreAllInputs(values map[string]interface{}) *NoParamsAction {
	// No matter what the user sends, discard it and return empty.
	return &NoParamsAction{}
}

type Position struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type DragDropAction struct {
	// Element-based approach
	ElementSource       *string   `json:"element_source,omitempty" jsonschema:"anyof_type=string;null,default=null"`
	ElementTarget       *string   `json:"element_target,omitempty" jsonschema:"anyof_type=string;null,default=null"`
	ElementSourceOffset *Position `json:"element_source_offset,omitempty" jsonschema:"anyof_type=object;null,default=null"`
	ElementTargetOffset *Position `json:"element_target_offset,omitempty" jsonschema:"anyof_type=object;null,default=null"`

	// Coordinate-based approach (used if selectors not provided)
	CoordSourceX *int `json:"coord_source_x,omitempty" jsonschema:"anyof_type=integer;null,default=null"`
	CoordSourceY *int `json:"coord_source_y,omitempty" jsonschema:"anyof_type=integer;null,default=null"`
	CoordTargetX *int `json:"coord_target_x,omitempty" jsonschema:"anyof_type=integer;null,default=null"`
	CoordTargetY *int `json:"coord_target_y,omitempty" jsonschema:"anyof_type=integer;null,default=null"`

	// Common options
	Steps   *int `json:"steps,omitempty" jsonschema:"anyof_type=integer;null,default=null"`
	DelayMs *int `json:"delay_ms,omitempty" jsonschema:"anyof_type=integer;null,default=null"`
}

func NewDragDropAction() *DragDropAction {
	return &DragDropAction{
		ElementSource:       nil,
		ElementTarget:       nil,
		ElementSourceOffset: nil,
		ElementTargetOffset: nil,
		CoordSourceX:        nil,
		CoordSourceY:        nil,
		CoordTargetX:        nil,
		CoordTargetY:        nil,
		Steps:               playwright.Int(10), // default
		DelayMs:             playwright.Int(5),
	}
}
