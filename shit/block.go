package shit

type Block interface {
	Type() string
	Render() // what to return?
}
