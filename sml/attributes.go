package sml

import (
	"fmt"
	"strings"

	"github.com/Drolfothesgnir/shitposter/scum"
)

// attrMapper handles the attribute a by converting it into an valid HTML attribute and
// writing the result into builder b, possibly adding some [SyntaxIssue]s to the issues list i.
// If the attribute a's payload is considered invalid and should not be added to the HTML string - returns false.
// Otherwise true should be returned.
type attrMapper func(b *strings.Builder, i *Issues, a scum.SerializableAttribute) bool

// attrMap map an attribute's name to its appropriate [attrMapper].
type attrMap map[string]attrMapper

// handleAttributes is used during the conversion of the input to the HTML string.
// It takes attributes of the node n and writes their HTML representation to the builder b.
func handleAttributes(b *strings.Builder, i *Issues, m attrMap, n scum.SerializableNode) {
	for _, a := range n.Attributes {
		var name string
		if a.IsFlag {
			name = strings.ToLower(a.Payload)
		} else {
			name = strings.ToLower(a.Name)
		}

		fn, ok := m[name]
		if !ok {
			i.Add(NewSyntaxIssueDescriptor(IssueAttributeNotAllowed, fmt.Sprintf("attribute %s is not allowed", name)))
			continue
		}

		var attr strings.Builder
		if !fn(&attr, i, a) {
			continue
		}

		b.WriteByte(' ')
		b.WriteString(attr.String())
	}
}
