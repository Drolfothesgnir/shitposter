package sml

import (
	"strings"
	"testing"

	"github.com/Drolfothesgnir/shitposter/scum"
	"github.com/stretchr/testify/require"
)

func TestAttrHref_AllowsHTTPS(t *testing.T) {
	var b strings.Builder
	issues := Issues{}

	ok := attrHref(&b, &issues, scum.SerializableAttribute{
		Name:    "href",
		Payload: "https://example.com?q=1&x=<y>",
	})

	require.True(t, ok)
	require.Empty(t, issues.list)
	require.Equal(t, `href="https://example.com?q=1&amp;x=&lt;y&gt;"`, b.String())
}

func TestAttrHref_AllowsRelativePath(t *testing.T) {
	var b strings.Builder
	issues := Issues{}

	ok := attrHref(&b, &issues, scum.SerializableAttribute{
		Name:    "href",
		Payload: "/posts/42?tab=top",
	})

	require.True(t, ok)
	require.Empty(t, issues.list)
	require.Equal(t, `href="/posts/42?tab=top"`, b.String())
}

func TestAttrHref_RejectsFlag(t *testing.T) {
	var b strings.Builder
	issues := Issues{}

	ok := attrHref(&b, &issues, scum.SerializableAttribute{
		Payload: "href",
		IsFlag:  true,
	})

	require.False(t, ok)
	require.Empty(t, b.String())
	requireIssueDescription(t, issues, "attribute href must have a value")
}

func TestAttrHref_RejectsJavascriptScheme(t *testing.T) {
	var b strings.Builder
	issues := Issues{}

	ok := attrHref(&b, &issues, scum.SerializableAttribute{
		Name:    "href",
		Payload: "javascript:alert(1)",
	})

	require.False(t, ok)
	require.Empty(t, b.String())
	requireIssueDescription(t, issues, `attribute href scheme "javascript" is not allowed`)
}

func TestHandleAttributes_SkipsUnknownAndKeepsAllowed(t *testing.T) {
	var b strings.Builder
	issues := Issues{}

	handleAttributes(&b, &issues, attrMap{"href": attrHref}, scum.SerializableNode{
		Attributes: []scum.SerializableAttribute{
			{Name: "onclick", Payload: "alert(1)"},
			{Name: "href", Payload: "https://example.com"},
		},
	})

	require.Equal(t, ` href="https://example.com"`, b.String())
	requireIssueDescription(t, issues, "attribute onclick is not allowed")
}

func requireIssueDescription(t *testing.T, issues Issues, desc string) {
	t.Helper()

	for _, issue := range issues.list {
		if issue.Description() == desc {
			return
		}
	}

	require.Failf(t, "missing issue description", "expected issue description %q in %#v", desc, issues.list)
}
