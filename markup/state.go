package markup

// state represents NaiveParser states during iterations of the main loop.
type state struct {

	// crumbs contains chain of the nodes in the current branch
	// with possibility of backtracking.
	crumbs stack[Node]

	// stack defines a chain of open tags to check correctness of
	// opening/closing tag syntax.
	stack stack[Type]
}

func (s *state) addCrumb(n Node) {
	s.crumbs.push(n)
}

func (s *state) backtrack(steps int) {
	n := len(s.crumbs.v)
	if steps < n {
		s.crumbs.v = s.crumbs.v[:n-steps]
	}
}

func (s *state) peekCrumb() (Node, bool) {
	return s.crumbs.peek()
}

func (s *state) addOpeningType(t Type) {
	s.stack.push(t)
}

func (s *state) removeOpeningType() {
	s.stack.pop()
}

func (s *state) peekOpeningType() (Type, bool) {
	return s.stack.peek()
}

type stack[T any] struct {
	v []T
}

func (s *stack[T]) push(t T) {
	s.v = append(s.v, t)
}

func (s *stack[T]) pop() {
	if len(s.v) > 0 {
		s.v = s.v[:len(s.v)-1]
	}
}

func (s *stack[T]) peek() (T, bool) {
	if len(s.v) > 0 {
		return s.v[len(s.v)-1], true
	}

	var empty T

	return empty, false
}

func newStack[T any]() *stack[T] {
	return &stack[T]{}
}
