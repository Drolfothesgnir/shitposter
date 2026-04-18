package sml

import (
	"fmt"
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

func validateHref(i *Issues, a scum.SerializableAttribute) bool {
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

	return true
}

func validateTarget(i *Issues, a scum.SerializableAttribute) (scum.SerializableAttribute, bool) {
	payload, ok := basicLinkCheck(i, a, "target")
	if !ok {
		return scum.SerializableAttribute{}, false
	}

	switch payload {
	case "_blank", "_self": // some others?
	default:
		i.Add(NewSyntaxIssueDescriptor(IssueAttributeInvalidPayload, `attribute target must be one of "_blank" or "_self"`))
		return scum.SerializableAttribute{}, false
	}

	var rel scum.SerializableAttribute
	if payload == "_blank" {
		rel.IsFlag = false
		rel.Name = "rel"
		rel.Payload = "noopener noreferrer"
	}
	return rel, true
}

func validateTitle(i *Issues, a scum.SerializableAttribute) bool {
	payload, ok := basicLinkCheck(i, a, "title")
	if !ok {
		return false
	}

	if utf8.RuneCountInString(payload) > MaxTitleLength {
		// TODO: find the idiomatic way to make max title length configurable
		i.Add(NewSyntaxIssueDescriptor(IssueAttributeInvalidPayload, fmt.Sprintf("attribute title must be at most %d characters long", MaxTitleLength)))
		return false
	}

	return true
}

func normalizeLink(n *scum.SerializableNode, issues *Issues) {
	var hasHref, hasTarget, hasTitle bool
	allowed := n.Attributes[:0]
	count := 3

	for i := 0; i < len(n.Attributes) && count > 0; i++ {
		a := n.Attributes[i]
		name := attrName(a)
		switch name {
		case "href":
			if hasHref {
				continue
			}

			if validateHref(issues, a) {
				allowed = append(allowed, a)
				hasHref = true
				count--
			}

		case "target":
			if hasTarget {
				continue
			}

			if rel, ok := validateTarget(issues, a); ok {
				allowed = append(allowed, a)
				if rel.Name != "" {
					allowed = append(allowed, rel)
				}
				hasTarget = true
				count--
			}

		case "title":
			if hasTitle {
				continue
			}

			if validateTitle(issues, a) {
				allowed = append(allowed, a)
				hasTitle = true
				count--
			}
		}
	}

	n.Attributes = allowed

	for i := range n.Children {
		c := &n.Children[i]
		normalizeNode(c, issues)
	}
}

func attrName(a scum.SerializableAttribute) string {
	if a.IsFlag {
		return strings.ToLower(a.Payload)
	}

	return strings.ToLower(a.Name)
}
