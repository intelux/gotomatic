package conditional

import "time"

type delayedCondition struct {
	Condition
	Delay        time.Duration
	subcondition Condition
	done         chan struct{}
}

// Delay returns a Condition whose state changes are reflected if they change
// at least for the specified duration. The initial state of the passed-in
// condition is copied without delay.
func Delay(condition Condition, delay time.Duration) Condition {
	state, channel := condition.GetAndWaitChange()
	c := &delayedCondition{
		Condition:    NewManualCondition(state),
		Delay:        delay,
		subcondition: condition,
		done:         make(chan struct{}),
	}

	go c.waitChange(state, channel)

	return c
}

// Close terminates the condition.
//
// Any pending wait on one of the returned channels via Wait() or
// WaitChange() will be unblocked.
//
// Calling Close() twice or more has no effect.
func (condition *delayedCondition) Close() error {
	if condition.done != nil {
		close(condition.done)
		condition.done = nil
	}

	condition.subcondition.Close()
	return condition.Condition.Close()
}

type timer interface {
	Wait() <-chan time.Time
	Stop()
}

type realTimer struct {
	timer *time.Timer
}

func (t realTimer) Wait() <-chan time.Time {
	return t.timer.C
}

func (t realTimer) Stop() {
	t.timer.Stop()
}

type foreverTimer struct {
	channel chan time.Time
}

func (t foreverTimer) Wait() <-chan time.Time {
	return t.channel
}

func (t foreverTimer) Stop() {
	close(t.channel)
}

func (condition delayedCondition) waitChange(state bool, channel <-chan error) {
	var timer timer = foreverTimer{
		channel: make(chan time.Time),
	}

	for {
		select {
		case <-condition.done:
			// The condition was closed.
			timer.Stop()
			return
		case <-channel:
			// The underlying condition changed, let's rewait and start a timer.
			state, channel = condition.subcondition.GetAndWaitChange()
			timer.Stop()
			timer = realTimer{timer: time.NewTimer(condition.Delay)}
		case <-timer.Wait():
			// The timer expired. Let's apply the last recovered state.
			condition.Condition.(*ManualCondition).Set(state)
			timer = foreverTimer{
				channel: make(chan time.Time),
			}
		}
	}
}
