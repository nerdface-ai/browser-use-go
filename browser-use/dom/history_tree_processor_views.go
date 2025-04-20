package dom

import "github.com/moznion/go-optional"

type HashedDomElement struct {
	BranchPathHash string
	AttributesHash string
	XpathHash      string
	// TextHash string
}

type Coordinates struct {
	X int
	Y int
}

func (c *Coordinates) ToDict() map[string]int {
	return map[string]int{
		"x": c.X,
		"y": c.Y,
	}
}

type CoordinateSet struct {
	TopLeft     Coordinates
	TopRight    Coordinates
	BottomLeft  Coordinates
	BottomRight Coordinates
	Center      Coordinates
	Width       int
	Height      int
}

func (c *CoordinateSet) ToDict() map[string]any {
	return map[string]any{
		"top_left":     c.TopLeft.ToDict(),
		"top_right":    c.TopRight.ToDict(),
		"bottom_left":  c.BottomLeft.ToDict(),
		"bottom_right": c.BottomRight.ToDict(),
		"center":       c.Center.ToDict(),
		"width":        c.Width,
		"height":       c.Height,
	}
}

type ViewportInfo struct {
	ScrollX int
	ScrollY int
	Width   int
	Height  int
}

func (v *ViewportInfo) ToDict() map[string]int {
	return map[string]int{
		"scroll_x": v.ScrollX,
		"scroll_y": v.ScrollY,
		"width":    v.Width,
		"height":   v.Height,
	}
}

type DOMHistoryElement struct {
	TagName                string
	Xpath                  string
	HighlightIndex         optional.Option[int]
	EntireParentBranchPath []string
	Attributes             map[string]string
	ShadowRoot             bool
	CssSelector            optional.Option[string]
	PageCoordinates        *CoordinateSet
	ViewportCoordinates    *CoordinateSet
	ViewportInfo           *ViewportInfo
}

func (e *DOMHistoryElement) ToDict() map[string]any {
	var pageCoordinates map[string]any = nil
	var viewportCoordinates map[string]any = nil
	var viewportInfo map[string]int = nil
	if e.PageCoordinates != nil {
		pageCoordinates = e.PageCoordinates.ToDict()
	}
	if e.ViewportCoordinates != nil {
		viewportCoordinates = e.ViewportCoordinates.ToDict()
	}
	if e.ViewportInfo != nil {
		viewportInfo = e.ViewportInfo.ToDict()
	}

	return map[string]any{
		"tag_name":                  e.TagName,
		"xpath":                     e.Xpath,
		"highlight_index":           e.HighlightIndex,
		"entire_parent_branch_path": e.EntireParentBranchPath,
		"attributes":                e.Attributes,
		"shadow_root":               e.ShadowRoot,
		"css_selector":              e.CssSelector,
		"page_coordinates":          pageCoordinates,
		"viewport_coordinates":      viewportCoordinates,
		"viewport_info":             viewportInfo,
	}
}
