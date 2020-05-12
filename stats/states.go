package stats

type state struct {
	id       int
	name     string
	dataType int
}
type activeStates map[int]state
