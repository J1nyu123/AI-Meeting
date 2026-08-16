package interview

import "fmt"

func CanTransition(from, to Status) bool {
	switch from {
	case StatusCreated:
		return to == StatusAnalyzing
	case StatusAnalyzing:
		return to == StatusReady || to == StatusFailed
	case StatusReady:
		return to == StatusInProgress || to == StatusCompleted
	case StatusInProgress:
		return to == StatusInProgress || to == StatusCompleted
	default:
		return false
	}
}
func Transition(session *Session, to Status) error {
	if !CanTransition(session.Status, to) {
		return fmt.Errorf("invalid interview transition %s -> %s", session.Status, to)
	}
	session.Status = to
	return nil
}
