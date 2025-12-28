package scum

// Action is a function triggered by a special symbol defined in the [Dictionary].
// It processes the input string starting from the index i, along with a previous char, and returns a [Token], byte stride and
// a boolean flag which tells if the returned token is empty.
type Action func(d *Dictionary, id byte, input string, i int, prevRune rune, warns *[]Warning) (token Token, stride int, skip bool)
