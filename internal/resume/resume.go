package resume

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

type Asset struct {
	ID            string `gorm:"primaryKey;size:36"`
	SessionID     string
	OriginalName  string
	StorageKey    string
	ContentType   string
	SizeBytes     int64
	SHA256        string
	ExtractedText string `gorm:"type:mediumtext"`
	CreatedAt     time.Time
}

func (Asset) TableName() string { return "resume_assets" }

type FileStorage interface {
	Save(io.Reader, string) (string, int64, string, error)
	Open(string) (io.ReadCloser, error)
	Delete(string) error
}
type LocalStorage struct{ Root string }

func (s LocalStorage) Save(src io.Reader, name string) (string, int64, string, error) {
	if err := os.MkdirAll(s.Root, 0750); err != nil {
		return "", 0, "", err
	}
	key := uuid.NewString() + strings.ToLower(filepath.Ext(name))
	path := filepath.Join(s.Root, key)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0640)
	if err != nil {
		return "", 0, "", err
	}
	h := sha256.New()
	n, copyErr := io.Copy(io.MultiWriter(f, h), src)
	closeErr := f.Close()
	if copyErr != nil {
		return "", 0, "", copyErr
	}
	if closeErr != nil {
		return "", 0, "", closeErr
	}
	return key, n, hex.EncodeToString(h.Sum(nil)), nil
}
func (s LocalStorage) Open(key string) (io.ReadCloser, error) {
	clean := filepath.Base(key)
	if clean != key {
		return nil, fmt.Errorf("invalid storage key")
	}
	return os.Open(filepath.Join(s.Root, clean))
}
func (s LocalStorage) Delete(key string) error {
	clean := filepath.Base(key)
	if clean != key {
		return fmt.Errorf("invalid storage key")
	}
	err := os.Remove(filepath.Join(s.Root, clean))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

type PDFExtractor interface {
	Extract(path string) (string, error)
}
type TextPDFExtractor struct{}

func (TextPDFExtractor) Extract(path string) (string, error) {
	output, err := exec.Command("pdftotext", "-layout", "-enc", "UTF-8", path, "-").Output()
	if err != nil {
		return "", fmt.Errorf("PDF_OCR_REQUIRED: pdftotext failed: %w", err)
	}
	text := normalizeExtractedText(output)
	if err := validateExtractedText(text); err != nil {
		return "", err
	}
	return text, nil
}

func normalizeExtractedText(output []byte) string {
	text := strings.ReplaceAll(string(output), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\f", "\n")
	return strings.TrimSpace(text)
}

func validateExtractedText(text string) error {
	if !utf8.ValidString(text) || len([]rune(text)) < 50 {
		return fmt.Errorf("PDF_OCR_REQUIRED: extracted text is empty or invalid UTF-8")
	}
	for _, value := range text {
		if value == unicode.ReplacementChar || (unicode.IsControl(value) && value != '\n' && value != '\r' && value != '\t') {
			return fmt.Errorf("PDF_OCR_REQUIRED: extracted text contains invalid characters")
		}
	}
	return nil
}
