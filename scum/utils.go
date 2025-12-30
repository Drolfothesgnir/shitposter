package scum

import (
	"errors"
	"fmt"
	"unicode/utf8"
)

// isASCIIPrintable returns true if the byte is a printable ASCII character, that is
// its value is between 32 and 126.
func isASCIIPrintable(b byte) bool {
	// Printable ASCII characters are in the range 32 (space) to 126 (~)
	return b >= 32 && b <= 126
}

// checkByteDifference compares substr against the beginning of seq.
// It returns the index of the first differing byte, or -1 if no difference is found.
// substrShorter is true if substr is a prefix of seq but is shorter in length.
func checkByteDifference(substr string, seq []byte) (diffIndex int, substrShorter bool) {
	lenSubstr := len(substr)
	lenSeq := len(seq)

	diffIndex = -1
	substrShorter = lenSubstr < lenSeq

	minLen := min(lenSubstr, lenSeq)

	for i := range minLen {
		if substr[i] != seq[i] {
			diffIndex = i
			return
		}
	}

	return
}

// extractNextRune returns the first value (either simple ASCII or an UTF-8 code point) of the non-empty substr.
// It also returns the byte count of the found char and a bool flag, which is false in case the char is
// not a valid UTF-8 code point, but an [utf8.RuneError].
//
// WARNING: [utf8.DecodeRuneInString] returns width 0 if the decoded char is erroneous.
func extractNextRune(substr string) (next rune, width int, ok bool) {
	b := substr[0]

	// check if the first byte is simple ASCII
	if b < 128 {
		return rune(b), 1, true
	}

	// else we must decode the code point
	next, width = utf8.DecodeRuneInString(substr)
	ok = next != utf8.RuneError
	return
}

// checkTagName checks the provided [Tag.Name] for being valid name and returns [ConfigError] if any issues occur.
func checkTagName(name string) error {
	// check if the name is not empty
	if name == "" {
		return NewConfigError(IssueInvalidTagNameLen, errors.New("tag's name is empty"))
	}

	// check if the name contains no more code points than [MaxTagNameLen]
	nameLen := utf8.RuneCountInString(name)

	if nameLen > MaxTagNameLen {
		return NewConfigError(IssueInvalidTagNameLen,
			fmt.Errorf("tag's name can be at most %d characters, but got %d", MaxTagNameLen, nameLen))
	}

	return nil
}

// chechTagConsistency checks if rules and greed values are consistent with Tag's other config and
// returns [ConfigError] if any issues.
func checkTagConsistency(isSingle, isUniversal bool, rule Rule, greed Greed) error {
	// validate enums
	if rule > MaxRule {
		return NewConfigError(IssueInvalidRule,
			fmt.Errorf("rule can have values up to %d but got %d instead", MaxRule, rule))
	}
	if greed > MaxGreedLevel {
		return NewConfigError(IssueInvalidGreedLevel,
			fmt.Errorf("greed level can be at most %d, but got %d instead", MaxGreedLevel, greed))
	}

	// rules are only for single-char universal tags
	if rule != RuleNA && !(isSingle && isUniversal) {
		return NewConfigError(IssueRuleInapplicable,
			fmt.Errorf("rule %d is applicable only to single-char universal tags", rule))
	}

	// rule/greed compatibility
	switch rule {
	case RuleNA:
		return nil

	case RuleInfraWord:
		if greed != NonGreedy {
			return NewConfigError(IssueInvalidRule,
				fmt.Errorf("rule %d (intra-word) requires greed=%d (NonGreedy), got %d", rule, NonGreedy, greed))
		}
		return nil

	case RuleTagVsContent:
		if greed == NonGreedy {
			return NewConfigError(IssueInvalidRule,
				fmt.Errorf("rule %d (tag-vs-content) requires greedy tag, got greed=%d (NonGreedy)", rule, greed))
		}
		return nil

	default:
		// unreachable because of rule > MaxRule check, but keeps switch future-proof
		return NewConfigError(IssueInvalidRule,
			fmt.Errorf("unknown rule %d", rule))
	}
}

// isASCIIPunct return true if b is one of these symbols:
// !  "  #  $  %  &  '  (  )  *  +  ,  -  .  /
// :  ;  <  =  >  ?  @
// [  \  ]  ^  _  `
// {  |  }  ~
func isASCIIPunct(b byte) bool {
	return (33 <= b && b <= 47) ||
		(58 <= b && b <= 64) ||
		(91 <= b && b <= 96) ||
		(123 <= b && b <= 126)
}

// isASCIIAlphanum return true if b is one of these symbols:
// 0 1 2 3 4 5 6 7 8 9
// a b c d e f g h i j k l m
// n o p q r s t u v w x y z
// A B C D E F G H I J K L M
// N O P Q R S T U V W X Y Z
func isASCIIAlphanum(b byte) bool {
	return (b >= '0' && b <= '9') ||
		(b >= 'A' && b <= 'Z') ||
		(b >= 'a' && b <= 'z')
}
