package sml

import (
	"fmt"

	"github.com/Drolfothesgnir/shitposter/scum"
)

type SyntaxIssue interface {
	fmt.Stringer
	Code() int
	Codename() string
	Description() string
}

type Issue int

const (
	issueCodeBase Issue = 2000
)

const (
	IssueInternal Issue = issueCodeBase + iota
	IssueConfig
	IssueUnknownNodeType
	IssueUnknownTag
	IssueAttributeNotAllowed
	IssueAttributeInvalidPayload

	maxIssueCode
)

const maxIssueNum = int(maxIssueCode - issueCodeBase)

var mapIssueToStr [maxIssueNum]string

func issueIndex(issue Issue) int {
	return int(issue - issueCodeBase)
}

func init() {
	mapIssueToStr[issueIndex(IssueInternal)] = "INTERNAL"
	mapIssueToStr[issueIndex(IssueConfig)] = "CONFIG"
	mapIssueToStr[issueIndex(IssueUnknownNodeType)] = "UNKNOWN_NODE_TYPE"
	mapIssueToStr[issueIndex(IssueUnknownTag)] = "UNKNOWN_TAG"
	mapIssueToStr[issueIndex(IssueAttributeNotAllowed)] = "ATTRIBUTE_NOT_ALLOWED"
	mapIssueToStr[issueIndex(IssueAttributeInvalidPayload)] = "ATTRIBUTE_INVALID_PAYLOAD"
}

type SyntaxIssueDescriptor struct {
	C     Issue  `json:"code"`
	CName string `json:"codename"`
	// Description is a human-readable description of the issue.
	Desc string `json:"description"`
}

func (i SyntaxIssueDescriptor) String() string {
	return fmt.Sprintf("SML: Codename - %s; %s", i.CName, i.Desc)
}

func (i SyntaxIssueDescriptor) Code() int {
	return int(i.C)
}

func (i SyntaxIssueDescriptor) Codename() string {
	return i.CName
}

func (i SyntaxIssueDescriptor) Description() string {
	return i.Desc
}

func NewSyntaxIssueDescriptor(code Issue, desc string) SyntaxIssueDescriptor {
	// i don't want to return any errors because i think returning errors from the error factory
	// is stupid. What do you think?
	if code < issueCodeBase || code >= maxIssueCode {
		code = IssueInternal
	}
	return SyntaxIssueDescriptor{
		C:     code,
		CName: mapIssueToStr[issueIndex(code)],
		Desc:  desc,
	}
}

type Warning struct {
	scum.SerializableWarning
}

func (w Warning) String() string {
	return fmt.Sprintf("SML - Parser: Codename - %s; %s", w.SerializableWarning.Codename, w.SerializableWarning.Description)
}

func (w Warning) Code() int {
	return int(w.SerializableWarning.Code)
}

func (w Warning) Codename() string {
	return w.SerializableWarning.Codename
}

func (w Warning) Description() string {
	return w.SerializableWarning.Description
}

type Issues struct {
	List []SyntaxIssue `json:"issues"`
}

func (i *Issues) Add(d SyntaxIssue) {
	i.List = append(i.List, d)
}

func NewIssues(cap int) Issues {
	l := make([]SyntaxIssue, 0, cap)
	return Issues{l}
}
