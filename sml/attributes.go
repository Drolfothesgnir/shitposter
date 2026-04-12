package sml

import (
	"fmt"
	"strings"

	"github.com/Drolfothesgnir/shitposter/scum"
)

type attrMapper func(b *strings.Builder, i *Issues, a scum.SerializableAttribute) bool

type attrMap map[string]attrMapper

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
