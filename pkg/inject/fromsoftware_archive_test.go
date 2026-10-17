package inject

import (
	"context"
	"strings"
	"testing"

	"farsiforge/pkg/core"
)

func TestFromSoftwareInjector_RejectsArchiveNestedEntries(t *testing.T) {
	tmp := t.TempDir()
	proj := core.NewProject("TestFSArchive", tmp, "fromsoftware")
	proj.WorkingDir = tmp

	// Add an entry marked as coming from an archive (Container != "")
	entry := core.StringEntry{
		ID:          "s_0001",
		Source:      "Elden Ring",
		Translation: "الدن رینگ",
		File:        "msg/engus/item.msgbnd.dcx/item.fmg",
		Container:   "Data0.bhd",
		Path:        "100",
		Status:      core.StatusTranslated,
	}
	proj.AddEntry(entry)

	injector := &FromSoftwareInjector{}
	info := &core.GameInfo{GameRoot: tmp, Engine: "fromsoftware"}
	res, err := injector.Inject(context.Background(), info, proj, nil, core.DefaultPersianOptions())

	if err == nil {
		t.Fatalf("expected error when injecting archive-nested entry, got nil (res: %+v)", res)
	}
	if !strings.Contains(err.Error(), "Data0.bhd") || !strings.Contains(err.Error(), "write-back is not supported") {
		t.Fatalf("expected error mentioning Data0.bhd and write-back not supported, got: %v", err)
	}
	if len(res.ModifiedFiles) > 0 {
		t.Fatalf("expected no modified files on rejected archive injection, got: %v", res.ModifiedFiles)
	}
}
