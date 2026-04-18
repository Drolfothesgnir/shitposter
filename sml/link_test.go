package sml

import (
	"strings"
	"testing"

	"github.com/Drolfothesgnir/shitposter/scum"
	"github.com/stretchr/testify/require"
)

func linkAttr(name, payload string) scum.SerializableAttribute {
	return scum.SerializableAttribute{Name: name, Payload: payload}
}

func linkFlag(payload string) scum.SerializableAttribute {
	return scum.SerializableAttribute{Payload: payload, IsFlag: true}
}

func renderNormalizedLink(attrs []scum.SerializableAttribute, children ...scum.SerializableNode) (string, Issues) {
	node := scum.SerializableNode{
		Name:       Link,
		Type:       "Tag",
		Attributes: append([]scum.SerializableAttribute(nil), attrs...),
		Children:   children,
	}
	issues := Issues{}

	normalizeLink(&node, &issues)

	var b strings.Builder
	renderLink(&b, node)
	return b.String(), issues
}

func TestAttrHref_AllowsHTTPS(t *testing.T) {
	html, issues := renderNormalizedLink([]scum.SerializableAttribute{
		linkAttr("href", "https://example.com?q=1&x=<y>"),
	})

	require.Empty(t, issues.List)
	require.Equal(t, `<a href="https://example.com?q=1&amp;x=&lt;y&gt;"></a>`, html)
}

func TestAttrHref_AllowsRelativePath(t *testing.T) {
	html, issues := renderNormalizedLink([]scum.SerializableAttribute{
		linkAttr("href", "/posts/42?tab=top"),
	})

	require.Empty(t, issues.List)
	require.Equal(t, `<a href="/posts/42?tab=top"></a>`, html)
}

func TestAttrHref_RejectsFlag(t *testing.T) {
	html, issues := renderNormalizedLink([]scum.SerializableAttribute{
		linkFlag("href"),
	})

	require.Equal(t, `<a></a>`, html)
	requireIssueDescription(t, issues, "attribute href must have a value")
}

func TestAttrHref_RejectsJavascriptScheme(t *testing.T) {
	html, issues := renderNormalizedLink([]scum.SerializableAttribute{
		linkAttr("href", "javascript:alert(1)"),
	})

	require.Equal(t, `<a></a>`, html)
	requireIssueDescription(t, issues, `attribute href scheme "javascript" is not allowed`)
}

func TestAttrHref_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		payload  string
		want     string
		wantDesc string
	}{
		{
			name:    "allows mailto and escapes",
			payload: `mailto:bugs@example.com?subject=<hi>&body="quote"`,
			want:    `<a href="mailto:bugs@example.com?subject=&lt;hi&gt;&amp;body=&#34;quote&#34;"></a>`,
		},
		{
			name:     "rejects empty payload after trimming",
			payload:  " \t ",
			wantDesc: "attribute href must not be empty",
		},
		{
			name:     "rejects control characters",
			payload:  "https://example.com/\nwat",
			wantDesc: "attribute href contains forbidden control characters",
		},
		{
			name:     "rejects invalid url escape",
			payload:  "https://example.com/%zz",
			wantDesc: `attribute href is invalid: parse "https://example.com/%zz": invalid URL escape "%zz"`,
		},
		{
			name:     "rejects protocol relative urls",
			payload:  "//evil.example/borrow-your-cookies",
			wantDesc: "attribute href must not be protocol-relative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, issues := renderNormalizedLink([]scum.SerializableAttribute{
				linkAttr("href", tt.payload),
			})

			if tt.wantDesc == "" {
				require.Empty(t, issues.List)
				require.Equal(t, tt.want, html)
				return
			}

			require.Equal(t, `<a></a>`, html)
			requireIssueDescription(t, issues, tt.wantDesc)
		})
	}
}

func TestAttrTarget_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		attr     scum.SerializableAttribute
		want     string
		wantDesc string
	}{
		{
			name: "allows blank with rel",
			attr: linkAttr("target", "_blank"),
			want: `<a target="_blank" rel="noopener noreferrer"></a>`,
		},
		{
			name: "allows self",
			attr: linkAttr("target", "_self"),
			want: `<a target="_self"></a>`,
		},
		{
			name:     "rejects flag",
			attr:     linkFlag("target"),
			wantDesc: "attribute target must have a value",
		},
		{
			name:     "rejects parent",
			attr:     linkAttr("target", "_parent"),
			wantDesc: `attribute target must be one of "_blank" or "_self"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, issues := renderNormalizedLink([]scum.SerializableAttribute{tt.attr})

			if tt.wantDesc == "" {
				require.Empty(t, issues.List)
				require.Equal(t, tt.want, html)
				return
			}

			require.Equal(t, `<a></a>`, html)
			requireIssueDescription(t, issues, tt.wantDesc)
		})
	}
}

func TestAttrTitle_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		attr     scum.SerializableAttribute
		want     string
		wantDesc string
	}{
		{
			name: "allows exact max length",
			attr: linkAttr("title", strings.Repeat("a", MaxTitleLength)),
			want: `<a title="` + strings.Repeat("a", MaxTitleLength) + `"></a>`,
		},
		{
			name: "escapes html characters",
			attr: linkAttr("title", `quote " <>&`),
			want: `<a title="quote &#34; &lt;&gt;&amp;"></a>`,
		},
		{
			name:     "rejects too long",
			attr:     linkAttr("title", strings.Repeat("a", MaxTitleLength+1)),
			wantDesc: "attribute title must be at most 65 characters long",
		},
		{
			name:     "rejects flag",
			attr:     linkFlag("title"),
			wantDesc: "attribute title must have a value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html, issues := renderNormalizedLink([]scum.SerializableAttribute{tt.attr})

			if tt.wantDesc == "" {
				require.Empty(t, issues.List)
				require.Equal(t, tt.want, html)
				return
			}

			require.Equal(t, `<a></a>`, html)
			requireIssueDescription(t, issues, tt.wantDesc)
		})
	}
}

func TestNormalizeLink_UsesFirstValidAttributesAndEscapesChildren(t *testing.T) {
	html, issues := renderNormalizedLink(
		[]scum.SerializableAttribute{
			linkAttr("onclick", "alert(1)"),
			linkAttr("href", "javascript:alert(1)"),
			linkAttr("href", "https://example.com?q=1&x=<y>"),
			linkAttr("href", "https://ignored.example"),
			linkAttr("target", "_parent"),
			linkAttr("target", "_blank"),
			linkFlag("title"),
			linkAttr("title", `safe "title" <ok>`),
		},
		scum.SerializableNode{
			Type:    "Text",
			Name:    "TEXT",
			Content: `click <here> & now`,
		},
	)

	require.Equal(t, `<a href="https://example.com?q=1&amp;x=&lt;y&gt;" target="_blank" rel="noopener noreferrer" title="safe &#34;title&#34; &lt;ok&gt;">click &lt;here&gt; &amp; now</a>`, html)
	requireIssueDescription(t, issues, `attribute href scheme "javascript" is not allowed`)
	requireIssueDescription(t, issues, `attribute target must be one of "_blank" or "_self"`)
	requireIssueDescription(t, issues, "attribute title must have a value")
}

func requireIssueDescription(t *testing.T, issues Issues, desc string) {
	t.Helper()

	for _, issue := range issues.List {
		if issue.Description() == desc {
			return
		}
	}

	require.Failf(t, "missing issue description", "expected issue description %q in %#v", desc, issues.List)
}
