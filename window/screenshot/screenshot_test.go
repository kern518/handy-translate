package screenshot

import (
	"image"
	"testing"
)

func TestCaptureSelectedScreenClampsToBounds(t *testing.T) {
	screenshotMu.Lock()
	ScreenshotImg = image.NewRGBA(image.Rect(0, 0, 100, 100))
	screenshotMu.Unlock()
	t.Cleanup(func() {
		screenshotMu.Lock()
		ScreenshotImg = nil
		screenshotMu.Unlock()
	})

	cropped := CaptureSelectedScreen(90, 90, 120, 120)
	if cropped == nil {
		t.Fatal("expected partially overlapping selection to be cropped")
	}
	if got := cropped.Bounds().Size(); got.X != 10 || got.Y != 10 {
		t.Fatalf("cropped size = %v, want 10x10", got)
	}
}

func TestCaptureSelectedScreenRejectsOutsideSelection(t *testing.T) {
	screenshotMu.Lock()
	ScreenshotImg = image.NewRGBA(image.Rect(0, 0, 100, 100))
	screenshotMu.Unlock()
	t.Cleanup(func() {
		screenshotMu.Lock()
		ScreenshotImg = nil
		screenshotMu.Unlock()
	})

	if cropped := CaptureSelectedScreen(120, 120, 140, 140); cropped != nil {
		t.Fatal("expected fully out-of-bounds selection to be rejected")
	}
}

func TestSelectDisplayIndexUsesDisplayContainingPoint(t *testing.T) {
	displays := []image.Rectangle{
		image.Rect(0, 0, 1920, 1080),
		image.Rect(-1280, 0, 0, 1024),
		image.Rect(1920, -200, 3840, 880),
	}

	tests := []struct {
		name  string
		point image.Point
		want  int
	}{
		{name: "primary", point: image.Pt(100, 100), want: 0},
		{name: "left display with negative coordinates", point: image.Pt(-500, 500), want: 1},
		{name: "right display", point: image.Pt(2500, 0), want: 2},
		{name: "outside falls back to primary", point: image.Pt(5000, 5000), want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := selectDisplayIndex(test.point, displays); got != test.want {
				t.Fatalf("selectDisplayIndex(%v) = %d, want %d", test.point, got, test.want)
			}
		})
	}
}
