package scum

// Action is a function triggered by a special symbol defined in the [Dictionary].
// It processes the input string starting from the index i and returns a [Token], byte stride and
// a boolean flag which tells if the returned token is empty.
// WARNING: an Action MUST always return a stride > 0, even when skip = true.
type Action func(d *Dictionary, id byte, input string, i int, warns *[]Warning) (token Token, stride int, skip bool)
