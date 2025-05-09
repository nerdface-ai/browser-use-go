package controller

import (
	"github.com/moznion/go-optional"
)

// Action Input Models
type SearchGoogleAction struct {
	Query string `json:"query"`
}

type GoToUrlAction struct {
	Url string `json:"url"`
}

type ClickElementAction struct {
	Index int                     `json:"index"`
	Xpath optional.Option[string] `json:"xpath,omitempty" jsonschema:"anyof_type=string;null,default=null"`
}

type InputTextAction struct {
	Index int                     `json:"index"`
	Text  string                  `json:"text"`
	Xpath optional.Option[string] `json:"xpath,omitempty" jsonschema:"anyof_type=string;null,default=null"`
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
	Amount optional.Option[int] `json:"amount,omitempty" jsonschema:"anyof_type=integer;null,default=null"`
}

type ScrollUpAction struct {
	Amount optional.Option[int] `json:"amount,omitempty" jsonschema:"anyof_type=integer;null,default=null"`
}

type SendKeysAction struct {
	Keys string `json:"keys"`
}

type GroupTabsAction struct {
	TabIds []int                   `json:"tab_ids"`
	Title  string                  `json:"title"`
	Color  optional.Option[string] `json:"color,omitempty" jsonschema:"anyof_type=string;null,default=null"`
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
	ElementSource       optional.Option[string]   `json:"element_source,omitempty" jsonschema:"anyof_type=string;null,default=null"`
	ElementTarget       optional.Option[string]   `json:"element_target,omitempty" jsonschema:"anyof_type=string;null,default=null"`
	ElementSourceOffset optional.Option[Position] `json:"element_source_offset,omitempty" jsonschema:"anyof_type=object;null,default=null"`
	ElementTargetOffset optional.Option[Position] `json:"element_target_offset,omitempty" jsonschema:"anyof_type=object;null,default=null"`

	// Coordinate-based approach (used if selectors not provided)
	CoordSourceX optional.Option[int] `json:"coord_source_x,omitempty" jsonschema:"anyof_type=integer;null,default=null"`
	CoordSourceY optional.Option[int] `json:"coord_source_y,omitempty" jsonschema:"anyof_type=integer;null,default=null"`
	CoordTargetX optional.Option[int] `json:"coord_target_x,omitempty" jsonschema:"anyof_type=integer;null,default=null"`
	CoordTargetY optional.Option[int] `json:"coord_target_y,omitempty" jsonschema:"anyof_type=integer;null,default=null"`

	// Common options
	Steps   optional.Option[int] `json:"steps,omitempty" jsonschema:"anyof_type=integer;null,default=null"`
	DelayMs optional.Option[int] `json:"delay_ms,omitempty" jsonschema:"anyof_type=integer;null,default=null"`
}

func NewDragDropAction() *DragDropAction {
	return &DragDropAction{
		ElementSource:       optional.None[string](),
		ElementTarget:       optional.None[string](),
		ElementSourceOffset: optional.None[Position](),
		ElementTargetOffset: optional.None[Position](),
		CoordSourceX:        optional.None[int](),
		CoordSourceY:        optional.None[int](),
		CoordTargetX:        optional.None[int](),
		CoordTargetY:        optional.None[int](),
		Steps:               optional.Some(10), // default
		DelayMs:             optional.Some(5),
	}
}
