package sml

import (
	"errors"
	"testing"

	"github.com/Drolfothesgnir/shitposter/scum"
	"github.com/stretchr/testify/require"
)

func TestPoopText(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop := eater.Munch("pre $bold$ [link]!href{https://example.com} hé")

	require.Empty(t, poop.Warnings)
	require.Equal(t, "pre bold link hé", poop.Text())
	require.Equal(t, len("pre bold link hé"), poop.TextByteLen())
}

func TestEaterMunch_RecordsParserWarnings(t *testing.T) {
	eater, err := NewEater(scum.WarnOverflowNoCap, 0)
	require.NoError(t, err)

	poop := eater.Munch("$unclosed")

	require.NotEmpty(t, poop.Warnings)
	requireIssueCodename(t, poop.Warnings, "UNCLOSED_TAG")
}

func TestNewEater_InvalidWarningConfig(t *testing.T) {
	_, err := NewEater(scum.WarnOverflowDrop, -1)
	require.Error(t, err)

	var configErr *ConfigError
	require.True(t, errors.As(err, &configErr))
	require.Equal(t, "SML Parser", configErr.SubjectName)
	require.Equal(t, ReasonInvalidParams, configErr.Reason)
	require.Contains(t, configErr.Error(), "warnings cap must be non-negative")
	require.Error(t, configErr.Unwrap())
}

func requireIssueCodename(t *testing.T, issues []SyntaxIssue, codename string) {
	t.Helper()

	for _, issue := range issues {
		if issue.Codename() == codename {
			return
		}
	}

	require.Failf(t, "missing issue codename", "expected issue codename %q in %#v", codename, issues)
}
