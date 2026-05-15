package model

import "testing"

func TestNewUISettingsAllFields(t *testing.T) {
	s := NewUISettings("My Dashboard", "logo.svg", "favicon.ico", "#ff0000", "#cc0000", "#ffeeee")
	if s == nil {
		t.Fatal("expected non-nil UISettings")
	}
	if s.Title != "My Dashboard" {
		t.Errorf("Title = %q, want %q", s.Title, "My Dashboard")
	}
	if s.Logo != "logo.svg" {
		t.Errorf("Logo = %q, want %q", s.Logo, "logo.svg")
	}
	if s.Favicon != "favicon.ico" {
		t.Errorf("Favicon = %q, want %q", s.Favicon, "favicon.ico")
	}
	if s.Palette == nil {
		t.Fatal("expected non-nil Palette")
	}
	if s.Palette.Accent != "#ff0000" {
		t.Errorf("Accent = %q, want %q", s.Palette.Accent, "#ff0000")
	}
}

func TestNewUISettingsAllEmpty(t *testing.T) {
	s := NewUISettings("", "", "", "", "", "")
	if s != nil {
		t.Errorf("expected nil for all-empty inputs, got %+v", s)
	}
}

func TestNewUISettingsTitleOnly(t *testing.T) {
	s := NewUISettings("Dashboard", "", "", "", "", "")
	if s == nil {
		t.Fatal("expected non-nil UISettings")
	}
	if s.Title != "Dashboard" {
		t.Errorf("Title = %q, want %q", s.Title, "Dashboard")
	}
	if s.Palette != nil {
		t.Errorf("expected nil Palette when no colors set, got %+v", s.Palette)
	}
}

func TestNewUISettingsPaletteOnly(t *testing.T) {
	s := NewUISettings("", "", "", "#abc", "", "")
	if s == nil {
		t.Fatal("expected non-nil UISettings")
	}
	if s.Palette == nil {
		t.Fatal("expected non-nil Palette")
	}
	if s.Palette.Accent != "#abc" {
		t.Errorf("Accent = %q, want %q", s.Palette.Accent, "#abc")
	}
}
