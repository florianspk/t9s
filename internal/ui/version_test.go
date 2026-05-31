package ui

import (
	"strings"
	"testing"
)

// ── checkVersionMismatch ──────────────────────────────────────────────────────

func TestCheckVersionMismatchEqual(t *testing.T) {
	if got := checkVersionMismatch("v1.13.3", "v1.13.3"); got != "" {
		t.Errorf("equal versions: want empty, got %q", got)
	}
}

func TestCheckVersionMismatchMinorDiffOne(t *testing.T) {
	got := checkVersionMismatch("v1.12.0", "v1.13.3")
	if got == "" {
		t.Error("1-minor diff: want non-empty warning")
	}
	// Light warning — no update URL expected.
	if strings.Contains(got, "curl") {
		t.Errorf("1-minor diff should be a light warning without curl command, got %q", got)
	}
}

func TestCheckVersionMismatchMinorDiffTwo(t *testing.T) {
	got := checkVersionMismatch("v1.11.0", "v1.13.3")
	if got == "" {
		t.Error("2-minor diff: want critical warning")
	}
	if !strings.Contains(got, "curl") {
		t.Errorf("2-minor diff must include update command, got %q", got)
	}
	if !strings.Contains(got, "v1.13.3") {
		t.Errorf("warning must mention server version, got %q", got)
	}
}

func TestCheckVersionMismatchMinorDiffThree(t *testing.T) {
	got := checkVersionMismatch("v1.10.0", "v1.13.3")
	if got == "" {
		t.Error("3-minor diff: want critical warning")
	}
	if !strings.Contains(got, "curl") {
		t.Errorf("3-minor diff must include update command, got %q", got)
	}
}

func TestCheckVersionMismatchEmptyClient(t *testing.T) {
	if got := checkVersionMismatch("", "v1.13.3"); got != "" {
		t.Errorf("empty client: want empty, got %q", got)
	}
}

func TestCheckVersionMismatchEmptyServer(t *testing.T) {
	if got := checkVersionMismatch("v1.13.3", ""); got != "" {
		t.Errorf("empty server: want empty, got %q", got)
	}
}

func TestCheckVersionMismatchBothEmpty(t *testing.T) {
	if got := checkVersionMismatch("", ""); got != "" {
		t.Errorf("both empty: want empty, got %q", got)
	}
}

func TestCheckVersionMismatchMalformed(t *testing.T) {
	// Malformed versions must not panic.
	_ = checkVersionMismatch("garbage", "v1.13.3")
	_ = checkVersionMismatch("v1", "v1.13.3")
	_ = checkVersionMismatch("v1.13.3", "notaversion")
}

func TestCheckVersionMismatchNoPrefixV(t *testing.T) {
	// Versions without leading "v" should still work.
	got := checkVersionMismatch("1.11.0", "1.13.3")
	if got == "" {
		t.Error("no-prefix versions: 2-minor diff should still warn")
	}
}

// ── computeScrollStart ────────────────────────────────────────────────────────

func TestComputeScrollStartFitsAll(t *testing.T) {
	// When all items fit, start is always 0.
	for cur := 0; cur < 5; cur++ {
		if got := computeScrollStart(cur, 5, 10); got != 0 {
			t.Errorf("cur=%d: all fit, want start=0, got %d", cur, got)
		}
	}
}

func TestComputeScrollStartCursorAtStart(t *testing.T) {
	if got := computeScrollStart(0, 100, 10); got != 0 {
		t.Errorf("cursor at 0: want start=0, got %d", got)
	}
}

func TestComputeScrollStartCursorAtEnd(t *testing.T) {
	got := computeScrollStart(99, 100, 10)
	if got > 90 {
		t.Errorf("cursor at end: start should be ≤90, got %d", got)
	}
	if got < 0 {
		t.Errorf("start must not be negative, got %d", got)
	}
}

func TestComputeScrollStartCursorCentered(t *testing.T) {
	// Cursor in the middle should produce a start that puts it near center.
	got := computeScrollStart(50, 100, 10)
	// cursor should be visible: start ≤ 50 < start+10
	if got > 50 || got+10 <= 50 {
		t.Errorf("cursor 50 not visible: start=%d, window=[%d,%d)", got, got, got+10)
	}
}

func TestComputeScrollStartNeverNegative(t *testing.T) {
	for cur := 0; cur < 20; cur++ {
		if got := computeScrollStart(cur, 20, 10); got < 0 {
			t.Errorf("cur=%d: got negative start %d", cur, got)
		}
	}
}

func TestComputeScrollStartNeverExceedsBound(t *testing.T) {
	total, maxRows := 20, 10
	for cur := 0; cur < total; cur++ {
		got := computeScrollStart(cur, total, maxRows)
		if got+maxRows > total {
			t.Errorf("cur=%d: start=%d overflows total=%d with window=%d", cur, got, total, maxRows)
		}
	}
}
