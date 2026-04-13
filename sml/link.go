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

func attrHref(b *strings.Builder, i *Issues, a scum.SerializableAttribute) bool {
	if a.IsFlag {
		i.Add(NewSyntaxIssueDescriptor(IssueAttributeInvalidPayload, "attribute href must have a value"))
		return false
	}

	payload := strings.TrimSpace(a.Payload)
	if payload == "" {
		i.Add(NewSyntaxIssueDescriptor(IssueAttributeInvalidPayload, "attribute href must not be empty"))
		return false
	}

	if strings.ContainsAny(payload, "\x00\r\n\t") {
		i.Add(NewSyntaxIssueDescriptor(IssueAttributeInvalidPayload, "attribute href contains forbidden control characters"))
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
	if a.IsFlag {
		i.Add(NewSyntaxIssueDescriptor(IssueAttributeInvalidPayload, "attribute target must have a value"))
		return false
	}

	payload := strings.TrimSpace(a.Payload)
	if payload == "" {
		i.Add(NewSyntaxIssueDescriptor(IssueAttributeInvalidPayload, "attribute target must not be empty"))
		return false
	}

	if strings.ContainsAny(payload, "\x00\r\n\t") {
		i.Add(NewSyntaxIssueDescriptor(IssueAttributeInvalidPayload, "attribute target contains forbidden control characters"))
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
	if a.IsFlag {
		i.Add(NewSyntaxIssueDescriptor(IssueAttributeInvalidPayload, "attribute title must have a value"))
		return false
	}

	payload := strings.TrimSpace(a.Payload)
	if payload == "" {
		i.Add(NewSyntaxIssueDescriptor(IssueAttributeInvalidPayload, "attribute title must not be empty"))
		return false
	}

	if strings.ContainsAny(payload, "\x00\r\n\t") {
		i.Add(NewSyntaxIssueDescriptor(IssueAttributeInvalidPayload, "attribute title contains forbidden control characters"))
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
