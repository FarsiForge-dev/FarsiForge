package tools

import (
	"context"
	"debug/pe"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ToolStatusLevel indicates the health of a tool.
type ToolStatusLevel string

const (
	StatusOk           ToolStatusLevel = "ok"
	StatusMissing      ToolStatusLevel = "missing"
	StatusIncompatible ToolStatusLevel = "incompatible"
	StatusCorrupt      ToolStatusLevel = "corrupt"
	StatusWarning      ToolStatusLevel = "warning"
)

// ToolDiagnostic provides comprehensive details on a detected or missing external tool.
type ToolDiagnostic struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Category     string          `json:"category"`
	Found        bool            `json:"found"`
	Path         string          `json:"path"`
	Version      string          `json:"version"`
	Architecture string          `json:"architecture"`
	License      string          `json:"license"`
	Source       string          `json:"source"`
	Status       ToolStatusLevel `json:"status"`
	TestResult   string          `json:"test_result"`
	ErrorMessage string          `json:"error_message,omitempty"`
}

// OperationCheckResult contains the capability decision for a specific engine and operation.
type OperationCheckResult struct {
	Engine          string           `json:"engine"`
	Operation       string           `json:"operation"`
	Allowed         bool             `json:"allowed"`
	MissingRequired []string         `json:"missing_required"`
	MissingOptional []string         `json:"missing_optional"`
	Warnings        []string         `json:"warnings"`
	Blockers        []string         `json:"blockers"`
	ToolsEvaluated  []ToolDiagnostic `json:"tools_evaluated"`
}

// DiagnosticReport is a complete snapshot of all helper tools and environment.
type DiagnosticReport struct {
	GeneratedAt time.Time        `json:"generated_at"`
	OS          string           `json:"os"`
	Arch        string           `json:"arch"`
	GoVersion   string           `json:"go_version"`
	ToolsDir    string           `json:"tools_dir"`
	Tools       []ToolDiagnostic `json:"tools"`
	Summary     string           `json:"summary"`
}

// DetectPEArchitecture inspects Windows PE binary headers to determine target architecture.
func DetectPEArchitecture(filePath string) (string, error) {
	f, err := pe.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	switch f.FileHeader.Machine {
	case pe.IMAGE_FILE_MACHINE_AMD64:
		return "x86_64", nil
	case pe.IMAGE_FILE_MACHINE_I386:
		return "x86", nil
	case pe.IMAGE_FILE_MACHINE_ARM64:
		return "arm64", nil
	default:
		return fmt.Sprintf("machine_0x%x", f.FileHeader.Machine), nil
	}
}

// ProbeTool runs a quick execution check against an executable tool to verify readability and version.
func ProbeTool(ctx context.Context, id, path string) (version string, arch string, testOutput string, err error) {
	if path == "" {
		return "", "", "", fmt.Errorf("tool path is empty")
	}

	fi, statErr := os.Stat(path)
	if statErr != nil {
		return "", "", "", statErr
	}

	// Validate PE header for Windows executables / DLLs
	if !fi.IsDir() && (strings.HasSuffix(strings.ToLower(path), ".exe") || strings.HasSuffix(strings.ToLower(path), ".dll")) {
		peArch, peErr := DetectPEArchitecture(path)
		if peErr != nil {
			return "", "", "", fmt.Errorf("corrupt executable (invalid PE header): %w", peErr)
		}
		arch = peArch
	}
	if arch == "" {
		arch = runtime.GOARCH
	}

	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
	}

	var args []string
	switch id {
	case "python":
		args = []string{"--version"}
	case "repak":
		args = []string{"--version"}
	case "retoc":
		args = []string{"--version"}
	case "fftools":
		args = []string{"version"}
	case "gdre_tools":
		args = []string{"--version"}
	case "die":
		args = []string{"-v"}
	default:
		return "detected", arch, "File exists and accessible", nil
	}

	res, runErr := RunSilentBounded(ctx, 64*1024, "", path, args...)
	testOutput = strings.TrimSpace(res.Combined)
	if runErr != nil {
		if testOutput != "" {
			version = strings.Split(testOutput, "\n")[0]
			return version, arch, testOutput, nil
		}
		return "", arch, testOutput, runErr
	}

	lines := strings.Split(testOutput, "\n")
	if len(lines) > 0 {
		version = strings.TrimSpace(lines[0])
	}
	return version, arch, testOutput, nil
}

// InspectTool gathers full diagnostic info for a specific tool in the registry.
func InspectTool(ctx context.Context, reg *Registry, id string) ToolDiagnostic {
	diag := ToolDiagnostic{
		ID:     id,
		Status: StatusMissing,
	}

	switch id {
	case "python":
		diag.Name = "Python 3 Runtime"
		diag.Category = "Runtime"
		diag.License = "PSF"
		diag.Path = reg.GetPython()
		if reg.IsAvailable("python") {
			diag.Found = true
			diag.Status = StatusOk
			diag.Version = reg.PythonVersion
			diag.Source = "System PATH / Environment"
			if strings.Contains(diag.Path, reg.RootDir) {
				diag.Source = "Bundled Tools"
			}
			v, arch, testOut, err := ProbeTool(ctx, "python", diag.Path)
			if err == nil {
				diag.Architecture = arch
				diag.TestResult = testOut
				if v != "" {
					diag.Version = v
				}
			}
		} else {
			diag.ErrorMessage = "Python executable not found in PATH or Tools folder"
		}

	case "unitypy":
		diag.Name = "UnityPy Python Module"
		diag.Category = "Unity"
		diag.License = "MIT"
		diag.Path = reg.GetPath("unitypy")
		if reg.IsAvailable("py_UnityPy") {
			diag.Found = true
			diag.Status = StatusOk
			diag.Source = "Python Environment"
			diag.Version = "Available"
			diag.TestResult = "Import UnityPy successful"
		} else {
			diag.ErrorMessage = "UnityPy module is missing. Install via pip: python -m pip install UnityPy"
		}

	case "repak":
		diag.Name = "repak (Unreal PAK CLI)"
		diag.Category = "Unreal"
		diag.License = "MIT"
		diag.Path = reg.Repak
		if reg.Repak != "" && fileExists(reg.Repak) {
			diag.Found = true
			diag.Source = "Bundled Tools"
			v, arch, testOut, err := ProbeTool(ctx, "repak", diag.Path)
			diag.Architecture = arch
			diag.TestResult = testOut
			if err == nil {
				diag.Status = StatusOk
				if v != "" {
					diag.Version = v
				}
			} else {
				if strings.Contains(err.Error(), "corrupt") {
					diag.Status = StatusCorrupt
				} else {
					diag.Status = StatusIncompatible
				}
				diag.ErrorMessage = fmt.Sprintf("execution test failed: %v", err)
			}
		} else {
			diag.ErrorMessage = "repak.exe not found in Tools/repak"
		}

	case "retoc":
		diag.Name = "retoc (UE5 IoStore utoc/ucas CLI)"
		diag.Category = "Unreal"
		diag.License = "MIT"
		diag.Path = reg.Retoc
		if reg.Retoc != "" && fileExists(reg.Retoc) {
			diag.Found = true
			diag.Source = "Bundled Tools"
			v, arch, testOut, err := ProbeTool(ctx, "retoc", diag.Path)
			diag.Architecture = arch
			diag.TestResult = testOut
			if err == nil {
				diag.Status = StatusOk
				if v != "" {
					diag.Version = v
				}
			} else {
				if strings.Contains(err.Error(), "corrupt") {
					diag.Status = StatusCorrupt
				} else {
					diag.Status = StatusIncompatible
				}
				diag.ErrorMessage = fmt.Sprintf("execution test failed: %v", err)
			}
		} else {
			diag.Status = StatusWarning
			diag.ErrorMessage = "retoc.exe not found (required for UE5 IoStore containers)"
		}

	case "gdre_tools":
		diag.Name = "GDRE Tools (Godot Reverse Engineering)"
		diag.Category = "Godot"
		diag.License = "MIT"
		diag.Path = reg.GDRETools
		if reg.GDRETools != "" && fileExists(reg.GDRETools) {
			diag.Found = true
			diag.Source = "Bundled Tools"
			v, arch, testOut, err := ProbeTool(ctx, "gdre_tools", diag.Path)
			diag.Architecture = arch
			diag.TestResult = testOut
			if err == nil {
				diag.Status = StatusOk
				if v != "" {
					diag.Version = v
				}
			} else {
				diag.Status = StatusCorrupt
				diag.ErrorMessage = err.Error()
			}
		} else {
			diag.ErrorMessage = "gdre_tools.exe not found in Tools/gdre_tools"
		}

	case "fftools":
		diag.Name = "FFTools (FarsiForge Native Binary Helper)"
		diag.Category = "General"
		diag.License = "MIT"
		diag.Path = reg.FFTools
		if reg.FFTools != "" && fileExists(reg.FFTools) {
			diag.Found = true
			diag.Source = "Bundled Tools"
			v, arch, testOut, err := ProbeTool(ctx, "fftools", diag.Path)
			diag.Architecture = arch
			diag.TestResult = testOut
			if err == nil {
				diag.Status = StatusOk
				if v != "" {
					diag.Version = v
				}
			} else {
				diag.Status = StatusCorrupt
				diag.ErrorMessage = err.Error()
			}
		} else {
			diag.ErrorMessage = "fftools.exe not found in Tools/fftools"
		}

	case "patcher":
		diag.Name = "FarsiForge Standalone Patcher"
		diag.Category = "Distribution"
		diag.License = "MIT"
		diag.Path = reg.Patcher
		if diag.Path == "" {
			diag.Path = filepath.Join(reg.RootDir, "patcher", "FarsiForgePatcher.exe")
		}
		if fileExists(diag.Path) {
			diag.Found = true
			diag.Source = "Bundled Tools"
			if arch, err := DetectPEArchitecture(diag.Path); err == nil {
				diag.Architecture = arch
				diag.Status = StatusOk
				diag.Version = "FFP1 v2.0"
				diag.TestResult = "Executable PE binary verified"
			} else {
				diag.Status = StatusCorrupt
				diag.ErrorMessage = fmt.Sprintf("corrupt patcher binary: %v", err)
			}
		} else {
			diag.ErrorMessage = "FarsiForgePatcher.exe not found (required for standalone patch packages)"
		}

	case "codewalker_magic":
		diag.Name = "CodeWalker magic.dat (RAGE NG Keys)"
		diag.Category = "RAGE"
		diag.License = "Proprietary / Key Material"
		diag.Path = reg.CodeWalkerMagic
		if diag.Path != "" && fileExists(diag.Path) {
			diag.Found = true
			diag.Status = StatusOk
			diag.Source = "Bundled Tools"
			diag.Architecture = "data"
			diag.Version = "Valid"
			diag.TestResult = "Key material present"
		} else {
			diag.Status = StatusWarning
			diag.ErrorMessage = "magic.dat not found in Tools/CodeWalker (required for GTA V / RAGE NG encrypted archives)"
		}

	case "oodle":
		diag.Name = "Oodle Decompression Library (oo2core)"
		diag.Category = "FromSoftware"
		diag.License = "Epic Games / RAD Game Tools"
		oodlePath := filepath.Join(reg.RootDir, "FromSoftware", "oo2core_6_win64.dll")
		diag.Path = oodlePath
		if fileExists(oodlePath) {
			diag.Found = true
			diag.Source = "Bundled Tools"
			if arch, err := DetectPEArchitecture(oodlePath); err == nil {
				diag.Architecture = arch
				diag.Status = StatusOk
				diag.Version = "Oodle 2.9"
				diag.TestResult = "Native DLL verified"
			} else {
				diag.Status = StatusCorrupt
				diag.ErrorMessage = err.Error()
			}
		} else {
			diag.Status = StatusWarning
			diag.ErrorMessage = "oo2core_6_win64.dll not found in Tools/FromSoftware (required for compressed FromSoftware BHD5/BDT archives)"
		}

	case "vazirmatn":
		diag.Name = "Vazirmatn Persian Typography Asset"
		diag.Category = "Typography"
		diag.License = "OFL-1.1"
		diag.Path = reg.VazirmatnFont
		if diag.Path != "" && fileExists(diag.Path) {
			diag.Found = true
			diag.Status = StatusOk
			diag.Source = "Bundled Tools"
			diag.Architecture = "font"
			diag.Version = "v33.003"
			diag.TestResult = "Font asset ready"
		} else {
			diag.Status = StatusWarning
			diag.ErrorMessage = "Vazirmatn font file not found in Tools/vazirmatn"
		}
	}

	return diag
}

// CheckOperationRequirements evaluates tool dependencies for an engine and operation.
func CheckOperationRequirements(reg *Registry, engine, op string) OperationCheckResult {
	res := OperationCheckResult{
		Engine:          engine,
		Operation:       op,
		Allowed:         true,
		MissingRequired: make([]string, 0),
		MissingOptional: make([]string, 0),
		Warnings:        make([]string, 0),
		Blockers:        make([]string, 0),
		ToolsEvaluated:  make([]ToolDiagnostic, 0),
	}

	if reg == nil {
		res.Allowed = false
		res.Blockers = append(res.Blockers, "Tool registry is not initialized")
		return res
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	evalTool := func(id string, required bool, desc string) ToolDiagnostic {
		diag := InspectTool(ctx, reg, id)
		res.ToolsEvaluated = append(res.ToolsEvaluated, diag)
		if !diag.Found {
			if required {
				res.MissingRequired = append(res.MissingRequired, id)
				res.Blockers = append(res.Blockers, fmt.Sprintf("%s is required: %s", diag.Name, desc))
				res.Allowed = false
			} else {
				res.MissingOptional = append(res.MissingOptional, id)
				res.Warnings = append(res.Warnings, fmt.Sprintf("%s is optional: %s", diag.Name, desc))
			}
		} else if diag.Status == StatusIncompatible || diag.Status == StatusCorrupt {
			if required {
				res.Blockers = append(res.Blockers, fmt.Sprintf("%s has status %s: %s", diag.Name, diag.Status, diag.ErrorMessage))
				res.Allowed = false
			} else {
				res.Warnings = append(res.Warnings, fmt.Sprintf("%s has status %s: %s", diag.Name, diag.Status, diag.ErrorMessage))
			}
		}
		return diag
	}

	normEngine := strings.ToLower(engine)
	normOp := strings.ToLower(op)

	switch normEngine {
	case "unreal":
		evalTool("repak", true, "needed to unpack and repack Unreal .pak files")
		if normOp == "rebuild_archive" || normOp == "iostore" {
			evalTool("retoc", false, "needed for UE5 IoStore .utoc/.ucas container packing")
		}
	case "unity":
		evalTool("python", true, "needed to execute Unity asset extraction/injection scripts")
		evalTool("unitypy", true, "needed to parse and rebuild Unity .assets/.bundle files")
	case "godot":
		t1 := evalTool("fftools", false, "native helper for Godot PCK extraction and translation injection")
		t2 := evalTool("gdre_tools", false, "reverse engineering toolkit for Godot PCK archives")
		if (t1.Status != StatusOk) && (t2.Status != StatusOk) {
			res.MissingRequired = append(res.MissingRequired, "gdre_tools/fftools")
			res.Blockers = append(res.Blockers, "Either fftools or gdre_tools is required for Godot operations")
			res.Allowed = false
		}
	case "fromsoftware":
		evalTool("fftools", true, "needed for FromSoftware FMG text and BND archive serialization")
		evalTool("oodle", false, "needed for decompressing BHD5/BDT game archives (Elden Ring, Sekiro)")
	case "rage":
		evalTool("codewalker_magic", false, "needed for decrypting modern GTA V NG RPF archives")
	}

	if normOp == "build_patch" || normOp == "package" {
		evalTool("patcher", true, "needed to bundle standalone installer executable")
	}

	return res
}

// GenerateDiagnosticReport runs a comprehensive system and tool diagnostic suite.
func GenerateDiagnosticReport(reg *Registry) *DiagnosticReport {
	if reg == nil {
		reg, _ = NewRegistry("")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	toolsToInspect := []string{
		"python",
		"unitypy",
		"repak",
		"retoc",
		"gdre_tools",
		"fftools",
		"patcher",
		"codewalker_magic",
		"oodle",
		"vazirmatn",
	}

	diags := make([]ToolDiagnostic, 0, len(toolsToInspect))
	for _, id := range toolsToInspect {
		diags = append(diags, InspectTool(ctx, reg, id))
	}

	report := &DiagnosticReport{
		GeneratedAt: time.Now(),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		GoVersion:   runtime.Version(),
		ToolsDir:    reg.RootDir,
		Tools:       diags,
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("FarsiForge Diagnostic Report (%s)\n", report.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("OS: %s | Arch: %s | Go: %s\n", report.OS, report.Arch, report.GoVersion))
	sb.WriteString(fmt.Sprintf("Tools Directory: %s\n\n", report.ToolsDir))

	for _, d := range diags {
		statusSym := "✓"
		if !d.Found {
			if d.Status == StatusWarning {
				statusSym = "⚠"
			} else {
				statusSym = "✗"
			}
		}
		sb.WriteString(fmt.Sprintf("%s [%s] %s (%s, %s)\n", statusSym, d.Status, d.Name, d.Source, d.Architecture))
		if d.Path != "" {
			sb.WriteString(fmt.Sprintf("   Path: %s\n", d.Path))
		}
		if d.Version != "" {
			sb.WriteString(fmt.Sprintf("   Version: %s\n", d.Version))
		}
		if d.ErrorMessage != "" {
			sb.WriteString(fmt.Sprintf("   Notice: %s\n", d.ErrorMessage))
		}
	}
	report.Summary = sb.String()

	return report
}

// ToMarkdown generates a clean GitHub-flavored markdown representation of the report.
func (r *DiagnosticReport) ToMarkdown() string {
	var sb strings.Builder
	sb.WriteString("# FarsiForge Tool & Environment Diagnostic Report\n\n")
	sb.WriteString(fmt.Sprintf("- **Generated:** `%s`\n", r.GeneratedAt.Format("2006-01-02 15:04:05 MST")))
	sb.WriteString(fmt.Sprintf("- **Platform:** `%s / %s`\n", r.OS, r.Arch))
	sb.WriteString(fmt.Sprintf("- **Runtime:** `%s`\n", r.GoVersion))
	sb.WriteString(fmt.Sprintf("- **Tools Root:** `%s`\n\n", r.ToolsDir))

	sb.WriteString("## Tool Status Matrix\n\n")
	sb.WriteString("| Tool | Category | Status | Arch | Version | License | Path |\n")
	sb.WriteString("| :--- | :--- | :---: | :---: | :--- | :--- | :--- |\n")

	for _, d := range r.Tools {
		statusBadge := "✅ OK"
		if !d.Found {
			if d.Status == StatusWarning {
				statusBadge = "⚠️ Optional Missing"
			} else {
				statusBadge = "❌ Missing"
			}
		} else if d.Status == StatusIncompatible {
			statusBadge = "⛔ Incompatible"
		} else if d.Status == StatusCorrupt {
			statusBadge = "💥 Corrupt"
		}
		sb.WriteString(fmt.Sprintf("| **%s** | %s | %s | `%s` | %s | %s | `%s` |\n",
			d.Name, d.Category, statusBadge, d.Architecture, d.Version, d.License, filepath.Base(d.Path)))
	}

	sb.WriteString("\n## Details & Diagnostics\n\n")
	for _, d := range r.Tools {
		sb.WriteString(fmt.Sprintf("### %s (`%s`)\n", d.Name, d.ID))
		sb.WriteString(fmt.Sprintf("- **Path:** `%s`\n", d.Path))
		sb.WriteString(fmt.Sprintf("- **Source:** %s\n", d.Source))
		if d.Version != "" {
			sb.WriteString(fmt.Sprintf("- **Version String:** `%s`\n", d.Version))
		}
		if d.TestResult != "" {
			sb.WriteString(fmt.Sprintf("- **Probe Result:**\n```text\n%s\n```\n", d.TestResult))
		}
		if d.ErrorMessage != "" {
			sb.WriteString(fmt.Sprintf("- **Error/Guidance:** %s\n", d.ErrorMessage))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
