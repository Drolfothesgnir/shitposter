package sml

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSyntaxIssueDescriptor_InvalidCodeFallsBackToInternal(t *testing.T) {
	issue := NewSyntaxIssueDescriptor(Issue(1), "something exploded sideways")

	require.Equal(t, int(IssueInternal), issue.Code())
	require.Equal(t, "INTERNAL", issue.Codename())
	require.Equal(t, "something exploded sideways", issue.Description())
	require.Equal(t, "SML: Codename - INTERNAL; something exploded sideways", issue.String())
}
