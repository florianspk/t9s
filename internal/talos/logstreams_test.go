package talos

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestGetLogStreams_usesNoDescCompletionWithExactArguments(t *testing.T) {
	// Given
	var calls [][]string
	run := func(_ context.Context, args ...string) ([]byte, error) {
		calls = append(calls, append([]string(nil), args...))

		return []byte("apid\nkubelet\n:4\n"), nil
	}

	// When
	streams, err := getLogStreams(context.Background(), "10.0.0.2", run)

	// Then
	if err != nil {
		t.Fatalf("GetLogStreams returned an error: %v", err)
	}
	wantCalls := [][]string{{"__completeNoDesc", "logs", "--nodes=10.0.0.2", ""}}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("completion calls = %v, want %v", calls, wantCalls)
	}
	wantStreams := []string{"apid", "kubelet"}
	if !reflect.DeepEqual(streams, wantStreams) {
		t.Fatalf("streams = %v, want %v", streams, wantStreams)
	}
}

func TestGetLogStreams_fallsBackToDescribedCompletionWhenNoDescIsUnsupported(t *testing.T) {
	// Given
	var calls [][]string
	run := func(_ context.Context, args ...string) ([]byte, error) {
		calls = append(calls, append([]string(nil), args...))
		if args[0] == "__completeNoDesc" {
			return nil, errors.New("exit status 1: unknown command \"__completeNoDesc\" for \"talosctl\"")
		}

		return []byte("apid\tAPI service\nkubelet\tKubernetes node agent\n:4\n"), nil
	}

	// When
	streams, err := getLogStreams(context.Background(), "node-a", run)

	// Then
	if err != nil {
		t.Fatalf("GetLogStreams returned an error: %v", err)
	}
	wantCalls := [][]string{
		{"__completeNoDesc", "logs", "--nodes=node-a", ""},
		{"__complete", "logs", "--nodes=node-a", ""},
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("completion calls = %v, want %v", calls, wantCalls)
	}
	wantStreams := []string{"apid", "kubelet"}
	if !reflect.DeepEqual(streams, wantStreams) {
		t.Fatalf("streams = %v, want %v", streams, wantStreams)
	}
}

func TestGetLogStreams_doesNotFallBackForUnrelatedErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "context cancellation with unsupported command text",
			err: fmt.Errorf(
				"unknown command \"__completeNoDesc\" for \"talosctl\": %w",
				context.Canceled,
			),
		},
		{name: "connectivity failure", err: errors.New("connection refused")},
		{name: "authentication failure", err: errors.New("permission denied")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// Given
			calls := 0
			run := func(_ context.Context, _ ...string) ([]byte, error) {
				calls++

				return nil, test.err
			}

			// When
			_, err := getLogStreams(context.Background(), "node-a", run)

			// Then
			if !errors.Is(err, test.err) {
				t.Fatalf("error = %v, want wrapped %v", err, test.err)
			}
			if calls != 1 {
				t.Fatalf("completion calls = %d, want 1", calls)
			}
		})
	}
}

func TestParseLogStreamCompletions_preservesOrderAndDeduplicatesTrimmedCandidates(t *testing.T) {
	// Given
	output := []byte(" kubelet \nkubelet\tduplicate description\n apid\tAPI service \ncontainerd\n apid \n:4\n")

	// When
	streams, err := parseLogStreamCompletions(output)

	// Then
	if err != nil {
		t.Fatalf("parse completions returned an error: %v", err)
	}
	want := []string{"kubelet", "apid", "containerd"}
	if !reflect.DeepEqual(streams, want) {
		t.Fatalf("streams = %v, want %v", streams, want)
	}
}

func TestParseLogStreamCompletions_acceptsValidEmptyOutput(t *testing.T) {
	// Given
	output := []byte("\n:4\n\n")

	// When
	streams, err := parseLogStreamCompletions(output)

	// Then
	if err != nil {
		t.Fatalf("parse completions returned an error: %v", err)
	}
	if len(streams) != 0 {
		t.Fatalf("streams = %v, want empty", streams)
	}
}

func TestParseLogStreamCompletions_rejectsMissingMalformedOrMisplacedDirective(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{name: "missing directive", output: "apid\nkubelet\n"},
		{name: "malformed directive", output: "apid\n:not-a-number\n"},
		{name: "negative directive", output: "apid\n:-1\n"},
		{name: "content after directive", output: "apid\n:4\nunexpected\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// When
			_, err := parseLogStreamCompletions([]byte(test.output))

			// Then
			if err == nil {
				t.Fatal("parse completions returned no error")
			}
		})
	}
}

func TestParseLogStreamCompletions_rejectsErrorDirective(t *testing.T) {
	// Given
	output := []byte("apid\n:5\n")

	// When
	_, err := parseLogStreamCompletions(output)

	// Then
	if err == nil {
		t.Fatal("parse completions returned no error")
	}
}
