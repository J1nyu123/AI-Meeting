package evaluation

type Result struct {
	Score            int      `json:"score"`
	Feedback         string   `json:"feedback"`
	MissingPoints    []string `json:"missing_points"`
	FollowUpNeeded   bool     `json:"follow_up_needed"`
	FollowUpQuestion string   `json:"follow_up_question"`
}
type Decision struct {
	NeedFollowUp bool
	Reason       string
}

func Decide(result Result, current, maximum int) Decision {
	if current >= maximum {
		return Decision{Reason: "limit_reached"}
	}
	if result.FollowUpNeeded {
		return Decision{true, "ai_suggestion"}
	}
	if result.Score < 60 {
		return Decision{true, "low_score"}
	}
	if len(result.MissingPoints) > 0 {
		return Decision{true, "missing_points"}
	}
	return Decision{Reason: "not_needed"}
}
