package googlegenerativeai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/vevila2020/writing_platform_ai/agent/model/providers"
	agentschema "github.com/vevila2020/writing_platform_ai/agent/schema"
)

func TestStreamNormalizesMixedSSEFraming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1beta/models/gemini-test:streamGenerateContent" {
			t.Errorf("path = %q", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := writer.(http.Flusher)
		// The relay separates frames with CRLF keep-alive lines and comments;
		// the genai SDK only accepts canonical blank-line-delimited data frames.
		frames := []string{
			"data: {\"candidates\":[{\"index\":0,\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"Hel\"}]}}]}\n\n",
			"\r\n",
			": keep-alive\r\n\r\n",
			"data: {\"candidates\":[{\"index\":0,\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"lo\"}]},\"finishReason\":\"STOP\"}],\"usageMetadata\":{\"promptTokenCount\":3,\"candidatesTokenCount\":2,\"totalTokenCount\":5}}\n\n",
		}
		for _, frame := range frames {
			_, _ = io.WriteString(writer, frame)
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer server.Close()

	protocolOptions, err := providers.EncodeProtocolOptions(Compatibility{APIVersion: "v1beta"})
	if err != nil {
		t.Fatal(err)
	}
	model, err := NewAdapter().New(context.Background(), providers.ModelConfig{
		Provider: "google-test", Protocol: providers.ProtocolGoogleGenerativeAI,
		BaseURL: server.URL, Model: "gemini-test", APIKey: "secret",
		HTTPClient: server.Client(), ProtocolOptions: protocolOptions,
	})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := model.Stream(context.Background(), []*agentschema.Message{agentschema.UserMessage("hello")})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	defer stream.Close()

	var content, finishReason string
	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("receive stream: %v", err)
		}
		content += chunk.Content
		if chunk.ResponseMeta != nil {
			finishReason = chunk.ResponseMeta.FinishReason
		}
	}
	if content != "Hello" {
		t.Fatalf("content = %q, want %q", content, "Hello")
	}
	if finishReason != "STOP" {
		t.Fatalf("finish reason = %q, want STOP", finishReason)
	}
}

func TestStreamPreservesBareJSONErrorFrames(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := writer.(http.Flusher)
		frames := []string{
			"data: {\"candidates\":[{\"index\":0,\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"partial\"}]}}]}\n\n",
			"\r\n",
			"{\"error\":{\"code\":500,\"message\":\"relay exploded\",\"status\":\"INTERNAL\"}}\n\n",
		}
		for _, frame := range frames {
			_, _ = io.WriteString(writer, frame)
			if flusher != nil {
				flusher.Flush()
			}
		}
	}))
	defer server.Close()

	protocolOptions, err := providers.EncodeProtocolOptions(Compatibility{APIVersion: "v1beta"})
	if err != nil {
		t.Fatal(err)
	}
	model, err := NewAdapter().New(context.Background(), providers.ModelConfig{
		Provider: "google-test", Protocol: providers.ProtocolGoogleGenerativeAI,
		BaseURL: server.URL, Model: "gemini-test", APIKey: "secret",
		HTTPClient: server.Client(), ProtocolOptions: protocolOptions,
	})
	if err != nil {
		t.Fatal(err)
	}
	stream, err := model.Stream(context.Background(), []*agentschema.Message{agentschema.UserMessage("hello")})
	if err != nil {
		t.Fatalf("stream: %v", err)
	}
	defer stream.Close()

	var streamErr error
	for {
		_, err := stream.Recv()
		if err == nil {
			continue
		}
		if errors.Is(err, io.EOF) {
			break
		}
		streamErr = err
		break
	}
	if streamErr == nil || !strings.Contains(streamErr.Error(), "relay exploded") {
		t.Fatalf("stream error = %v, want the relay error", streamErr)
	}
}

func TestSSENormalizeTransportLeavesJSONResponsesAlone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(writer, `{"candidates":[]}`)
	}))
	defer server.Close()

	response, err := wrapSSEStreamingClient(server.Client()).Get(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if _, ok := response.Body.(*sseNormalizingBody); ok {
		t.Fatal("JSON response body was wrapped as an SSE stream")
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"candidates":[]}` {
		t.Fatalf("body = %q", body)
	}
}
