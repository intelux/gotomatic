package trigger

import "io"

type funcTrigger struct {
	f func(w io.Writer, name string, state bool) error
}

// TriggerFunc creates a trigger from a function.
func TriggerFunc(f func(io.Writer, string, bool) error) Trigger {
	return funcTrigger{f: f}
}

func (t funcTrigger) run(w io.Writer, name string, state bool) error {
	return t.f(w, name, state)
}