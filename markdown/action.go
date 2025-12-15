package markdown

// action defines a function used to process the substring after the
// corresponding special symbol occured.
//
// Recieves a part of the string, starting from the index of the special symbol - `substr`
// current rune - `cur`
// width of the current rune in bytes - `width`
// index of the occurance in the original string - `i`
// and indicator if the occured rune was last in the sequence - `isLastRune`.
//
// Return values:
//   - token Token - Token created as the result of the procces. Can be empty, 'ok' will be true if it's not.
//   - warnings []Warning - List of warnings occured during the process. Can be empty.
//   - stride int - number of bytes processed by the process, to adjust the pointer offset in the main loop.
//   - ok bool - indicates if the Token returned is not empty.
type action func(substr string, cur rune, width int, i int, isLastRune bool) (token Token, warnings []Warning, stride int, ok bool)
