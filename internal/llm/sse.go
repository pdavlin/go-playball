package llm

import (
	"bufio"
	"io"
	"strings"
)

// SSEEvent is one server-sent-events block: an optional event type and the
// concatenated data lines (joined by "\n", with the leading "data: " and
// trailing CR stripped).
type SSEEvent struct {
	Type string
	Data string
}

// ScanSSE reads server-sent events from r and yields each complete event to
// fn. ScanSSE returns when fn returns false, when r reports EOF, or on a
// read error. Multi-line data: payloads are concatenated with newlines per
// the SSE spec. Empty events (data == "" and type == "") are skipped.
func ScanSSE(r io.Reader, fn func(SSEEvent) bool) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var ev SSEEvent
	var dataLines []string

	dispatch := func() bool {
		if len(dataLines) == 0 && ev.Type == "" {
			return true
		}
		ev.Data = strings.Join(dataLines, "\n")
		ok := fn(ev)
		ev = SSEEvent{}
		dataLines = dataLines[:0]
		return ok
	}

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" {
			if !dispatch() {
				return nil
			}
			continue
		}
		// Comment lines per SSE spec.
		if strings.HasPrefix(line, ":") {
			continue
		}
		field, value, ok := splitField(line)
		if !ok {
			continue
		}
		switch field {
		case "event":
			ev.Type = value
		case "data":
			dataLines = append(dataLines, value)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	// Flush any trailing event without a terminating blank line.
	dispatch()
	return nil
}

// splitField splits an SSE field line "name: value" into name and value,
// stripping one optional space after the colon per the spec.
func splitField(line string) (field, value string, ok bool) {
	idx := strings.IndexByte(line, ':')
	if idx < 0 {
		return line, "", true
	}
	field = line[:idx]
	value = line[idx+1:]
	if strings.HasPrefix(value, " ") {
		value = value[1:]
	}
	return field, value, true
}
