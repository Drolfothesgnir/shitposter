package scum

// TODO: document steps for Myself
type parserState struct {
	ast           AST
	breadcrumbs   []int
	cumWidth      []int
	skip          [256]int
	openedTags    [256]bool
	stack         []byte
	maxDepth      int
	lastNodeIdx   int
	totalTagNodes int
	textNodes     int
}

func (s parserState) peekCrumb() int {
	// since there will always be a root node, 0 len should not be an issue
	return s.breadcrumbs[len(s.breadcrumbs)-1]
}

func (s *parserState) popCrumb() int {
	lastItemIdx := len(s.breadcrumbs) - 1
	lastItem := s.breadcrumbs[lastItemIdx]
	s.breadcrumbs = s.breadcrumbs[:lastItemIdx]
	return lastItem
}

func (s *parserState) pushCrumb(idx int) {
	s.breadcrumbs = append(s.breadcrumbs, idx)
}

func (s parserState) peekStack() byte {
	l := len(s.stack)
	if l > 0 {
		return s.stack[l-1]
	}

	return 0
}

func (s *parserState) popStack() byte {
	lastItemIdx := len(s.stack) - 1
	lastItem := s.stack[lastItemIdx]
	s.stack = s.stack[:lastItemIdx]
	return lastItem
}

func (s *parserState) pushStack(b byte) {
	s.stack = append(s.stack, b)
	s.maxDepth = max(s.maxDepth, len(s.stack))
}

func (s *parserState) incrementCumWidth(w int) {
	s.cumWidth[len(s.cumWidth)-1] += w
}

func (s *parserState) pushCumWidth(w int) {
	s.cumWidth = append(s.cumWidth, w)
}

func (s *parserState) popCumWidth(w int) int {
	lastItemIdx := len(s.cumWidth) - 1
	lastItem := s.cumWidth[lastItemIdx]
	delta := lastItem + w
	s.cumWidth[lastItemIdx-1] += delta
	s.cumWidth = s.cumWidth[:lastItemIdx]
	return delta
}

func (s *parserState) peekCumWidth() int {
	return s.cumWidth[len(s.cumWidth)-1]
}

func newParserState(input string, out TokenizerOutput) parserState {
	root := NewNode()

	// we expect to have at most as many nodes as there are tags and text tokens
	totalExpectedNodes := max(out.TagsTotal+out.TextTokens, 1)
	nodes := make([]Node, 1, totalExpectedNodes)
	nodes[0] = root

	ast := AST{
		Input:      input,
		Nodes:      nodes,
		Attributes: make([]Attribute, 0, out.Attributes),
	}

	return parserState{
		ast: ast,
		// root node should always be present
		breadcrumbs: []int{0},
		// cumulative width of the root should always be present
		cumWidth: []int{0},
		maxDepth: 1,
	}
}
