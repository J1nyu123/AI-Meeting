package resume

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"
)

func TestLocalStorageRoundTripAndKeyValidation(t *testing.T) {
	storage := LocalStorage{Root: t.TempDir()}
	content := "%PDF-1.4\ntest resume"
	key, size, digest, err := storage.Save(strings.NewReader(content), "Candidate.PDF")
	require.NoError(t, err)
	require.Equal(t, int64(len(content)), size)
	sum := sha256.Sum256([]byte(content))
	require.Equal(t, hex.EncodeToString(sum[:]), digest)
	require.True(t, strings.HasSuffix(key, ".pdf"))

	file, err := storage.Open(key)
	require.NoError(t, err)
	data, err := io.ReadAll(file)
	require.NoError(t, err)
	require.Equal(t, content, string(data))
	require.NoError(t, file.Close())
	require.NoError(t, storage.Delete(key))
	_, err = storage.Open(key)
	require.ErrorIs(t, err, os.ErrNotExist)
	require.NoError(t, storage.Delete(key))

	_, err = storage.Open("../outside.pdf")
	require.Error(t, err)
	require.Error(t, storage.Delete("../outside.pdf"))
}

func TestTextPDFExtractorRejectsInvalidPDF(t *testing.T) {
	_, err := (TextPDFExtractor{}).Extract(filepath.Join(t.TempDir(), "missing.pdf"))
	require.ErrorContains(t, err, "PDF_OCR_REQUIRED")
}

func TestValidateExtractedTextRejectsUnreadableContent(t *testing.T) {
	require.ErrorContains(t, validateExtractedText("short"), "PDF_OCR_REQUIRED")
	require.ErrorContains(t, validateExtractedText(strings.Repeat("简历内容", 20)+"\uFFFD"), "PDF_OCR_REQUIRED")
	require.ErrorContains(t, validateExtractedText(strings.Repeat("resume", 20)+"\x06"), "PDF_OCR_REQUIRED")
	require.NoError(t, validateExtractedText(strings.Repeat("项目经历包含完整的工程实践和量化结果。", 5)))
	require.Equal(t, "第一页\n第二页", normalizeExtractedText([]byte("第一页\f第二页\r\n")))
}

func TestTextPDFExtractorExternalFixture(t *testing.T) {
	path := os.Getenv("AI_MEETING_PDF_FIXTURE")
	if path == "" {
		t.Skip("AI_MEETING_PDF_FIXTURE is not set")
	}

	text, err := (TextPDFExtractor{}).Extract(path)
	require.NoError(t, err)
	if expected := os.Getenv("AI_MEETING_PDF_EXPECTED_TEXT"); expected != "" {
		require.Contains(t, text, expected)
	}
	require.NotContains(t, text, "\uFFFD")
	for _, value := range text {
		require.False(t, unicode.IsControl(value) && value != '\n' && value != '\r' && value != '\t')
	}
}
