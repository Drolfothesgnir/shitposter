package scum

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	d := testDict(t)
	warns := newWarnings(t)

	input := "Hello, $$world\\!$$"
	tree := Parse(input, &d, warns)
	require.Empty(t, warns.List())
	require.NotNil(t, tree)

}
