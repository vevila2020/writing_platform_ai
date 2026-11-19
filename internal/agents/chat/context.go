package chat

import (
	"github.com/vevila2020/writing_platform_ai/internal/book"
)

// readReferencedFile reads one explicit workspace reference without applying a
// second, caller-local budget. The shared context assembler is the sole owner
// of UTF-8-safe truncation and model-visible truncation notices.
func readReferencedFile(bookService *book.Service, relPath string) (string, error) {
	content, err := bookService.ReadFile(relPath)
	if err != nil {
		return "", err
	}
	return content, nil
}
