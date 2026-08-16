package media

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func intp(value int) *int { return &value }

func TestAssemblerAppendReplaceAndFinal(t *testing.T) {
	a := NewASTAssembler()
	s, changed := a.Apply(ASTSegment{ID: 1, Text: "你好", PGS: "apd", Finalized: true}, false)
	require.True(t, changed)
	require.Equal(t, "你好", s.CommittedText)
	s, _ = a.Apply(ASTSegment{ID: 2, Text: "世", PGS: "apd"}, false)
	require.Equal(t, "你好世", s.DisplayText)
	require.Equal(t, "世", s.LiveText)
	s, _ = a.Apply(ASTSegment{ID: 3, Text: "世界", PGS: "rpl", Range: []int{2, 2}}, true)
	require.Equal(t, "你好世界", s.DisplayText)
	require.Equal(t, s.DisplayText, s.CommittedText)
	require.Empty(t, s.LiveText)
}

func TestAssemblerNoPGSReusesOverlappingEvolutionAndPunctuation(t *testing.T) {
	a := NewASTAssembler()
	a.Apply(ASTSegment{ID: 1, Text: "我正在开发", Begin: intp(0), End: intp(100)}, false)
	s, _ := a.Apply(ASTSegment{ID: 2, Text: "我正在开发系统", Begin: intp(0), End: intp(120)}, false)
	require.Equal(t, "我正在开发系统", s.DisplayText)
	s, _ = a.Apply(ASTSegment{ID: 3, Text: "。", Begin: intp(0), End: intp(120)}, false)
	require.Equal(t, "我正在开发系统。", s.DisplayText)
	_, changed := a.Apply(ASTSegment{ID: 4, Text: "。", Begin: intp(0), End: intp(120)}, false)
	require.False(t, changed)
}

func TestAssemblerOrdersOutOfOrderSegments(t *testing.T) {
	a := NewASTAssembler()
	a.Apply(ASTSegment{ID: 2, Text: "二", PGS: "apd"}, false)
	s, _ := a.Apply(ASTSegment{ID: 1, Text: "一", PGS: "apd"}, false)
	require.Equal(t, "一二", s.DisplayText)
}
