package scum

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLongestCommonSubPrefix(t *testing.T) {
	testCases := []struct {
		name              string
		src               string
		seq               []byte
		expectedContained bool
		expectedStartIdx  int
		expectedWidth     int
	}{
		{
			name:              "3Attempts_OK",
			src:               "$hey  $$you $$$ok $$hello",
			seq:               []byte{'$', '$', '$'},
			expectedContained: true,
			expectedStartIdx:  12,
			expectedWidth:     3,
		},
		{
			name:              "1Attempt_Not_Found",
			src:               "$hello world$$ !",
			seq:               []byte{'$', '$', '$'},
			expectedContained: false,
			expectedStartIdx:  12,
			expectedWidth:     2,
		},
		{
			name:              "LastInString_Not_Found",
			src:               "hello world<=",
			seq:               []byte{'<', '=', '3'},
			expectedContained: false,
			expectedStartIdx:  11,
			expectedWidth:     2,
		},
		{
			name:              "InputLongerThanSequence",
			src:               "A",
			seq:               []byte("ABC"),
			expectedContained: false,
			expectedStartIdx:  0,
			expectedWidth:     1,
		},
		{
			name:              "EmptySearchSeq",
			src:               "A",
			seq:               []byte(""),
			expectedContained: false,
			expectedStartIdx:  -1,
			expectedWidth:     0,
		},
		{
			name:              "Exact Match",
			src:               "hello world",
			seq:               []byte("world"),
			expectedContained: true,
			expectedStartIdx:  6,
			expectedWidth:     5,
		},
		{
			name:              "Partial Match (Longest)",
			src:               "I like sho-making",
			seq:               []byte("shout"),
			expectedContained: false,
			expectedStartIdx:  7,
			expectedWidth:     3,
		},
		{
			name:              "Overlapping Starts",
			src:               "AAAB",
			seq:               []byte("AAB"),
			expectedContained: true,
			expectedStartIdx:  1,
			expectedWidth:     3,
		},
		{
			name:              "Multiple Partial Matches (Pick Best)",
			src:               "ab-abc-abcd",
			seq:               []byte("abcde"),
			expectedContained: false,
			expectedStartIdx:  7,
			expectedWidth:     4,
		},
		{
			name:              "Sequence Longer Than Source",
			src:               "Go",
			seq:               []byte("Gopher"),
			expectedContained: false,
			expectedStartIdx:  0,
			expectedWidth:     2,
		},
		{
			name:              "No Match At All",
			src:               "abcdef",
			seq:               []byte("xyz"),
			expectedContained: false,
			expectedStartIdx:  -1,
			expectedWidth:     0,
		},
		{
			name:              "Partial match cut off by EOF",
			src:               "hello",
			seq:               []byte("hello world"),
			expectedContained: false,
			expectedStartIdx:  0,
			expectedWidth:     5,
		},
		{
			name:              "SingleCharSeqOK",
			src:               "hello world",
			seq:               []byte("h"),
			expectedContained: true,
			expectedStartIdx:  0,
			expectedWidth:     1,
		},
		{
			name:              "SingleCharSeqFail",
			src:               "hello world",
			seq:               []byte("A"),
			expectedContained: false,
			expectedStartIdx:  -1,
			expectedWidth:     0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, i, w := longestCommonSubPrefix(tc.src, tc.seq)
			require.Equal(t, tc.expectedContained, c)
			require.Equal(t, tc.expectedStartIdx, i)
			require.Equal(t, tc.expectedWidth, w)
		})
	}
}
