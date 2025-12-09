package content

// Schema is an implementation agnostic post-body content parser interface.
//
// The goal is to be able to maintain different body schemas.
type Schema interface {
	// Name returns the name of the particular body schema.
	Name() string

	// Version returns the version number of the particular body schema.
	Version() int32

	// Parse tries to transform raw bytes into the particular body schema.
	//
	// Parse validates + normalizes raw JSON.
	// Returns canonical, sanitized JSON for storage.
	Parse(raw []byte) ([]byte, error)
}
