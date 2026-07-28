package windows

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestBuildHotkeyMatchesEitherSideModifiers(t *testing.T) {
	spec, err := buildHotkey([]string{"ctrl", "alt", "q"})
	if err != nil {
		t.Fatalf("buildHotkey() error = %v", err)
	}

	leftModifiers := map[uint32]bool{
		VK_LCONTROL: true,
		VK_LMENU:    true,
		'Q':         true,
	}
	if !spec.matches(leftModifiers) {
		t.Fatal("expected left-side modifiers to match")
	}

	rightModifiers := map[uint32]bool{
		VK_RCONTROL: true,
		VK_RMENU:    true,
		'Q':         true,
	}
	if !spec.matches(rightModifiers) {
		t.Fatal("expected right-side modifiers to match")
	}
}

func TestBuildHotkeyRequiresEveryModifier(t *testing.T) {
	spec, err := buildHotkey([]string{"ctrl", "alt", "q"})
	if err != nil {
		t.Fatalf("buildHotkey() error = %v", err)
	}

	withoutAlt := map[uint32]bool{
		VK_LCONTROL: true,
		'Q':         true,
	}
	if spec.matches(withoutAlt) {
		t.Fatal("hotkey matched without Alt")
	}
}

func TestBuildHotkeySupportsFunctionKeys(t *testing.T) {
	spec, err := buildHotkey([]string{"win", "f8"})
	if err != nil {
		t.Fatalf("buildHotkey() error = %v", err)
	}
	if spec.key != 0x77 {
		t.Fatalf("F8 virtual key = %#x, want %#x", spec.key, 0x77)
	}
}

func TestParseHotkeyFallsBackForInvalidConfig(t *testing.T) {
	spec := parseHotkey(
		[]string{"ctrl", "not-a-key"},
		[]string{"alt", "shift", "q"},
	)
	if spec.label != "Alt+Shift+Q" {
		t.Fatalf("fallback label = %q, want %q", spec.label, "Alt+Shift+Q")
	}
}

func TestMiddleMouseMessagesAreRecognizedAsHookTrigger(t *testing.T) {
	for _, message := range []uintptr{WM_MBUTTONDOWN, WM_MBUTTONUP} {
		if !isMiddleMouseMessage(message) {
			t.Fatalf("message %#x should be recognized as middle mouse", message)
		}
	}
	if isMiddleMouseMessage(0x0201) { // WM_LBUTTONDOWN
		t.Fatal("left mouse message must not be recognized as middle mouse")
	}
}

func TestWaitForClipboardChange(t *testing.T) {
	var sequence atomic.Uint32
	sequence.Store(10)

	go func() {
		time.Sleep(10 * time.Millisecond)
		sequence.Store(11)
	}()

	if !waitForClipboardChange(10, 200*time.Millisecond, time.Millisecond, sequence.Load) {
		t.Fatal("expected clipboard sequence change to be detected")
	}
}

func TestWaitForClipboardChangeTimesOut(t *testing.T) {
	if waitForClipboardChange(10, 10*time.Millisecond, time.Millisecond, func() uint32 { return 10 }) {
		t.Fatal("unexpected clipboard sequence change")
	}
}
