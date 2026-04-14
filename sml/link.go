package sml

import (
	"fmt"
	"html"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/Drolfothesgnir/shitposter/scum"
)

const MaxTitleLength int = 65

func basicLinkCheck(i *Issues, a scum.SerializableAttribute, attrName string) (string, bool) {
	if a.IsFlag {
		i.Add(NewSyntaxIssueDescriptor(IssueAttributeInvalidPayload, fmt.Sprintf("attribute %s must have a value", attrName)))
		return "", false
	}

	payload := strings.TrimSpace(a.Payload)
	if payload == "" {
		i.Add(NewSyntaxIssueDescriptor(IssueAttributeInvalidPayload, fmt.Sprintf("attribute %s must not be empty", attrName)))
		return "", false
	}

	if strings.ContainsAny(payload, "\x00\r\n\t") {
		i.Add(NewSyntaxIssueDescriptor(IssueAttributeInvalidPayload, fmt.Sprintf("attribute %s contains forbidden control characters", attrName)))
		return "", false
	}

	return payload, true
}

func attrHref(b *strings.Builder, i *Issues, a scum.SerializableAttribute) bool {
	payload, ok := basicLinkCheck(i, a, "href")
	if !ok {
		return false
	}

	u, err := url.Parse(payload)
	if err != nil {
		i.Add(NewSyntaxIssueDescriptor(IssueAttributeInvalidPayload, fmt.Sprintf("attribute href is invalid: %v", err)))
		return false
	}

	switch strings.ToLower(u.Scheme) {
	case "":
		// Allow relative references, but reject protocol-relative URLs such as //evil.com.
		if strings.HasPrefix(payload, "//") {
			i.Add(NewSyntaxIssueDescriptor(IssueAttributeInvalidPayload, "attribute href must not be protocol-relative"))
			return false
		}

	case "http", "https", "mailto":
		// Allowed schemes.

	default:
		i.Add(NewSyntaxIssueDescriptor(IssueAttributeInvalidPayload, fmt.Sprintf("attribute href scheme %q is not allowed", u.Scheme)))
		return false
	}

	b.WriteString(`href="`)
	b.WriteString(html.EscapeString(payload))
	b.WriteByte('"')
	return true
}

func attrTarget(b *strings.Builder, i *Issues, a scum.SerializableAttribute) bool {
	payload, ok := basicLinkCheck(i, a, "target")
	if !ok {
		return false
	}

	switch payload {
	case "_blank", "_self": // some others?
	default:
		i.Add(NewSyntaxIssueDescriptor(IssueAttributeInvalidPayload, `attribute target must be one of "_blank" or "_self"`))
		return false
	}

	b.WriteString(`target="`)
	b.WriteString(payload)
	b.WriteByte('"')

	if payload == "_blank" {
		b.WriteString(` rel="noopener noreferrer"`)
	}
	return true
}

func attrTitle(b *strings.Builder, i *Issues, a scum.SerializableAttribute) bool {
	payload, ok := basicLinkCheck(i, a, "title")
	if !ok {
		return false
	}

	if utf8.RuneCountInString(payload) > MaxTitleLength {
		// TODO: find the idiomatic way to make max title length configurable
		i.Add(NewSyntaxIssueDescriptor(IssueAttributeInvalidPayload, fmt.Sprintf("attribute title must be at most %d characters long", MaxTitleLength)))
		return false
	}

	b.WriteString(`title="`)
	b.WriteString(html.EscapeString(payload))
	b.WriteByte('"')
	return true
}
