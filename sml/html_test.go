package sml

import (
	"strings"
	"testing"

	"github.com/Drolfothesgnir/shitposter/scum"
	"github.com/stretchr/testify/require"
)

func TestPoopHTML_RendersNestedTagsAndEscapesUserText(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop := eater.Munch(`plain <&> $bold *italic _under [link <>&] _*$ tail`)
	html, issues := poop.HTML()

	require.Empty(t, poop.Warnings)
	require.Empty(t, issues)
	require.Equal(t, `plain &lt;&amp;&gt; <strong>bold <em>italic <span class="sml-internal-underline">under <a>link &lt;&gt;&amp;</a> </span></em></strong> tail`, html)
}

func TestPoopHTML_RenderingMalformedNestedTagsStillEscapesText(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop := eater.Munch(`$bold *italic$ <script>alert("nope")</script>*`)
	html, issues := poop.HTML()

	require.NotEmpty(t, poop.Warnings)
	require.Empty(t, issues)
	require.NotContains(t, html, `<script>`)
	require.Contains(t, html, `&lt;script&gt;alert(&#34;nope&#34;)&lt;/script&gt;`)
}

func TestPoopHTML_AttributesOnNonLinkTagsAreRejected(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop := eater.Munch(`$bold$!href{https://example.com} *italic*!target{blank} _under_`)
	html, issues := poop.HTML()

	require.Equal(t, `<strong>bold</strong> <em>italic</em> <span class="sml-internal-underline">under</span>`, html)
	requireIssueDescription(t, Issues{List: issues}, "attribute href is not allowed")
	requireIssueDescription(t, Issues{List: issues}, "attribute target is not allowed")
}

func TestHandleNode_DefensiveUnknownNodeAndTagIssues(t *testing.T) {
	var b strings.Builder
	issues := Issues{}

	handleNode(&b, &issues, scum.SerializableNode{
		Type: "Tag",
		Name: "CHAOS",
	})
	handleNode(&b, &issues, scum.SerializableNode{
		Type: "Portal",
		Name: "TEXT_BUT_SIDEWAYS",
	})

	require.Empty(t, b.String())
	requireIssueDescription(t, issues, `unknown tag "CHAOS" encountered`)
	requireIssueDescription(t, issues, "unknown node type encountered: Portal")
}

func TestPoopHTML_LinkUsesFirstDuplicateAttributes(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop := eater.Munch(`[link]!onclick{alert(1)}!href{https://first.example}!HREF{https://second.example}!target{_self}!TARGET{_blank}!title{first}!title{second}`)
	html, issues := poop.HTML()

	require.Equal(t, `<a href="https://first.example" target="_self" title="first">link</a>`, html)
	requireIssueDescription(t, Issues{List: issues}, "attribute onclick is not allowed")
}

func TestPoopHTML_LinkTargetBlankPayloadIsNotUnderlineMarkup(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop := eater.Munch(`[link]!target{_blank}`)
	html, issues := poop.HTML()

	require.Empty(t, poop.Warnings)
	require.Empty(t, issues)
	require.Equal(t, `<a target="_blank" rel="noopener noreferrer">link</a>`, html)
}

func TestPoopHTML_UnderlineClosesImmediatelyAfterLinkTargetBlank(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop := eater.Munch(`_under [link]!target{_blank}_`)
	html, issues := poop.HTML()

	require.Empty(t, poop.Warnings)
	require.Empty(t, issues)
	require.Equal(t, `<span class="sml-internal-underline">under <a target="_blank" rel="noopener noreferrer">link</a></span>`, html)
}

func TestPoopHTML_TightNestedClosersAfterLinkTargetBlank(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop := eater.Munch(`$bold *italic _under [link]!target{_blank}_*$`)
	html, issues := poop.HTML()

	require.Empty(t, poop.Warnings)
	require.Empty(t, issues)
	require.Equal(t, `<strong>bold <em>italic <span class="sml-internal-underline">under <a target="_blank" rel="noopener noreferrer">link</a></span></em></strong>`, html)
}

func TestPoopHTML_UnderlineClosesBeforeFollowingAttribute(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop := eater.Munch(`_under_!title{hello}`)
	html, issues := poop.HTML()

	require.Empty(t, poop.Warnings)
	require.Equal(t, `<span class="sml-internal-underline">under</span>`, html)
	requireIssueDescription(t, Issues{List: issues}, "attribute title is not allowed")
}

func TestPoopHTML_LinkFlagCountsAsDuplicateAttributeName(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop := eater.Munch(`[link]!{href}!href{https://example.com}`)
	html, issues := poop.HTML()

	require.Equal(t, `<a>link</a>`, html)
	requireIssueDescription(t, Issues{List: issues}, "attribute href must have a value")
}
