package inject

import (
	"context"
	"strings"
	"testing"

	"farsiforge/pkg/core"
)

type mockToolReg struct{}

func (m *mockToolReg) IsAvailable(name string) bool { return true }
func (m *mockToolReg) GetPath(name string) string   { return "fftools.exe" }
func (m *mockToolReg) GetPython() string            { return "python.exe" }
func (m *mockToolReg) Summary() string              { return "mock" }

func TestUnrealInjector_RejectsArchiveNestedLocres(t *testing.T) {
	tmp := t.TempDir()
	proj := core.NewProject("TestUEArchive", tmp, "unreal")
	proj.WorkingDir = tmp

	// Add an entry marked as coming from a .pak archive (Container != "")
	entry := core.StringEntry{
		ID:          "s_0001",
		Source:      "New Game",
		Translation: "بازی جدید",
		File:        "Game/Content/Localization/Game/en/Game.locres",
		Container:   "Game/Content/Paks/Game-WindowsNoEditor.pak",
		Path:        "Menu_NewGame",
		Status:      core.StatusTranslated,
	}
	proj.AddEntry(entry)

	injector := &UnrealInjector{}
	info := &core.GameInfo{GameRoot: tmp, Engine: "unreal"}
	res, err := injector.Inject(context.Background(), info, proj, &mockToolReg{}, core.DefaultPersianOptions())

	if err == nil {
		t.Fatalf("expected error when injecting archive-nested locres, got nil (res: %+v)", res)
	}
	if !strings.Contains(err.Error(), "Game-WindowsNoEditor.pak") || !strings.Contains(err.Error(), "archive write-back is not supported") {
		t.Fatalf("expected error mentioning pak container and archive write-back not supported, got: %v", err)
	}
	if len(res.ModifiedFiles) > 0 {
		t.Fatalf("expected no modified files on rejected archive injection, got: %v", res.ModifiedFiles)
	}
}

func TestUnrealInjector_RejectsUexpCompiledAssets(t *testing.T) {
	tmp := t.TempDir()
	proj := core.NewProject("TestUEUexp", tmp, "unreal")
	proj.WorkingDir = tmp

	entry := core.StringEntry{
		ID:          "s_0001",
		Source:      "Health",
		Translation: "سلامت",
		File:        "Game/Content/UI/WBP_HUD.uexp",
		Path:        "literal:1234",
		Status:      core.StatusTranslated,
	}
	proj.AddEntry(entry)

	injector := &UnrealInjector{}
	info := &core.GameInfo{GameRoot: tmp, Engine: "unreal"}
	res, err := injector.Inject(context.Background(), info, proj, &mockToolReg{}, core.DefaultPersianOptions())

	if err == nil {
		t.Fatalf("expected error when injecting .uexp compiled asset, got nil")
	}
	if !strings.Contains(err.Error(), ".uexp") || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("expected error mentioning .uexp is read-only, got: %v", err)
	}
	if len(res.ModifiedFiles) > 0 {
		t.Fatalf("expected no modified files on rejected uexp injection, got: %v", res.ModifiedFiles)
	}
}
