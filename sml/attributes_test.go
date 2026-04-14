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
	require.Empty(t, issues.List)
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
	require.Empty(t, issues.List)
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
			want:    `href="mailto:bugs@example.com?subject=&lt;hi&gt;&amp;body=&#34;quote&#34;"`,
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
			var b strings.Builder
			issues := Issues{}

			ok := attrHref(&b, &issues, scum.SerializableAttribute{
				Name:    "href",
				Payload: tt.payload,
			})

			if tt.wantDesc == "" {
				require.True(t, ok)
				require.Empty(t, issues.List)
				require.Equal(t, tt.want, b.String())
				return
			}

			require.False(t, ok)
			require.Empty(t, b.String())
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
			attr: scum.SerializableAttribute{Name: "target", Payload: "_blank"},
			want: `target="_blank" rel="noopener noreferrer"`,
		},
		{
			name: "allows self",
			attr: scum.SerializableAttribute{Name: "target", Payload: "_self"},
			want: `target="_self"`,
		},
		{
			name:     "rejects flag",
			attr:     scum.SerializableAttribute{Payload: "target", IsFlag: true},
			wantDesc: "attribute target must have a value",
		},
		{
			name:     "rejects parent",
			attr:     scum.SerializableAttribute{Name: "target", Payload: "_parent"},
			wantDesc: `attribute target must be one of "_blank" or "_self"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b strings.Builder
			issues := Issues{}

			ok := attrTarget(&b, &issues, tt.attr)

			if tt.wantDesc == "" {
				require.True(t, ok)
				require.Empty(t, issues.List)
				require.Equal(t, tt.want, b.String())
				return
			}

			require.False(t, ok)
			require.Empty(t, b.String())
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
			attr: scum.SerializableAttribute{Name: "title", Payload: strings.Repeat("a", MaxTitleLength)},
			want: `title="` + strings.Repeat("a", MaxTitleLength) + `"`,
		},
		{
			name: "escapes html characters",
			attr: scum.SerializableAttribute{Name: "title", Payload: `quote " <>&`},
			want: `title="quote &#34; &lt;&gt;&amp;"`,
		},
		{
			name:     "rejects too long",
			attr:     scum.SerializableAttribute{Name: "title", Payload: strings.Repeat("a", MaxTitleLength+1)},
			wantDesc: "attribute title must be at most 65 characters long",
		},
		{
			name:     "rejects flag",
			attr:     scum.SerializableAttribute{Payload: "title", IsFlag: true},
			wantDesc: "attribute title must have a value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b strings.Builder
			issues := Issues{}

			ok := attrTitle(&b, &issues, tt.attr)

			if tt.wantDesc == "" {
				require.True(t, ok)
				require.Empty(t, issues.List)
				require.Equal(t, tt.want, b.String())
				return
			}

			require.False(t, ok)
			require.Empty(t, b.String())
			requireIssueDescription(t, issues, tt.wantDesc)
		})
	}
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

	for _, issue := range issues.List {
		if issue.Description() == desc {
			return
		}
	}

	require.Failf(t, "missing issue description", "expected issue description %q in %#v", desc, issues.List)
}
