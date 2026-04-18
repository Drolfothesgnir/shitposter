package sml

import (
	"testing"

	"github.com/Drolfothesgnir/shitposter/scum"
	"github.com/stretchr/testify/require"
)

func TestPoopHTML_RendersNestedTagsAndEscapesUserText(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop, issues := eater.Munch(`plain <&> $bold *italic _under [link <>&] _*$ tail`)
	html := poop.HTML()

	require.Empty(t, issues)
	require.Equal(t, `plain &lt;&amp;&gt; <strong>bold <em>italic <span class="sml-internal-underline">under <a>link &lt;&gt;&amp;</a> </span></em></strong> tail`, html)
}

func TestPoopHTML_RenderingMalformedNestedTagsStillEscapesText(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop, issues := eater.Munch(`$bold *italic$ <script>alert("nope")</script>*`)
	html := poop.HTML()

	require.NotEmpty(t, issues)
	require.NotContains(t, html, `<script>`)
	require.Contains(t, html, `&lt;script&gt;alert(&#34;nope&#34;)&lt;/script&gt;`)
}

func TestPoopHTML_AttributesOnNonLinkTagsAreRejected(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop, issues := eater.Munch(`$bold$!href{https://example.com} *italic*!target{blank} _under_`)
	html := poop.HTML()

	require.Equal(t, `<strong>bold</strong> <em>italic</em> <span class="sml-internal-underline">under</span>`, html)
	require.Len(t, issues, 2)
	requireIssueDescription(t, Issues{List: issues}, `unknown attribute "href" for the tag "BOLD"`)
	requireIssueDescription(t, Issues{List: issues}, `unknown attribute "target" for the tag "ITALIC"`)
}

func TestRenderNode_DefensiveUnknownNodeAndTagPanics(t *testing.T) {
	require.Panics(t, func() {
		renderNode(nil, scum.SerializableNode{
			Type: "Tag",
			Name: "CHAOS",
		})
	})
	require.Panics(t, func() {
		renderNode(nil, scum.SerializableNode{
			Type: "Portal",
			Name: "TEXT_BUT_SIDEWAYS",
		})
	})
}

func TestNormalizeNode_DefensiveUnknownNodeAndTagPanics(t *testing.T) {
	issues := Issues{}
	require.Panics(t, func() {
		normalizeNode(&scum.SerializableNode{
			Type: "Tag",
			Name: "CHAOS",
		}, &issues)
	})
	require.Panics(t, func() {
		normalizeNode(&scum.SerializableNode{
			Type: "Portal",
			Name: "TEXT_BUT_SIDEWAYS",
		}, &issues)
	})
}

func TestPoopHTML_NestedLinkAttributesAreNormalized(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop, issues := eater.Munch(`$bold [link]!onclick{alert(1)}!href{https://example.com?q=1&x=<y>}!target{_blank}$`)
	html := poop.HTML()

	require.Empty(t, issues)
	require.Equal(t, `<strong>bold <a href="https://example.com?q=1&amp;x=&lt;y&gt;" target="_blank" rel="noopener noreferrer">link</a></strong>`, html)
}

func TestNormalizeNode_StripsNestedTextAttributes(t *testing.T) {
	issues := Issues{}
	node := scum.SerializableNode{
		Type: "Tag",
		Name: Bold,
		Children: []scum.SerializableNode{
			{
				Type:    "Text",
				Name:    "TEXT",
				Content: "hello",
				Attributes: []scum.SerializableAttribute{
					{Name: "style", Payload: "nope"},
				},
			},
		},
	}

	normalizeNode(&node, &issues)

	require.Empty(t, node.Children[0].Attributes)
	requireIssueDescription(t, issues, `unknown attribute "style" for the text node`)
}

func TestPoopHTML_LinkUsesFirstDuplicateAttributes(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop, issues := eater.Munch(`[link]!onclick{alert(1)}!href{https://first.example}!HREF{https://second.example}!target{_self}!TARGET{_blank}!title{first}!title{second}`)
	html := poop.HTML()

	require.Equal(t, `<a href="https://first.example" target="_self" title="first">link</a>`, html)
	require.Empty(t, issues)
}

func TestPoopHTML_LinkTargetBlankPayloadIsNotUnderlineMarkup(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop, issues := eater.Munch(`[link]!target{_blank}`)
	html := poop.HTML()

	require.Empty(t, issues)
	require.Equal(t, `<a target="_blank" rel="noopener noreferrer">link</a>`, html)
}

func TestPoopHTML_UnderlineClosesImmediatelyAfterLinkTargetBlank(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop, issues := eater.Munch(`_under [link]!target{_blank}_`)
	html := poop.HTML()

	require.Empty(t, issues)
	require.Equal(t, `<span class="sml-internal-underline">under <a target="_blank" rel="noopener noreferrer">link</a></span>`, html)
}

func TestPoopHTML_TightNestedClosersAfterLinkTargetBlank(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop, issues := eater.Munch(`$bold *italic _under [link]!target{_blank}_*$`)
	html := poop.HTML()

	require.Empty(t, issues)
	require.Equal(t, `<strong>bold <em>italic <span class="sml-internal-underline">under <a target="_blank" rel="noopener noreferrer">link</a></span></em></strong>`, html)
}

func TestPoopHTML_UnderlineClosesBeforeFollowingAttribute(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop, issues := eater.Munch(`_under_!title{hello}`)
	html := poop.HTML()

	require.Equal(t, `<span class="sml-internal-underline">under</span>`, html)
	require.Len(t, issues, 1)
	requireIssueDescription(t, Issues{List: issues}, `unknown attribute "title" for the tag "UNDERLINE"`)
}

func TestPoopHTML_LinkFlagCountsAsDuplicateAttributeName(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop, issues := eater.Munch(`[link]!{href}!href{https://example.com}`)
	html := poop.HTML()

	require.Equal(t, `<a href="https://example.com">link</a>`, html)
	requireIssueDescription(t, Issues{List: issues}, "attribute href must have a value")
}
