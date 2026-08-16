package interview

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestStateMachine(t *testing.T) {
	s := Session{Status: StatusCreated}
	require.NoError(t, Transition(&s, StatusAnalyzing))
	require.NoError(t, Transition(&s, StatusReady))
	require.NoError(t, Transition(&s, StatusInProgress))
	require.NoError(t, Transition(&s, StatusInProgress))
	require.NoError(t, Transition(&s, StatusCompleted))
	require.Error(t, Transition(&s, StatusInProgress))
}
