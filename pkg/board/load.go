package board

import (
	"fmt"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"

	"github.com/err0r500/event-modeling-dcb-spec/pkg/render"
)

// Board holds a loaded and validated board.
type Board struct {
	Name  string
	Value cue.Value
	Flow  []FlowItem
}

// FlowItem is a lightweight representation of one instant in the flow.
type FlowItem struct {
	Index    int
	Kind     string // "slice" or "story"
	Name     string // slice name or (sliceRef) for stories
	Type     string // "change" or "view" (empty for story)
	SliceRef string // for stories: the referenced slice name
	CUEValue cue.Value
}

// LoadBoard loads a CUE file, finds the board, validates it, and extracts the flow.
func LoadBoard(filePath, boardName string) (*Board, error) {
	b, warnings, err := LoadBoardPermissive(filePath, boardName)
	if err != nil {
		return nil, err
	}
	if len(warnings) > 0 {
		return nil, fmt.Errorf("validation errors: %v", warnings)
	}
	return b, nil
}

// LoadBoardPermissive loads a board, returning validation issues as warnings instead of errors.
// Hard errors (CUE parse/build failures) are still returned as errors.
func LoadBoardPermissive(filePath, boardName string) (*Board, []string, error) {
	absFile, err := filepath.Abs(filePath)
	if err != nil {
		return nil, nil, fmt.Errorf("abs path: %w", err)
	}

	cfg := &load.Config{Dir: filepath.Dir(absFile)}
	instances := load.Instances([]string{"."}, cfg)
	if len(instances) == 0 {
		return nil, nil, fmt.Errorf("no instances loaded")
	}

	inst := instances[0]
	if inst.Err != nil {
		return nil, nil, fmt.Errorf("load: %w", inst.Err)
	}

	ctx := cuecontext.New()
	v := ctx.BuildInstance(inst)
	// Use Validate(All) to get full error details including type mismatches
	if err := v.Validate(cue.All()); err != nil {
		return nil, nil, fmt.Errorf("build: %s", render.FormatCUEError(err))
	}

	boardVal := FindBoard(v, boardName)
	if !boardVal.Exists() {
		return nil, nil, fmt.Errorf("board not found: %q", boardName)
	}
	if boardVal.Err() != nil {
		return nil, nil, fmt.Errorf("board: %s", render.FormatCUEError(boardVal.Err()))
	}

	warnings := render.ValidateBoard(boardVal)

	name := getString(boardVal, "name")
	flow, err := extractFlow(boardVal)
	if err != nil {
		return nil, nil, err
	}

	return &Board{Name: name, Value: boardVal, Flow: flow}, warnings, nil
}

// FindBoard finds a board in the CUE value by name, or returns the first board found.
func FindBoard(v cue.Value, boardName string) cue.Value {
	if boardName != "" {
		return v.LookupPath(cue.ParsePath(boardName))
	}
	iter, err := v.Fields()
	if err != nil {
		return cue.Value{}
	}
	for iter.Next() {
		val := iter.Value()
		if flow := val.LookupPath(cue.ParsePath("flow")); flow.Err() == nil {
			return val
		}
	}
	return cue.Value{}
}

func extractFlow(boardVal cue.Value) ([]FlowItem, error) {
	flowVal := boardVal.LookupPath(cue.ParsePath("flow"))
	if flowVal.Err() != nil {
		return nil, fmt.Errorf("flow not found: %w", flowVal.Err())
	}

	iter, err := flowVal.List()
	if err != nil {
		return nil, fmt.Errorf("flow list: %w", err)
	}

	var items []FlowItem
	idx := 0
	for iter.Next() {
		v := iter.Value()
		kind := getString(v, "kind")

		item := FlowItem{
			Index:    idx,
			Kind:     kind,
			CUEValue: v,
		}

		switch kind {
		case "slice":
			item.Name = getString(v, "name")
			item.Type = getString(v, "type")
		case "story":
			ref := getString(v, "slice.name")
			item.Name = "(" + ref + ")"
			item.SliceRef = ref
		}

		items = append(items, item)
		idx++
	}

	return items, nil
}

func getString(v cue.Value, path string) string {
	val := v.LookupPath(cue.ParsePath(path))
	if val.Err() != nil {
		return ""
	}
	s, err := val.String()
	if err != nil {
		return ""
	}
	return s
}
