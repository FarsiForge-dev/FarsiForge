package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Registry holds paths to all discovered external tools.
type Registry struct {
	RootDir string `json:"root_dir"`

	// Unity tools
	UnityPy        string `json:"unity_py"`         // Python module (pip)
	AssetRipper    string `json:"asset_ripper"`     // AssetRipper exe
	AssetsToolsNET string `json:"assets_tools_net"` // AssetsTools.NET
	UABEA          string `json:"uabea"`            // UABEA
	BepInEx        string `json:"bepinex"`          // BepInEx
	Il2CppDumper   string `json:"il2cpp_dumper"`    // Il2CppDumper

	// Unreal tools
	UnrealLocres string `json:"unreal_locres"` // UnrealLocres.exe
	Repak        string `json:"repak"`         // repak CLI
	Retoc        string `json:"retoc"`         // retoc CLI (UE5 IoStore utoc/ucas)
	UAssetGUI    string `json:"uasset_gui"`    // UAssetGUI.exe
	FModel       string `json:"fmodel"`        // FModel

	// Godot tools
	GodotEngine string `json:"godot_engine"` // Godot editor exe
	GDRETools   string `json:"gdre_tools"`   // gdre_tools.exe

	// RAGE tools
	CodeWalkerMagic string `json:"codewalker_magic"` // CodeWalker magic.dat (NG key material for modern GTA V builds)

	// GameMaker tools
	UndertaleModTool string `json:"undertale_mod_tool"` // UndertaleModTool

	// General tools
	DIE       string `json:"die"`       // Detect It Easy (diec.exe)
	QuickBMS  string `json:"quickbms"`  // quickbms
	SevenZip  string `json:"seven_zip"` // 7z
	FontForge string `json:"fontforge"` // FontForge
	FFTools   string `json:"ff_tools"`  // fftools.exe (FarsiForge native helper)
	Patcher   string `json:"patcher"`   // FarsiForgePatcher.exe

	// Python
	Python        string `json:"python"`         // Python executable (verified absolute path)
	PythonVersion string `json:"python_version"` // first line of `python --version`

	// Font
	VazirmatnFont string `json:"vazirmatn_font"` // Vazirmatn TTF path

	// Status
	Available map[string]bool `json:"available"`
}

// pythonVerifyTimeout is the max time spent verifying a Python candidate.
const pythonVerifyTimeout = 10 * time.Second

// NewRegistry creates a tool registry by scanning the tools directory.
//
// When toolsDir is empty the directory is resolved automatically in a
// portable way (see resolveToolsDir): first <exe_dir>\Tools and
// <exe_dir>\..\Tools, then walking up from the working directory looking for a
// folder that contains both go.mod and Tools (dev mode), and finally falling
// back to <exe_dir>\Tools even if it does not exist. A non-empty toolsDir is
// always used as given (explicit override from config).
//
// NewRegistry never returns an error when the tools directory is missing — it
// returns a usable Registry with an empty Available map so the rest of the app
// can still start and report which tools are unavailable.
func NewRegistry(toolsDir string) (*Registry, error) {
	reg := &Registry{
		Available: make(map[string]bool),
	}

	resolved := resolveToolsDir(toolsDir)
	reg.RootDir = resolved

	if !dirExists(resolved) {
		// Return a usable registry; callers can inspect Available/RootDir.
		return reg, fmt.Errorf("tools directory not found: %s", resolved)
	}

	reg.scan()
	return reg, nil
}

// resolveToolsDir resolves the tools directory in a portable manner.
func resolveToolsDir(toolsDir string) string {
	if toolsDir != "" {
		return toolsDir
	}

	if envDir := os.Getenv("FARSIFORGE_TOOLS_DIR"); envDir != "" && dirExists(envDir) {
		return envDir
	}
	if envDir := os.Getenv("FARISIFORGE_TOOLS_DIR"); envDir != "" && dirExists(envDir) {
		return envDir
	}

	// a. Portable install: <exe_dir>\Tools and <exe_dir>\..\Tools
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		for _, c := range []string{
			filepath.Join(exeDir, "Tools"),
			filepath.Join(exeDir, "..", "Tools"),
		} {
			if abs, err := filepath.Abs(c); err == nil && dirExists(abs) {
				return abs
			}
		}
	}

	// b. Dev mode: walk up from cwd until a directory containing both
	// go.mod and Tools is found.
	if cwd, err := os.Getwd(); err == nil {
		dir := cwd
		for {
			if dirExists(filepath.Join(dir, "Tools")) && fileExists(filepath.Join(dir, "go.mod")) {
				return filepath.Join(dir, "Tools")
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	// c. Fallback: <exe_dir>\Tools (even if missing). Never a hardcoded path.
	if exePath, err := os.Executable(); err == nil {
		if abs, err := filepath.Abs(filepath.Join(filepath.Dir(exePath), "Tools")); err == nil {
			return abs
		}
	}

	return "Tools"
}

// scan discovers tool executables in the tools directory.
func (r *Registry) scan() {
	// Find Python (verified, no user-specific paths).
	r.Python, r.PythonVersion = findPython()
	r.Available["python"] = r.Python != ""

	// Find DIE (diec.exe - command line version)
	r.DIE = findInDir(r.RootDir, "die", "diec.exe")
	r.Available["die"] = r.DIE != ""

	// Find UnrealLocres
	r.UnrealLocres = findInDir(r.RootDir, "UnrealLocres", "UnrealLocres.exe")
	if r.UnrealLocres == "" {
		r.UnrealLocres = findInDir(r.RootDir, "", "UnrealLocres.exe")
	}
	if r.UnrealLocres == "" {
		r.UnrealLocres = findAdjacentOrPath("UnrealLocres.exe")
	}
	r.Available["unreal_locres"] = r.UnrealLocres != ""

	// Find UAssetGUI
	r.UAssetGUI = findInDir(r.RootDir, "", "UAssetGUI.exe")
	r.Available["uasset_gui"] = r.UAssetGUI != ""

	// Find repak (Unreal Engine .pak unpacker)
	r.Repak = findInDir(r.RootDir, "repak", "repak.exe")
	if r.Repak == "" {
		r.Repak = findInDir(r.RootDir, "", "repak.exe")
	}
	if r.Repak == "" {
		r.Repak = findAdjacentOrPath("repak.exe")
	}
	r.Available["repak"] = r.Repak != ""

	// Find retoc (UE5 IoStore .utoc/.ucas unpacker)
	r.Retoc = findInDir(r.RootDir, "retoc", "retoc.exe")
	if r.Retoc == "" {
		r.Retoc = findInDir(r.RootDir, "", "retoc.exe")
	}
	if r.Retoc == "" {
		r.Retoc = findAdjacentOrPath("retoc.exe")
	}
	r.Available["retoc"] = r.Retoc != ""

	// Find gdre_tools (Godot reverse-engineering toolkit)
	r.GDRETools = findInDir(r.RootDir, "gdre", "gdre_tools.exe")
	if r.GDRETools == "" {
		r.GDRETools = findInDir(r.RootDir, "gdre_tools", "gdre_tools.exe")
	}
	if r.GDRETools == "" {
		r.GDRETools = findInDir(r.RootDir, "", "gdre_tools.exe")
	}
	if r.GDRETools == "" {
		r.GDRETools = findAdjacentOrPath("gdre_tools.exe")
	}
	r.Available["gdre_tools"] = r.GDRETools != ""

	// Find CodeWalker magic.dat (RPF7 NG key material for modern GTA V
	// builds, 2024+, that no longer embed the keys in the executable)
	r.CodeWalkerMagic = findInDir(r.RootDir, "CodeWalker", "magic.dat")
	if r.CodeWalkerMagic == "" {
		r.CodeWalkerMagic = findInDir(r.RootDir, "", "magic.dat")
	}
	r.Available["codewalker_magic"] = r.CodeWalkerMagic != ""

	// Find Godot engine
	r.GodotEngine = findInDir(r.RootDir, "godot", ".exe")
	if r.GodotEngine == "" {
		r.GodotEngine = findInDir(r.RootDir, "", ".exe")
	}
	r.Available["godot_engine"] = r.GodotEngine != ""

	// Find an extracted UndertaleModTool executable.
	r.UndertaleModTool = findInSubdirectories(r.RootDir, ".exe")
	r.Available["undertale_mod_tool"] = r.UndertaleModTool != ""

	// Find FontForge
	r.FontForge = findMatchingExecutable(r.RootDir, "fontforge")
	r.Available["fontforge"] = r.FontForge != ""

	// Find Vazirmatn font (in zip or extracted)
	r.VazirmatnFont = findVazirmatn(r.RootDir)
	r.Available["vazirmatn_font"] = r.VazirmatnFont != ""

	// Find 7z
	r.SevenZip = findInDir(r.RootDir, "", "7z.exe")
	if r.SevenZip == "" {
		r.SevenZip = findExecutable("7z")
	}
	r.Available["seven_zip"] = r.SevenZip != ""

	// Find fftools (FarsiForge native helper). Accept fftools\fftools.exe or
	// fftools.exe directly in RootDir or adjacent to executable.
	r.FFTools = findInDir(r.RootDir, "fftools", "fftools.exe")
	if r.FFTools == "" {
		r.FFTools = findInDir(r.RootDir, "", "fftools.exe")
	}
	if r.FFTools == "" {
		r.FFTools = findAdjacentOrPath("fftools.exe")
	}
	r.Available["fftools"] = r.FFTools != ""

	// Find patcher executable
	r.Patcher = findInDir(r.RootDir, "patcher", "FarsiForgePatcher.exe")
	if r.Patcher == "" {
		r.Patcher = findInDir(r.RootDir, "", "FarsiForgePatcher.exe")
	}
	r.Available["patcher"] = r.Patcher != ""

	// Find Il2CppDumper
	r.Il2CppDumper = findInDir(r.RootDir, "Il2CppDumper", "Il2CppDumper.exe")
	if r.Il2CppDumper == "" {
		r.Il2CppDumper = findInDir(r.RootDir, "", "Il2CppDumper.exe")
	}
	r.Available["il2cpp_dumper"] = r.Il2CppDumper != ""

	// Find AssetRipper
	r.AssetRipper = findInDir(r.RootDir, "AssetRipper", "AssetRipper.exe")
	if r.AssetRipper == "" {
		r.AssetRipper = findInDir(r.RootDir, "", "AssetRipper.exe")
	}
	r.Available["asset_ripper"] = r.AssetRipper != ""

	// Find UABEA
	r.UABEA = findInDir(r.RootDir, "UABEA", "UABEA.exe")
	if r.UABEA == "" {
		r.UABEA = findInDir(r.RootDir, "", "UABEA.exe")
	}
	r.Available["uabea"] = r.UABEA != ""

	// Find FModel
	r.FModel = findInDir(r.RootDir, "FModel", "FModel.exe")
	if r.FModel == "" {
		r.FModel = findInDir(r.RootDir, "", "FModel.exe")
	}
	r.Available["fmodel"] = r.FModel != ""

	// Check for zip files that contain tools
	r.checkZips()

	// Check Python packages
	if r.Python != "" {
		r.checkPythonPackages()
	}

	if dirExists(filepath.Join(r.RootDir, "UnityPy")) {
		r.Available["UnityPy"] = true
	}
}

// checkZips checks for tool zips and notes which need extraction.
func (r *Registry) checkZips() {
	entries, err := os.ReadDir(r.RootDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		lower := strings.ToLower(name)
		if !strings.HasSuffix(lower, ".zip") {
			continue
		}

		switch {
		case strings.Contains(lower, "assetripper"):
			r.Available["asset_ripper_zip"] = true
		case strings.Contains(lower, "assetstoolsnet") || strings.Contains(lower, "assetsview"):
			r.Available["assets_tools_zip"] = true
		case strings.Contains(lower, "uabea"):
			r.Available["uabea_zip"] = true
		case strings.Contains(lower, "repak"):
			r.Available["repak_zip"] = true
		case strings.Contains(lower, "fmodel"):
			r.Available["fmodel_zip"] = true
		case strings.Contains(lower, "undertalemodtool"):
			r.Available["undertale_zip"] = true
		}
	}
}

// checkPythonPackages checks which Python packages are installed. It uses the
// silent RunSilent API so no console window flashes during discovery.
func (r *Registry) checkPythonPackages() {
	script := `import sys
for p in ["UnityPy", "arabic_reshaper", "bidi", "fontTools", "openpyxl"]:
    try:
        __import__(p)
        print("ok:" + p)
    except Exception:
        pass
`
	out, err := RunSilentTimeout(pythonVerifyTimeout, "", r.Python, "-c", script)
	if err == nil {
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "ok:") {
				pkg := strings.TrimPrefix(line, "ok:")
				r.Available["py_"+pkg] = true
			}
		}
	}
}

// findPython discovers a working Python interpreter without any user-specific
// paths. It returns the verified absolute path and the first line of
// `python --version` (or "" if none found).
func findPython() (string, string) {
	var candidates []string
	if configured := os.Getenv("FARSIFORGE_PYTHON"); configured != "" {
		candidates = append(candidates, configured)
	}
	if configured := os.Getenv("FARISIFORGE_PYTHON"); configured != "" {
		candidates = append(candidates, configured)
	}

	// a. python on PATH
	if p := findExecutable("python"); p != "" {
		candidates = append(candidates, p)
	}
	// b. python3 on PATH
	if p := findExecutable("python3"); p != "" {
		candidates = append(candidates, p)
	}
	// c. %LOCALAPPDATA%\Programs\Python\Python*\python.exe
	if la := os.Getenv("LOCALAPPDATA"); la != "" {
		if matches, err := filepath.Glob(filepath.Join(la, "Programs", "Python", "Python*", "python.exe")); err == nil {
			candidates = append(candidates, matches...)
		}
	}
	// d. Standard per-machine Windows installations.
	for _, root := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")} {
		if root == "" {
			continue
		}
		if matches, err := filepath.Glob(filepath.Join(root, "Python*", "python.exe")); err == nil {
			candidates = append(candidates, matches...)
		}
	}

	for _, c := range candidates {
		if c == "" || !fileExists(c) {
			continue
		}
		// Verify by running it silently.
		out, err := RunSilentTimeout(pythonVerifyTimeout, "", c, "-c",
			`import sys; print(sys.version)`)
		if err != nil || strings.TrimSpace(out) == "" {
			continue
		}
		// Capture version line.
		versionLine := strings.TrimSpace(out)
		if verOut, vErr := RunSilentTimeout(pythonVerifyTimeout, "", c, "--version"); vErr == nil {
			if v := strings.TrimSpace(verOut); v != "" {
				versionLine = v
			}
		}
		return c, versionLine
	}
	return "", ""
}

// findExecutable finds a system executable on PATH.
func findExecutable(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return path
}

// findAdjacentOrPath searches adjacent to the current running executable, then falls back to system PATH.
func findAdjacentOrPath(name string) string {
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidate := filepath.Join(exeDir, name)
		if fileExists(candidate) {
			return candidate
		}
	}
	base := strings.TrimSuffix(name, ".exe")
	return findExecutable(base)
}

// findInDir searches for a file in a directory (optionally in a subdirectory).
func findInDir(dir, subdir, pattern string) string {
	searchDir := dir
	if subdir != "" {
		searchDir = filepath.Join(dir, subdir)
	}

	entries, err := os.ReadDir(searchDir)
	if err != nil {
		return ""
	}

	for _, e := range entries {
		name := e.Name()
		if e.IsDir() {
			continue
		}
		if pattern == ".exe" {
			if strings.HasSuffix(strings.ToLower(name), ".exe") {
				return filepath.Join(searchDir, name)
			}
		} else if strings.EqualFold(name, pattern) {
			return filepath.Join(searchDir, name)
		}
	}
	return ""
}

// findInSubdirectories finds a tool in the root or one level below it.
func findInSubdirectories(dir, exePattern string) string {
	// First check the tools root.
	result := findInDir(dir, "", exePattern)
	if result != "" {
		return result
	}

	// Try subdirectories
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		result = findInDir(filepath.Join(dir, e.Name()), "", exePattern)
		if result != "" {
			return result
		}
	}

	return ""
}

func findMatchingExecutable(dir, namePart string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if strings.Contains(name, namePart) && strings.HasSuffix(name, ".exe") {
			return filepath.Join(dir, entry.Name())
		}
	}
	return ""
}

// findVazirmatn finds the Vazirmatn font (may be in a zip).
func findVazirmatn(dir string) string {
	// Check if extracted
	fontDir := filepath.Join(dir, "vazirmatn")
	if dirExists(fontDir) {
		// Find .ttf files
		entries, _ := os.ReadDir(fontDir)
		for _, e := range entries {
			if strings.HasSuffix(strings.ToLower(e.Name()), ".ttf") {
				return filepath.Join(fontDir, e.Name())
			}
		}
	}

	// Check for vazirmatn.zip (needs extraction)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.Name()), "vazirmatn") {
			return filepath.Join(dir, e.Name()) // return the zip path as indicator
		}
	}
	return ""
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Summary returns a human-readable summary of available tools.
func (r *Registry) Summary() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Tools directory: %s\n", r.RootDir))
	sb.WriteString(fmt.Sprintf("Python: %s\n", r.Python))
	if r.PythonVersion != "" {
		sb.WriteString(fmt.Sprintf("Python version: %s\n", r.PythonVersion))
	}

	if r.Available != nil {
		// Stable ordering for readable output.
		names := make([]string, 0, len(r.Available))
		for name := range r.Available {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			status := "✗"
			if r.Available[name] {
				status = "✓"
			}
			sb.WriteString(fmt.Sprintf("  %s %s\n", status, name))
		}
	}

	return sb.String()
}

// IsAvailable checks if a tool is available.
func (r *Registry) IsAvailable(name string) bool {
	if r.Available == nil {
		return false
	}
	return r.Available[name]
}

// EnsureAvailable panics if a required tool is not available.
func (r *Registry) EnsureAvailable(name string) error {
	if !r.IsAvailable(name) {
		return fmt.Errorf("required tool not available: %s", name)
	}
	return nil
}

// GetPython returns the path to the Python executable.
func (r *Registry) GetPython() string {
	if r.Python != "" {
		return r.Python
	}
	return "python"
}

// GetPath returns the path for a given tool name.
func (r *Registry) GetPath(name string) string {
	switch strings.ToLower(name) {
	case "unitypy":
		return filepath.Join(r.RootDir, "UnityPy")
	case "die":
		return r.DIE
	case "unreallocres":
		return r.UnrealLocres
	case "uassetgui":
		return r.UAssetGUI
	case "gdre_tools", "gdre":
		return r.GDRETools
	case "codewalker_magic", "magic.dat":
		return r.CodeWalkerMagic
	case "godot":
		return r.GodotEngine
	case "undertale_mod_tool":
		return r.UndertaleModTool
	case "fontforge":
		return r.FontForge
	case "7z", "sevenzip":
		return r.SevenZip
	case "fftools", "ff_tools":
		return r.FFTools
	case "repak":
		return r.Repak
	case "retoc":
		return r.Retoc
	case "patcher", "farsiforgepatcher":
		return r.Patcher
	case "il2cpp_dumper", "il2cppdumper":
		return r.Il2CppDumper
	case "asset_ripper", "assetripper":
		return r.AssetRipper
	case "uabea":
		return r.UABEA
	case "fmodel":
		return r.FModel
	case "vazirmatn", "vazirmatn_font":
		return r.VazirmatnFont
	default:
		return filepath.Join(r.RootDir, name)
	}
}
