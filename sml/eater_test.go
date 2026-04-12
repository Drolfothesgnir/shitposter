package sml

import (
	"testing"

	"github.com/Drolfothesgnir/shitposter/scum"
	"github.com/stretchr/testify/require"
)

func TestPoopText(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop, err := eater.Munch("pre $bold$ [link]!href{https://example.com} hé")
	require.NoError(t, err)

	require.Empty(t, poop.Warnings)
	require.Equal(t, "pre bold link hé", poop.Text())
	require.Equal(t, len("pre bold link hé"), poop.TextByteLen())
}
