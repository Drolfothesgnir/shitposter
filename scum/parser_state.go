package scum

// parserState holds all mutable state for the single-pass parser.
// It builds an [AST] incrementally as tokens are processed.
//
// The parser tracks the current position in the tree using three parallel stacks:
//   - breadcrumbs: indices into [AST.Nodes] forming the path from root to the
//     current open tag (the ancestor chain). The top element is the node that
//     new children get appended to.
//   - cumWidth: cumulative byte width at each depth. Each entry accumulates the
//     total byte width (opening tag + children + closing tag) of the node at
//     that depth. When a tag closes, its cumWidth entry is popped and added to
//     the parent's entry, which allows computing the final [Node.Span].End for
//     each node without a second pass.
//   - stack: the tag IDs (byte) of each open tag at each depth, used to match
//     closing tags against their openers and to detect duplicate nesting.
//
// These three stacks always have the same length (rooted at depth 0 for the
// root node) and are pushed/popped together when tags open and close.
type parserState struct {
	// ast is the AST being constructed.
	ast AST

	// breadcrumbs is the ancestor chain from the root to the current open tag.
	// Each entry is an index into [AST.Nodes]. The bottom element (index 0)
	// is always the root node. The top element is the current parent for
	// new children.
	breadcrumbs []int

	// cumWidth tracks the cumulative byte width at each nesting depth, parallel
	// to breadcrumbs. When a tag opens, a new entry is pushed with the opening
	// tag's width. As children (text or nested tags) are processed, the top
	// entry is incremented. When the tag closes, the entry is popped, the
	// closing tag's width is added, and the total is folded into the parent's
	// entry. This enables single-pass [Node.Span].End computation for every node.
	cumWidth []int

	// skip counts how many times each tag byte should be silently discarded.
	// Indexed by the tag's byte value. When a duplicate nested tag is detected,
	// its corresponding close tag ID gets an incremented skip count so the
	// orphaned closing token is consumed without error.
	skip [256]int

	// openedTags tracks which tag IDs are currently open anywhere in the
	// ancestor chain. Indexed by tag byte value. Used to detect duplicate
	// nesting: if openedTags[id] is true, a second opening of the same tag
	// is rejected.
	openedTags [256]bool

	// stack records the tag ID at each nesting depth, parallel to breadcrumbs.
	// Used to match closing tags: a closing token is valid only when its
	// corresponding open ID equals the top of this stack.
	stack []byte

	// maxDepth records the deepest nesting level reached during parsing.
	maxDepth int

	// lastNodeIdx is the index in [AST.Nodes] of the most recently created or
	// closed node. Attributes that follow a tag or text token are attached to
	// the node at this index.
	lastNodeIdx int

	// totalTagNodes counts non-greedy tag nodes that were successfully opened.
	// Duplicate nested tags and greedy tags are excluded from this count.
	totalTagNodes int

	// textNodes counts closing tags that were demoted to text nodes due to
	// an open/close mismatch. These are added to the tokenizer's TextTokens
	// count to produce [AST.TotalTextNodes].
	textNodes int
}

// peekCrumb returns the index inside [AST.Nodes] of the deepest last Node in the current branch.
func (s parserState) peekCrumb() int {
	// since there will always be a root node, 0 len should not be an issue
	return s.breadcrumbs[len(s.breadcrumbs)-1]
}

// popCrumb removes and returns the index inside [AST.Nodes] of the deepest Node in the current branch.
func (s *parserState) popCrumb() int {
	lastItemIdx := len(s.breadcrumbs) - 1
	lastItem := s.breadcrumbs[lastItemIdx]
	s.breadcrumbs = s.breadcrumbs[:lastItemIdx]
	return lastItem
}

// pushCrumb adds an index of the Node to the breadcrumbs slice.
func (s *parserState) pushCrumb(idx int) {
	s.breadcrumbs = append(s.breadcrumbs, idx)
}

// peekStack returns the ID of the latest opened Tag in the current branch.
func (s parserState) peekStack() byte {
	l := len(s.stack)
	if l > 0 {
		return s.stack[l-1]
	}

	return 0
}

// popStack removes the last item from the stack and returns it.
func (s *parserState) popStack() byte {
	lastItemIdx := len(s.stack) - 1
	lastItem := s.stack[lastItemIdx]
	s.stack = s.stack[:lastItemIdx]
	return lastItem
}

// pushStack adds new item to the end of the stack.
func (s *parserState) pushStack(b byte) {
	s.stack = append(s.stack, b)
	s.maxDepth = max(s.maxDepth, len(s.stack))
}

// incrementCumWidth adds w bytes to the cumulative width at the current depth.
// Called when a text or greedy token is consumed as a child of the current open tag.
func (s *parserState) incrementCumWidth(w int) {
	s.cumWidth[len(s.cumWidth)-1] += w
}

// pushCumWidth pushes a new depth level onto the cumWidth stack, initialized to
// w (typically the opening tag's byte width). Called when a new tag is opened.
func (s *parserState) pushCumWidth(w int) {
	s.cumWidth = append(s.cumWidth, w)
}

// popCumWidth pops the top cumWidth entry, adds the closing tag width w, and
// folds the total into the parent's entry. Returns the total byte width of
// the tag that was just closed (opening + children + closing).
func (s *parserState) popCumWidth(w int) int {
	lastItemIdx := len(s.cumWidth) - 1
	lastItem := s.cumWidth[lastItemIdx]
	delta := lastItem + w
	s.cumWidth[lastItemIdx-1] += delta
	s.cumWidth = s.cumWidth[:lastItemIdx]
	return delta
}

// peekCumWidth returns the cumulative byte width at the current depth without
// modifying the stack. Used to finalize the root node's Span.End.
func (s *parserState) peekCumWidth() int {
	return s.cumWidth[len(s.cumWidth)-1]
}

// newParserState initializes a parserState with a root node and pre-allocated
// arenas sized according to the tokenizer output.
func newParserState(input string, out TokenizerOutput) parserState {
	root := NewNode()

	totalExpectedNodes := max(out.TagsTotal+out.TextTokens, 1)
	nodes := make([]Node, 1, totalExpectedNodes)
	nodes[0] = root

	ast := AST{
		Input:      input,
		Nodes:      nodes,
		Attributes: make([]Attribute, 0, out.Attributes),
	}

	return parserState{
		ast:         ast,
		breadcrumbs: []int{0},  // root is always present
		cumWidth:    []int{0},  // root's cumulative width starts at 0
		maxDepth:    1,
	}
}
