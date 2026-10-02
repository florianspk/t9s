package talos

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const cobraDirectiveError = 1

var (
	errInvalidCompletionOutput = errors.New("talos: invalid completion output")
	errCompletionFailed        = errors.New("talos: completion directive reports an error")
)

type completionRunner func(context.Context, ...string) ([]byte, error)

// GetLogStreams discovers log targets supported by the configured talosctl.
func (c *Client) GetLogStreams(ctx context.Context, node string) ([]string, error) {
	return getLogStreams(ctx, node, c.run)
}

func getLogStreams(ctx context.Context, node string, run completionRunner) ([]string, error) {
	output, err := run(ctx, "__completeNoDesc", "logs", "--nodes="+node, "")
	if err != nil {
		if !isNoDescCompletionUnsupported(err) {
			return nil, fmt.Errorf("discover log streams: %w", err)
		}

		output, err = run(ctx, "__complete", "logs", "--nodes="+node, "")
		if err != nil {
			return nil, fmt.Errorf("discover log streams with described completion: %w", err)
		}
	}

	streams, err := parseLogStreamCompletions(output)
	if err != nil {
		return nil, fmt.Errorf("discover log streams: %w", err)
	}

	return streams, nil
}

func isNoDescCompletionUnsupported(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	return strings.Contains(err.Error(), `unknown command "__completeNoDesc" for "talosctl"`)
}

func parseLogStreamCompletions(output []byte) ([]string, error) {
	lines := strings.Split(string(output), "\n")
	directiveIndex := -1
	for index := len(lines) - 1; index >= 0; index-- {
		if strings.TrimSpace(lines[index]) != "" {
			directiveIndex = index
			break
		}
	}
	if directiveIndex < 0 {
		return nil, fmt.Errorf("missing directive: %w", errInvalidCompletionOutput)
	}

	directive, err := parseCompletionDirective(strings.TrimSpace(lines[directiveIndex]))
	if err != nil {
		return nil, err
	}
	if directive&cobraDirectiveError != 0 {
		return nil, fmt.Errorf("directive %d: %w", directive, errCompletionFailed)
	}

	streams := make([]string, 0, directiveIndex)
	seen := make(map[string]struct{}, directiveIndex)
	for _, line := range lines[:directiveIndex] {
		candidate := strings.TrimSpace(strings.SplitN(line, "\t", 2)[0])
		if candidate == "" {
			continue
		}
		if _, err := parseCompletionDirective(candidate); err == nil {
			return nil, fmt.Errorf("content follows directive %q: %w", candidate, errInvalidCompletionOutput)
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		streams = append(streams, candidate)
	}

	return streams, nil
}

func parseCompletionDirective(line string) (uint64, error) {
	if len(line) < 2 || line[0] != ':' {
		return 0, fmt.Errorf("malformed directive %q: %w", line, errInvalidCompletionOutput)
	}
	for _, digit := range line[1:] {
		if digit < '0' || digit > '9' {
			return 0, fmt.Errorf("malformed directive %q: %w", line, errInvalidCompletionOutput)
		}
	}

	directive, err := strconv.ParseUint(line[1:], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse directive %q: %w", line, errInvalidCompletionOutput)
	}

	return directive, nil
}
