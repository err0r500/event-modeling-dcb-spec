package tui

import "github.com/err0r500/event-modeling-dcb-spec/pkg/board"

// NodeKind identifies tree node types.
type NodeKind int

const (
	NodeContext NodeKind = iota
	NodeChapter
	NodeSlice
)

// TreeNode represents a node in the hierarchical tree.
type TreeNode struct {
	Kind        NodeKind
	Name        string
	Description string
	Depth       int // 0=context, 1=chapter, 2=slice
	FlowIndex   int // -1 for non-slices
	Children    []*TreeNode
	Parent      *TreeNode

	// For slices: extra display info
	SliceType string // "change" or "view"
	DevStatus string
}

// TreeState manages expand/collapse state and cursor position.
type TreeState struct {
	Nodes    []*TreeNode          // root nodes (contexts)
	Expanded map[*TreeNode]bool   // expansion state
	FlatView []*TreeNode          // visible nodes based on expansion
	Cursor   int                  // cursor in FlatView
	nodeByFlowIndex map[int]*TreeNode // lookup slice nodes by flow index
}

// NewTreeState creates tree state from manifest contexts.
func NewTreeState(manifest *board.BoardManifest, slices map[string]map[string]any) *TreeState {
	ts := &TreeState{
		Expanded:        make(map[*TreeNode]bool),
		nodeByFlowIndex: make(map[int]*TreeNode),
	}

	for _, ctx := range manifest.Contexts {
		ctxNode := &TreeNode{
			Kind:        NodeContext,
			Name:        ctx.Name,
			Description: ctx.Description,
			Depth:       0,
			FlowIndex:   -1,
		}

		for _, chap := range ctx.Chapters {
			chapNode := &TreeNode{
				Kind:        NodeChapter,
				Name:        chap.Name,
				Description: chap.Description,
				Depth:       1,
				FlowIndex:   -1,
				Parent:      ctxNode,
			}

			for _, idx := range chap.FlowIndices {
				if idx < 0 || idx >= len(manifest.Flow) {
					continue
				}
				entry := manifest.Flow[idx]

				sliceNode := &TreeNode{
					Kind:      NodeSlice,
					Name:      entry.Name,
					Depth:     2,
					FlowIndex: idx,
					Parent:    chapNode,
					SliceType: entry.Type,
				}

				// Get devstatus from slice data
				if data, ok := slices[entry.File]; ok {
					if ds, ok := data["devstatus"].(string); ok {
						sliceNode.DevStatus = ds
					}
				}

				chapNode.Children = append(chapNode.Children, sliceNode)
				ts.nodeByFlowIndex[idx] = sliceNode
			}

			ctxNode.Children = append(ctxNode.Children, chapNode)
		}

		ts.Nodes = append(ts.Nodes, ctxNode)
	}

	// Start with all contexts expanded
	for _, n := range ts.Nodes {
		ts.Expanded[n] = true
	}

	ts.rebuildFlatView()
	return ts
}

// rebuildFlatView updates FlatView based on current expansion state.
func (ts *TreeState) rebuildFlatView() {
	ts.FlatView = nil
	for _, node := range ts.Nodes {
		ts.addToFlatView(node)
	}
}

func (ts *TreeState) addToFlatView(node *TreeNode) {
	ts.FlatView = append(ts.FlatView, node)
	if ts.Expanded[node] {
		for _, child := range node.Children {
			ts.addToFlatView(child)
		}
	}
}

// Toggle expands or collapses the node at cursor.
func (ts *TreeState) Toggle() {
	if ts.Cursor < 0 || ts.Cursor >= len(ts.FlatView) {
		return
	}
	node := ts.FlatView[ts.Cursor]
	if len(node.Children) == 0 {
		return
	}
	ts.Expanded[node] = !ts.Expanded[node]
	ts.rebuildFlatView()
	// Clamp cursor if needed
	if ts.Cursor >= len(ts.FlatView) {
		ts.Cursor = len(ts.FlatView) - 1
	}
}

// Expand expands the node at cursor. Returns true if node is a slice (leaf).
func (ts *TreeState) Expand() bool {
	if ts.Cursor < 0 || ts.Cursor >= len(ts.FlatView) {
		return false
	}
	node := ts.FlatView[ts.Cursor]
	if len(node.Children) == 0 {
		// Leaf node (slice) - signal to open detail
		return node.Kind == NodeSlice
	}
	ts.Expanded[node] = true
	ts.rebuildFlatView()
	return false
}

// Collapse collapses current node or jumps to parent.
func (ts *TreeState) Collapse() {
	if ts.Cursor < 0 || ts.Cursor >= len(ts.FlatView) {
		return
	}
	node := ts.FlatView[ts.Cursor]

	// If expanded with children, collapse it
	if ts.Expanded[node] && len(node.Children) > 0 {
		ts.Expanded[node] = false
		ts.rebuildFlatView()
		return
	}

	// Otherwise jump to parent
	if node.Parent != nil {
		ts.moveTo(node.Parent)
	}
}

// MoveUp moves cursor up in flat view.
func (ts *TreeState) MoveUp() {
	if ts.Cursor > 0 {
		ts.Cursor--
	}
}

// MoveDown moves cursor down in flat view.
func (ts *TreeState) MoveDown() {
	if ts.Cursor < len(ts.FlatView)-1 {
		ts.Cursor++
	}
}

// moveTo moves cursor to given node.
func (ts *TreeState) moveTo(target *TreeNode) {
	for i, node := range ts.FlatView {
		if node == target {
			ts.Cursor = i
			return
		}
	}
}

// Current returns the currently selected node, or nil.
func (ts *TreeState) Current() *TreeNode {
	if ts.Cursor < 0 || ts.Cursor >= len(ts.FlatView) {
		return nil
	}
	return ts.FlatView[ts.Cursor]
}

// CurrentFlowIndex returns the flow index of current node, or -1.
func (ts *TreeState) CurrentFlowIndex() int {
	if node := ts.Current(); node != nil {
		return node.FlowIndex
	}
	return -1
}
