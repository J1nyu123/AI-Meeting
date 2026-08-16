package evaluation

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestDecide(t *testing.T) {
	tests := []struct {
		name    string
		result  Result
		current int
		want    Decision
	}{
		{"limit", Result{Score: 50, FollowUpNeeded: true}, 2, Decision{Reason: "limit_reached"}},
		{"ai suggestion", Result{Score: 90, FollowUpNeeded: true}, 0, Decision{true, "ai_suggestion"}},
		{"low score", Result{Score: 50}, 0, Decision{true, "low_score"}},
		{"missing points", Result{Score: 90, MissingPoints: []string{"量化结果"}}, 0, Decision{true, "missing_points"}},
		{"not needed", Result{Score: 90}, 0, Decision{Reason: "not_needed"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, Decide(tt.result, tt.current, 2))
		})
	}
}
