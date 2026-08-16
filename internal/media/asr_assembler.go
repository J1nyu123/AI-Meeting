package media

import (
	"sort"
	"strings"
	"unicode"
)

const (
	maxASRSegments = 2048
	maxASRRunes    = 100000
)

type ASTSegment struct {
	ID        int
	Text      string
	PGS       string
	Range     []int
	Begin     *int
	End       *int
	Finalized bool
}

type TranscriptionSnapshot struct {
	DisplayText   string `json:"displayText"`
	CommittedText string `json:"committedText"`
	LiveText      string `json:"liveText"`
	Revision      int64  `json:"revision"`
}

type segmentState struct {
	ASTSegment
}

type ASTAssembler struct {
	segments map[int]*segmentState
	revision int64
	last     string
}

func NewASTAssembler() *ASTAssembler { return &ASTAssembler{segments: make(map[int]*segmentState)} }

func (a *ASTAssembler) Apply(next ASTSegment, finalPacket bool) (TranscriptionSnapshot, bool) {
	next.Text = strings.TrimSpace(next.Text)
	if next.Text == "" {
		return TranscriptionSnapshot{}, false
	}
	if next.PGS == "rpl" && len(next.Range) >= 2 {
		start, end := next.Range[0], next.Range[1]
		if start > end {
			start, end = end, start
		}
		for id := start; id <= end; id++ {
			delete(a.segments, id)
		}
	}
	if next.PGS == "" && next.Begin != nil && next.End != nil {
		if a.applyWithoutPGS(next) {
			return a.snapshot(finalPacket)
		}
	}
	a.upsert(next)
	return a.snapshot(finalPacket)
}

func (a *ASTAssembler) applyWithoutPGS(next ASTSegment) bool {
	for _, current := range a.segments {
		if current.Begin == nil || current.End == nil {
			continue
		}
		if *current.Begin == *next.Begin && *current.End == *next.End && punctuationOnly(next.Text) {
			current.Text = appendPunctuation(current.Text, next.Text)
			current.Finalized = current.Finalized || next.Finalized
			return true
		}
	}
	var best *segmentState
	bestScore := -1.0
	for _, current := range a.segments {
		if current.Begin == nil || current.End == nil {
			continue
		}
		ratio := overlapRatio(*next.Begin, *next.End, *current.Begin, *current.End)
		if ratio < .6 || !sameEvolution(current.Text, next.Text) {
			continue
		}
		if ratio > bestScore {
			best, bestScore = current, ratio
		}
	}
	if best == nil {
		return false
	}
	best.Text = mergeEvolvingText(best.Text, next.Text)
	best.Begin, best.End = next.Begin, next.End
	best.Finalized = best.Finalized || next.Finalized
	for id, current := range a.segments {
		if current == best || current.Begin == nil || current.End == nil {
			continue
		}
		if *current.Begin >= *next.Begin && *current.End <= *next.End && sameEvolution(current.Text, next.Text) {
			delete(a.segments, id)
		}
	}
	return true
}

func (a *ASTAssembler) upsert(next ASTSegment) {
	if existing := a.segments[next.ID]; existing != nil {
		next.Text = mergeEvolvingText(existing.Text, next.Text)
		next.Finalized = existing.Finalized || next.Finalized
	}
	if len(a.segments) >= maxASRSegments && a.segments[next.ID] == nil {
		return
	}
	a.segments[next.ID] = &segmentState{ASTSegment: next}
}

func (a *ASTAssembler) snapshot(finalPacket bool) (TranscriptionSnapshot, bool) {
	ids := make([]int, 0, len(a.segments))
	for id := range a.segments {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	var display, committed strings.Builder
	for _, id := range ids {
		state := a.segments[id]
		display.WriteString(state.Text)
		if finalPacket || state.Finalized {
			committed.WriteString(state.Text)
		}
		if display.Len() > maxASRRunes*4 {
			break
		}
	}
	displayText := truncateRunes(display.String(), maxASRRunes)
	if displayText == a.last {
		return TranscriptionSnapshot{}, false
	}
	committedText := truncateRunes(committed.String(), maxASRRunes)
	liveText := strings.TrimPrefix(displayText, committedText)
	if finalPacket {
		committedText, liveText = displayText, ""
	}
	a.last = displayText
	a.revision++
	return TranscriptionSnapshot{DisplayText: displayText, CommittedText: committedText, LiveText: liveText, Revision: a.revision}, true
}

func overlapRatio(a1, a2, b1, b2 int) float64 {
	if a1 > a2 {
		a1, a2 = a2, a1
	}
	if b1 > b2 {
		b1, b2 = b2, b1
	}
	start, end := maxInt(a1, b1), minInt(a2, b2)
	if end < start {
		return 0
	}
	intersection := end - start + 1
	denom := minInt(a2-a1+1, b2-b1+1)
	if denom <= 0 {
		return 0
	}
	return float64(intersection) / float64(denom)
}

func comparable(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, value)
}
func sameEvolution(a, b string) bool {
	a, b = comparable(a), comparable(b)
	if a == "" || b == "" {
		return false
	}
	if a == b || strings.Contains(a, b) || strings.Contains(b, a) {
		return true
	}
	limit := minInt(len([]rune(a)), len([]rune(b)))
	common := 0
	ar, br := []rune(a), []rune(b)
	for common < limit && ar[common] == br[common] {
		common++
	}
	return float64(common)/float64(limit) >= .8
}
func mergeEvolvingText(old, next string) string {
	if old == next || strings.Contains(old, next) {
		return old
	}
	if strings.Contains(next, old) {
		return next
	}
	for size := minInt(len([]rune(old)), len([]rune(next))); size > 0; size-- {
		o, n := []rune(old), []rune(next)
		if string(o[len(o)-size:]) == string(n[:size]) {
			return old + string(n[size:])
		}
	}
	return next
}
func punctuationOnly(value string) bool {
	seen := false
	for _, r := range strings.TrimSpace(value) {
		seen = true
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return false
		}
	}
	return seen
}
func appendPunctuation(base, suffix string) string {
	if strings.HasSuffix(base, suffix) {
		return base
	}
	return base + suffix
}
func truncateRunes(value string, limit int) string {
	r := []rune(value)
	if len(r) > limit {
		return string(r[:limit])
	}
	return value
}
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
