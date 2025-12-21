package markup

// Issue describes the kind of problem detected during parsing,
// e.g. unclosed tag, mis-nested tag, redundant escape, malformed link, etc.
type Issue int

const (
	IssueUnclosedTag Issue = iota
	IssueMisNestedTag
	IssueRedundantEscape
	IssueMalformedLink
	IssueUnexpectedEOL
	IssueUnexpectedSymbol
)
