package scum

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func mustNewWarnings(t *testing.T, policy WarningOverflowPolicy, cap int) Warnings {
	t.Helper()
	w, err := NewWarnings(policy, cap)
	require.NoError(t, err)
	return w
}

func newWarn(pos int) Warning {
	return Warning{
		Issue: IssueUnexpectedEOL,
		Pos:   pos,
	}
}

func TestNewWarnings_NegativeCap_ReturnsConfigError(t *testing.T) {
	_, err := NewWarnings(WarnOverflowDrop, -1)
	require.Error(t, err)

	var ce *ConfigError
	require.True(t, errors.As(err, &ce), "expected *ConfigError, got %T (%v)", err, err)
	require.Equal(t, IssueNegativeWarningsCap, ce.Issue)
}

func TestNewWarnings_CapZero_DoesNotStartOverflow(t *testing.T) {
	w := mustNewWarnings(t, WarnOverflowTrunc, 0)

	require.False(t, w.IsOverflow(), "overflow must be false until at least one warning is dropped")
	require.Equal(t, 0, w.DroppedCount())
	require.Equal(t, 0, w.FirstDropPos())
	require.Len(t, w.List(), 0)
}

func TestWarnings_NoRec_NoOp(t *testing.T) {
	w := mustNewWarnings(t, WarnOverflowNoRec, 3)

	w.Add(newWarn(0))
	w.Add(newWarn(1))

	require.False(t, w.IsOverflow())
	require.Equal(t, 0, w.DroppedCount())
	require.Equal(t, 0, w.FirstDropPos())
	require.Len(t, w.List(), 0)
}

func TestWarnings_NoCap_IgnoresCap(t *testing.T) {
	w := mustNewWarnings(t, WarnOverflowNoCap, 2)

	for i := 0; i < 10; i++ {
		w.Add(newWarn(i))
	}

	require.False(t, w.IsOverflow())
	require.Equal(t, 0, w.DroppedCount())
	require.Len(t, w.List(), 10)
	require.Equal(t, 0, w.List()[0].Pos)
	require.Equal(t, 9, w.List()[9].Pos)
}

func TestWarnings_Drop_KeepsFirstN_ThenDiscards(t *testing.T) {
	w := mustNewWarnings(t, WarnOverflowDrop, 3)

	w.Add(newWarn(0))
	w.Add(newWarn(1))
	w.Add(newWarn(2))
	w.Add(newWarn(3))
	w.Add(newWarn(4))

	require.True(t, w.IsOverflow())
	require.Equal(t, 0, w.DroppedCount(), "Drop policy should not count dropped warnings")
	require.Equal(t, 3, w.FirstDropPos(), "first dropped warning should define FirstDropPos")
	require.Len(t, w.List(), 3)
	require.Equal(t, 0, w.List()[0].Pos)
	require.Equal(t, 2, w.List()[2].Pos)
}

func TestWarnings_Trunc_ReservesSlotForMarker(t *testing.T) {
	w := mustNewWarnings(t, WarnOverflowTrunc, 3)

	// With cap=3 and Trunc policy, we keep 2 real warnings + 1 truncation marker.
	w.Add(newWarn(0))
	w.Add(newWarn(1))

	// These will be dropped (including this first dropped one),
	// and the truncation marker should appear as the last element.
	w.Add(newWarn(2))
	w.Add(newWarn(3))
	w.Add(newWarn(4))

	require.True(t, w.IsOverflow())
	require.Equal(t, 3, w.DroppedCount(), "expected dropped warnings: pos 2,3,4")
	require.Equal(t, 2, w.FirstDropPos())

	require.Len(t, w.List(), 3)
	require.Equal(t, 0, w.List()[0].Pos)
	require.Equal(t, 1, w.List()[1].Pos)

	last := w.List()[2]
	require.Equal(t, IssueWarningsTruncated, last.Issue)
	require.Equal(t, 2, last.Pos)
}

func TestWarnings_Trunc_Cap1_ListContainsOnlyMarker(t *testing.T) {
	w := mustNewWarnings(t, WarnOverflowTrunc, 1)

	w.Add(newWarn(10))
	w.Add(newWarn(11))

	require.True(t, w.IsOverflow())
	require.Equal(t, 2, w.DroppedCount())
	require.Equal(t, 10, w.FirstDropPos())

	require.Len(t, w.List(), 1)
	require.Equal(t, IssueWarningsTruncated, w.List()[0].Issue)
	require.Equal(t, 10, w.List()[0].Pos)
}

func TestWarnings_Trunc_Cap0_StoresNothingButOverflowsOnFirstAdd(t *testing.T) {
	w := mustNewWarnings(t, WarnOverflowTrunc, 0)

	w.Add(newWarn(7))
	w.Add(newWarn(8))

	require.True(t, w.IsOverflow())
	require.Equal(t, 2, w.DroppedCount())
	require.Equal(t, 7, w.FirstDropPos())
	require.Len(t, w.List(), 0, "cap=0 must store nothing, even truncation marker")
}

func TestWarnings_Drop_Cap0_OverflowsOnFirstAdd(t *testing.T) {
	w := mustNewWarnings(t, WarnOverflowDrop, 0)

	w.Add(newWarn(5))
	w.Add(newWarn(6))

	require.True(t, w.IsOverflow())
	require.Len(t, w.List(), 0)
	require.Equal(t, 0, w.DroppedCount(), "Drop policy should not count dropped warnings")
	require.Equal(t, 5, w.FirstDropPos())
}
