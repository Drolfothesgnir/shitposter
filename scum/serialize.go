package scum

var mapNodeTypeToName [NumNodeTypes]string

func init() {
	mapNodeTypeToName[NodeRoot] = "Root"
	mapNodeTypeToName[NodeTag] = "Tag"
	mapNodeTypeToName[NodeText] = "Text"
}

// SerializableAttribute is a JSON-friendly view of an [Attribute].
// Unlike [Attribute], it stores resolved strings instead of spans into [AST.Input].
type SerializableAttribute struct {
	Name    string `json:"name"`
	Payload string `json:"payload"`
	IsFlag  bool   `json:"is_flag"`
}

// serializeAttributes resolves attribute spans into strings and appends them to dest.
func serializeAttributes(dest *[]SerializableAttribute, ast *AST, r Range) {
	win := ast.Attributes[r.Start : r.Start+r.Len]
	for _, a := range win {
		*dest = append(*dest, SerializableAttribute{
			Name:    ast.Input[a.Name.Start:a.Name.End],
			Payload: ast.Input[a.Payload.Start:a.Payload.End],
			IsFlag:  a.IsFlag,
		})
	}
}

// SerializableNode is a JSON-friendly tree node produced by [AST.Serialize].
//
// It describes parser output, not render semantics. The tree tells the consumer
// what was parsed and how nodes are nested, while the renderer decides how a
// particular node name, ID, content and attributes should be interpreted.
type SerializableNode struct {
	// Name is the semantic node name.
	//
	// For tag nodes it comes from the [Dictionary]. For non-tag nodes serializer
	// uses conventional names such as "ROOT" and "TEXT".
	Name string `json:"name"`

	// Type is the coarse node kind: "Root", "Tag" or "Text".
	Type string `json:"type"`

	// Attributes contains attributes attached directly to this node.
	Attributes []SerializableAttribute `json:"attributes"`

	// Content is the exact slice of the original input covered by this node.
	//
	// For text nodes this is plain text. For tag nodes this is parser data, not
	// a ready-to-render value: renderers may interpret it differently depending
	// on the tag's meaning and may also rely on Children.
	Content string `json:"content"`

	// Children contains this node's parsed descendants in source order.
	Children []SerializableNode `json:"children"`

	// ID is the numeric trigger byte of the tag.
	//
	// It is 0 for nodes that are not backed by a concrete tag, such as the root.
	ID byte `json:"id"`
}

type serializeTask struct {
	parent   *SerializableNode
	childIdx int // index in parent.Children
	nodeIdx  int // index in ast.Nodes
}

// Serialize converts [AST] into a JSON-friendly tree.
//
// The provided [Dictionary] is used to resolve tag names for tag nodes.
// The result is meant to be a stable transport shape for APIs, previews and
// tooling; it should not be treated as final rendered output.
func (ast AST) Serialize(d *Dictionary) (tree SerializableNode) {
	root := ast.Nodes[0]

	// Single allocation for ALL child nodes (total nodes minus root)
	allChildren := make([]SerializableNode, len(ast.Nodes)-1)
	childrenUsed := 0

	rootAttrs := make([]SerializableAttribute, 0, root.Attributes.Len)

	serializeAttributes(&rootAttrs, &ast, root.Attributes)

	tree = SerializableNode{
		Name:       "ROOT",
		Type:       mapNodeTypeToName[NodeRoot],
		Content:    ast.Input[root.Span.Start:root.Span.End],
		Children:   allChildren[childrenUsed : childrenUsed+root.ChildCount],
		ID:         root.TagID,
		Attributes: rootAttrs,
	}
	childrenUsed += root.ChildCount

	stack := make([]serializeTask, 0, ast.MaxDepth)
	childIdx := root.FirstChild
	for i := 0; i < root.ChildCount; i++ {
		stack = append(stack, serializeTask{&tree, i, childIdx})
		childIdx = ast.Nodes[childIdx].NextSibling
	}

	for len(stack) > 0 {
		task := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		node := ast.Nodes[task.nodeIdx]

		// Slice from pre-allocated backing array
		children := allChildren[childrenUsed : childrenUsed+node.ChildCount]
		childrenUsed += node.ChildCount

		nodeAttrs := make([]SerializableAttribute, 0, node.Attributes.Len)

		serializeAttributes(&nodeAttrs, &ast, node.Attributes)

		var name string
		if node.Type == NodeTag {
			name = d.tags[node.TagID].Name
		} else {
			name = "TEXT"
		}

		task.parent.Children[task.childIdx] = SerializableNode{
			Name:       name,
			Type:       mapNodeTypeToName[node.Type],
			Content:    ast.Input[node.Span.Start:node.Span.End],
			Children:   children,
			ID:         node.TagID,
			Attributes: nodeAttrs,
		}

		// push children tasks
		childIdx := node.FirstChild
		for i := 0; i < node.ChildCount; i++ {
			stack = append(stack, serializeTask{
				parent:   &task.parent.Children[task.childIdx],
				childIdx: i,
				nodeIdx:  childIdx,
			})
			childIdx = ast.Nodes[childIdx].NextSibling
		}
	}

	return
}
