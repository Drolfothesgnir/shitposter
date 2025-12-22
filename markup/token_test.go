package markup

import (
	"testing"
)

func TestTokenizeCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType Type
		wantVal  string
		wantWarn bool
	}{
		{
			name:     "Simple inline code",
			input:    "`code`",
			wantType: TypeCodeInline,
			wantVal:  "`code`",
		},
		{
			name:     "N+1 rule: ignore single backtick in double",
			input:    "``a ` b``",
			wantType: TypeCodeInline,
			wantVal:  "``a ` b``",
		},
		{
			name:     "Code block with 3 backticks",
			input:    "```block```",
			wantType: TypeCodeBlock,
			wantVal:  "```block```",
		},
		{
			name:     "Malformed: only backticks",
			input:    "```",
			wantType: TypeText,
			wantVal:  "```",
			wantWarn: true,
		},
		{
			name:     "Unclosed inline treated as text",
			input:    "`unclosed",
			wantType: TypeText,
			wantVal:  "`",
			wantWarn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, warns := Tokenize(tt.input)

			// We check the first token for simplicity
			if tokens[0].Type != tt.wantType {
				t.Errorf("got type %v, want %v", tokens[0].Type, tt.wantType)
			}
			if tokens[0].Val != tt.wantVal {
				t.Errorf("got val %q, want %q", tokens[0].Val, tt.wantVal)
			}
			if (len(warns) > 0) != tt.wantWarn {
				t.Errorf("got warns %v, want %v", len(warns) > 0, tt.wantWarn)
			}
		})
	}
}
