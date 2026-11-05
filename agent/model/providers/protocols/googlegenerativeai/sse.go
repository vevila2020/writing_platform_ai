package googlegenerativeai

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

// SSE relays occasionally separate events with mixed line endings or emit bare
// keep-alive lines between data frames. The genai SDK splits the body on the
// literal "\n\n" and "\r\n\r\n" separators and then requires each chunk to
// start with "data:", so a stray CRLF between events turns the next frame into
// an "invalid stream chunk" error. Rewrite the stream into canonical
// "data: {...}\n\n" frames before the SDK reads it.
func wrapSSEStreamingClient(client *http.Client) *http.Client {
	clone := *client
	if _, ok := clone.Transport.(sseNormalizeTransport); ok {
		return &clone
	}
	clone.Transport = sseNormalizeTransport{next: clone.Transport}
	return &clone
}

type sseNormalizeTransport struct{ next http.RoundTripper }

func (transport sseNormalizeTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	next := transport.next
	if next == nil {
		next = http.DefaultTransport
	}
	response, err := next.RoundTrip(request)
	if err != nil || response == nil || response.Body == nil {
		return response, err
	}
	if !isSSEResponse(response) {
		return response, nil
	}
	response.Body = newSSENormalizingBody(response.Body)
	response.ContentLength = -1
	response.Header.Del("Content-Length")
	return response, nil
}

func isSSEResponse(response *http.Response) bool {
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return false
	}
	return strings.Contains(strings.ToLower(response.Header.Get("Content-Type")), "text/event-stream")
}

func newSSENormalizingBody(source io.ReadCloser) io.ReadCloser {
	pipeReader, pipeWriter := io.Pipe()
	body := &sseNormalizingBody{PipeReader: pipeReader, source: source}
	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				_ = pipeWriter.CloseWithError(fmt.Errorf("google generative AI SSE normalize panic: %v", recovered))
			}
			_ = body.closeSource()
		}()
		reader := bufio.NewReader(source)
		var data bytes.Buffer
		flush := func() error {
			payload := bytes.TrimSpace(data.Bytes())
			data.Reset()
			if len(payload) == 0 {
				return nil
			}
			frame := make([]byte, 0, len(payload)+8)
			frame = append(frame, "data: "...)
			frame = append(frame, payload...)
			frame = append(frame, '\n', '\n')
			_, err := pipeWriter.Write(frame)
			return err
		}
		for {
			line, readErr := reader.ReadBytes('\n')
			if len(line) > 0 {
				line = bytes.TrimRight(line, "\r\n")
				line = bytes.TrimPrefix(line, []byte{0xEF, 0xBB, 0xBF})
				switch {
				case len(bytes.TrimSpace(line)) == 0:
					// A blank line dispatches the current event.
					if writeErr := flush(); writeErr != nil {
						_ = pipeWriter.CloseWithError(writeErr)
						return
					}
				case line[0] == ':':
					// SSE comment or keep-alive. The SDK cannot parse it.
				default:
					name, value, found := bytes.Cut(line, []byte{':'})
					if found && strings.EqualFold(string(name), "data") {
						value = bytes.TrimPrefix(value, []byte{' '})
						if data.Len() > 0 {
							data.WriteByte('\n')
						}
						_, _ = data.Write(value)
						continue
					}
					// Preserve a bare JSON frame (typically a mid-stream error)
					// so the SDK can surface it instead of silently dropping it.
					if line[0] == '{' {
						if writeErr := flush(); writeErr != nil {
							_ = pipeWriter.CloseWithError(writeErr)
							return
						}
						frame := append(append([]byte(nil), line...), '\n', '\n')
						if _, writeErr := pipeWriter.Write(frame); writeErr != nil {
							_ = pipeWriter.CloseWithError(writeErr)
							return
						}
					}
				}
			}
			if readErr != nil {
				if writeErr := flush(); writeErr != nil {
					_ = pipeWriter.CloseWithError(writeErr)
					return
				}
				if readErr == io.EOF {
					_ = pipeWriter.Close()
					return
				}
				_ = pipeWriter.CloseWithError(readErr)
				return
			}
		}
	}()
	return body
}

type sseNormalizingBody struct {
	*io.PipeReader
	source    io.Closer
	closeOnce sync.Once
	closeErr  error
}

func (b *sseNormalizingBody) Close() error {
	pipeErr := b.PipeReader.Close()
	sourceErr := b.closeSource()
	if pipeErr != nil {
		return pipeErr
	}
	return sourceErr
}

func (b *sseNormalizingBody) closeSource() error {
	b.closeOnce.Do(func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				b.closeErr = fmt.Errorf("google generative AI SSE source close panic: %v", recovered)
			}
		}()
		b.closeErr = b.source.Close()
	})
	return b.closeErr
}
