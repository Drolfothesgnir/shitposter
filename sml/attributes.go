package sml

import (
	"fmt"
	"html"
	"net/url"
	"strings"

	"github.com/Drolfothesgnir/shitposter/scum"
)

type attrMapper func(b *strings.Builder, w *[]string, a scum.SerializableAttribute) bool

type attrMap map[string]attrMapper

func handleAttributes(b *strings.Builder, w *[]string, m attrMap, n scum.SerializableNode) {
	for _, a := range n.Attributes {
		var name string
		if a.IsFlag {
			name = strings.ToLower(a.Payload)
		} else {
			name = strings.ToLower(a.Name)
		}

		fn, ok := m[name]
		if !ok {
			*w = append(*w, fmt.Sprintf("attribute %s is not allowed", name))
			continue
		}

		var attr strings.Builder
		if !fn(&attr, w, a) {
			continue
		}

		b.WriteByte(' ')
		b.WriteString(attr.String())
	}
}

func attrHref(b *strings.Builder, w *[]string, a scum.SerializableAttribute) bool {
	if a.IsFlag {
		*w = append(*w, "attribute href must have a value")
		return false
	}

	payload := strings.TrimSpace(a.Payload)
	if payload == "" {
		*w = append(*w, "attribute href must not be empty")
		return false
	}

	if strings.ContainsAny(payload, "\x00\r\n\t") {
		*w = append(*w, "attribute href contains forbidden control characters")
		return false
	}

	u, err := url.Parse(payload)
	if err != nil {
		*w = append(*w, fmt.Sprintf("attribute href is invalid: %v", err))
		return false
	}

	switch strings.ToLower(u.Scheme) {
	case "":
		// Allow relative references, but reject protocol-relative URLs such as //evil.com.
		if strings.HasPrefix(payload, "//") {
			*w = append(*w, "attribute href must not be protocol-relative")
			return false
		}

	case "http", "https", "mailto":
		// Allowed schemes.

	default:
		*w = append(*w, fmt.Sprintf("attribute href scheme %q is not allowed", u.Scheme))
		return false
	}

	b.WriteString(`href="`)
	b.WriteString(html.EscapeString(payload))
	b.WriteByte('"')
	return true
}
