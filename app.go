package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"farsiforge/pkg/assetbrowser"
	"farsiforge/pkg/core"
	"farsiforge/pkg/detection"
	"farsiforge/pkg/exchange"
	"farsiforge/pkg/extract"
	"farsiforge/pkg/fonts"
	"farsiforge/pkg/inject"
	"farsiforge/pkg/installer"
	"farsiforge/pkg/textfilter"
	"farsiforge/pkg/tools"
	"farsiforge/pkg/validate"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx      context.Context
	registry *detection.Registry
	toolReg  *tools.Registry
}

// NewApp creates a new App application struct. It uses the portable tool
// registry (NewRegistry with empty string → auto-resolved Tools directory,
// or user-configured tools directory if set in config).
func NewApp() *App {
	toolsDir := ""
	if cfg, err := core.LoadConfig(core.ConfigPath()); err == nil && cfg.ToolsDir != "" {
		toolsDir = cfg.ToolsDir
	}
	toolReg, _ := tools.NewRegistry(toolsDir)
	return &App{
		registry: detection.DefaultRegistry(),
		toolReg:  toolReg,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ProgressEvent communicates pipeline progress and stage descriptions to the frontend.
type ProgressEvent struct {
	Stage   string  `json:"stage"`
	StageFa string  `json:"stage_fa"`
	Percent float64 `json:"percent"`
	Detail  string  `json:"detail"`
}

func (a *App) emitProgress(stage, stageFa string, percent float64, detail string) {
	if a.ctx != nil {
		wailsruntime.EventsEmit(a.ctx, "pipeline:progress", ProgressEvent{
			Stage:   stage,
			StageFa: stageFa,
			Percent: percent,
			Detail:  detail,
		})
	}
}

// ── Settings ────────────────────────────────────────────────────────

// Settings is the JSON-serializable settings struct exposed to the frontend.
// It mirrors the user-tunable subset of core.Config.
type Settings struct {
	ToolsDir        string              `json:"tools_dir"`
	OutputDir       string              `json:"output_dir"`
	GameSearchPaths []string            `json:"game_search_paths"`
	DefaultPersian  core.PersianOptions `json:"default_persian_opts"`
	DefaultAuthor   string              `json:"default_author"`
	DefaultLanguage string              `json:"default_language"`
	LogLevel        string              `json:"log_level"`
}

// GetSettings loads and returns the current application settings.
func (a *App) GetSettings() (*Settings, error) {
	cfg, err := core.LoadConfig(core.ConfigPath())
	if err != nil {
		// LoadConfig returns DefaultConfig on missing file, so this is only
		// a parse error.
		return nil, fmt.Errorf("load config: %v", err)
	}
	return &Settings{
		ToolsDir:        cfg.ToolsDir,
		OutputDir:       cfg.OutputDir,
		GameSearchPaths: cfg.GameSearchPaths,
		DefaultPersian:  cfg.DefaultPersianOpts,
		DefaultAuthor:   cfg.DefaultAuthor,
		DefaultLanguage: cfg.DefaultLanguage,
		LogLevel:        cfg.LogLevel,
	}, nil
}

// SaveSettings persists the given settings via the core config API.
// Validates custom paths and tool registry before persisting.
func (a *App) SaveSettings(s *Settings) error {
	if s == nil {
		return fmt.Errorf("settings are nil")
	}
	cfg, err := core.LoadConfig(core.ConfigPath())
	if err != nil {
		return fmt.Errorf("load config: %v", err)
	}

	// Validate custom tools directory and ensure registry initializes cleanly
	if s.ToolsDir != "" && s.ToolsDir != cfg.ToolsDir {
		newReg, err := tools.NewRegistry(s.ToolsDir)
		if err != nil {
			return fmt.Errorf("failed to initialize tool registry for path %q: %w", s.ToolsDir, err)
		}
		a.toolReg = newReg
	}

	// Validate output directory
	if s.OutputDir != "" {
		if err := os.MkdirAll(s.OutputDir, 0755); err != nil {
			return fmt.Errorf("invalid output directory %q: %w", s.OutputDir, err)
		}
	}

	cfg.ToolsDir = s.ToolsDir
	cfg.OutputDir = s.OutputDir
	cfg.GameSearchPaths = s.GameSearchPaths
	cfg.DefaultPersianOpts = s.DefaultPersian
	cfg.DefaultAuthor = s.DefaultAuthor
	cfg.DefaultLanguage = s.DefaultLanguage
	cfg.LogLevel = s.LogLevel
	if err := core.SaveConfig(cfg, core.ConfigPath()); err != nil {
		return fmt.Errorf("save config: %v", err)
	}
	return nil
}

// ── Tool status ─────────────────────────────────────────────────────

// ToolStatus reports the presence and version of the tools FarsiForge relies
// on. All fields are JSON-serializable for the Wails frontend.
type ToolStatus struct {
	ToolsDir      string          `json:"tools_dir"`
	Python        string          `json:"python"`
	PythonVersion string          `json:"python_version"`
	PythonOK      bool            `json:"python_ok"`
	UnityPy       bool            `json:"unity_py"`
	FFTools       string          `json:"ff_tools"`
	FFToolsOK     bool            `json:"ff_tools_ok"`
	Repak         string          `json:"repak"`
	RepakOK       bool            `json:"repak_ok"`
	Retoc         string          `json:"retoc"`
	RetocOK       bool            `json:"retoc_ok"`
	GDRETools     string          `json:"gdre_tools"`
	GDREToolsOK   bool            `json:"gdre_tools_ok"`
	PatcherExe    string          `json:"patcher_exe"`
	PatcherOK     bool            `json:"patcher_ok"`
	Available     map[string]bool `json:"available"`
}

// GetToolStatus reports the presence/version of external tools (Python, UnityPy,
// fftools, repak, retoc, gdre_tools, patcher).
func (a *App) GetToolStatus() *ToolStatus {
	reg := a.toolReg
	if reg == nil {
		reg, _ = tools.NewRegistry("")
		a.toolReg = reg
	}
	patcherExe := reg.Patcher
	if patcherExe == "" {
		patcherExe = filepath.Join(reg.RootDir, "patcher", "FarsiForgePatcher.exe")
	}
	status := &ToolStatus{
		ToolsDir:      reg.RootDir,
		Python:        reg.Python,
		PythonVersion: reg.PythonVersion,
		PythonOK:      reg.IsAvailable("python"),
		UnityPy:       reg.IsAvailable("py_UnityPy"),
		FFTools:       reg.FFTools,
		FFToolsOK:     reg.IsAvailable("fftools"),
		Repak:         reg.Repak,
		RepakOK:       reg.IsAvailable("repak"),
		Retoc:         reg.Retoc,
		RetocOK:       reg.IsAvailable("retoc"),
		GDRETools:     reg.GDRETools,
		GDREToolsOK:   reg.IsAvailable("gdre_tools"),
		PatcherExe:    patcherExe,
		PatcherOK:     fileExists(patcherExe),
		Available:     reg.Available,
	}
	return status
}

// GetDetailedToolDiagnostics returns a rich diagnostic report covering all helper
// tools, architecture, version probes, and licenses.
func (a *App) GetDetailedToolDiagnostics() *tools.DiagnosticReport {
	if a.toolReg == nil {
		a.toolReg, _ = tools.NewRegistry("")
	}
	return tools.GenerateDiagnosticReport(a.toolReg)
}

// CheckOperationPrerequisites verifies whether all required external tools for
// an engine and operation (extract, inject, rebuild_archive, build_patch) are available.
func (a *App) CheckOperationPrerequisites(engine, operation string) tools.OperationCheckResult {
	if a.toolReg == nil {
		a.toolReg, _ = tools.NewRegistry("")
	}
	return tools.CheckOperationRequirements(a.toolReg, engine, operation)
}

// RefreshTools rescans the tools directory and updates the active tool registry.
func (a *App) RefreshTools() (*tools.DiagnosticReport, error) {
	toolsDir := ""
	if a.toolReg != nil {
		toolsDir = a.toolReg.RootDir
	}
	newReg, err := tools.NewRegistry(toolsDir)
	if err != nil {
		return nil, fmt.Errorf("refresh tools failed: %w", err)
	}
	a.toolReg = newReg
	return tools.GenerateDiagnosticReport(a.toolReg), nil
}

// GetEngineCapabilities returns the unified support matrix of all game engines.
func (a *App) GetEngineCapabilities() []core.EngineCapability {
	return core.AllEngineCapabilities()
}

// GetEngineCapability returns the capability details for a specific engine.
func (a *App) GetEngineCapability(engine string) core.EngineCapability {
	return core.FindEngineCapability(engine)
}

// Allowed Python dependencies for FarsiForge toolchain
var allowedPythonPackages = map[string]string{
	"unitypy": "UnityPy",
	"pillow":  "Pillow",
	"brotli":  "brotli",
}

// InstallPythonPackage runs `pip install <name>` silently via pkg/tools for approved packages
// and returns the combined output/error text.
func (a *App) InstallPythonPackage(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("package name is required")
	}
	canonicalPkg, allowed := allowedPythonPackages[strings.ToLower(name)]
	if !allowed {
		return "", fmt.Errorf("package %q is not an approved FarsiForge dependency; only approved packages (UnityPy) may be installed", name)
	}

	reg := a.toolReg
	if reg == nil || reg.Python == "" {
		return "", fmt.Errorf("Python is not available; install Python first")
	}

	// Use `python -m pip install` so it targets the same interpreter.
	out, err := tools.RunSilent(a.ctx, "", reg.Python, "-m", "pip", "install", canonicalPkg)
	if err != nil {
		return out, fmt.Errorf("pip install failed: %v\n%s", err, out)
	}
	// Refresh the registry so newly installed packages are detected.
	reg, _ = tools.NewRegistry(reg.RootDir)
	a.toolReg = reg
	return out, nil
}

// ── Detection / extraction / injection ──────────────────────────────

// DetectEngine runs the engine detection.
func (a *App) DetectEngine(path string) (*core.GameInfo, error) {
	gameRoot, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve game directory: %w", err)
	}
	reg := a.registry
	if reg == nil {
		reg = detection.DefaultRegistry()
	}
	res, err := reg.Detect(gameRoot)
	if err != nil {
		return nil, fmt.Errorf("detection failed: %v", err)
	}

	dataPath := ""
	if len(res.DataPaths) > 0 {
		dataPath = res.DataPaths[0]
	}
	gameName := strings.TrimSpace(res.GameName)
	if gameName == "" {
		gameName = filepath.Base(gameRoot)
	}

	return &core.GameInfo{
		Engine:     res.Engine,
		Backend:    res.Backend,
		Version:    res.Version,
		GameName:   gameName,
		GameExe:    res.GameExe,
		GameRoot:   gameRoot,
		DataPath:   dataPath,
		Confidence: res.Confidence,
		Evidence:   res.Evidence,
		Metadata:   res.Metadata,
	}, nil
}

// Extract runs the extraction pipeline.
func (a *App) Extract(info *core.GameInfo) (int, error) {
	if info == nil {
		return 0, fmt.Errorf("game information is required")
	}
	a.emitProgress("extract", "آماده‌سازی ساختار پروژه و دایرکتوری‌ها...", 15, info.GameRoot)
	previous, loadErr := core.LoadGameProject(info.GameRoot)
	if loadErr != nil && !errors.Is(loadErr, os.ErrNotExist) {
		return 0, fmt.Errorf("load existing project: %w", loadErr)
	}
	gameName := strings.TrimSpace(info.GameName)
	if gameName == "" {
		gameName = filepath.Base(filepath.Clean(info.GameRoot))
	}
	proj, projectPath, err := core.NewGameProject(gameName+" Localization", info.GameRoot, info.Engine)
	if err != nil {
		return 0, err
	}
	proj.GameName = gameName
	proj.GameExe = info.GameExe
	proj.Backend = info.Backend
	proj.Version = info.Version

	a.emitProgress("extract", fmt.Sprintf("در حال استخراج متون با ماژول %s...", info.Engine), 40, "خواندن بسته‌ها و فایل‌های متنی")
	if err := extract.Run(a.ctx, info, proj, a.toolReg); err != nil {
		a.emitProgress("extract", "خطا در استخراج متون", 0, err.Error())
		return 0, fmt.Errorf("extraction failed: %v", err)
	}

	a.emitProgress("extract", "دسته‌بندی و فیلتر دقیق کاندیداها (Classification)...", 70, "تحلیل ساختار دیالوگ، رابط کاربری و امنیت نوشتن")
	stats := textfilter.ClassifyProject(proj)

	if previous != nil {
		a.emitProgress("extract", "تطبیق و ادغام ترجمه‌های پیشین...", 85, "ادغام ترجمه‌ها")
		proj.MergeTranslations(previous)
	}
	a.emitProgress("extract", "ذخیره‌سازی دیتابیس پروژه...", 95, "نوشتن فایل‌های پروژه")
	if err := proj.Save(projectPath); err != nil {
		return 0, fmt.Errorf("failed to save project: %v", err)
	}

	a.emitProgress("extract", "استخراج متون با موفقیت به پایان رسید.", 100, fmt.Sprintf("%d رشته (پذیرفته: %d، با اطمینان: %d)", len(proj.Entries), stats.AcceptedCount, stats.HighConfidenceCount))
	return len(proj.Entries), nil
}

// ClassifyProjectEntries runs the precision candidate classification and filtering pipeline on the loaded project.
func (a *App) ClassifyProjectEntries(gameRoot string) (*core.ExtractionStats, error) {
	proj, err := core.LoadGameProject(gameRoot)
	if err != nil {
		return nil, fmt.Errorf("load project: %w", err)
	}
	stats := textfilter.ClassifyProject(proj)
	projectPath, err := core.ProjectFilePath(gameRoot)
	if err != nil {
		return &stats, err
	}
	if err := proj.Save(projectPath); err != nil {
		return &stats, fmt.Errorf("save project after classification: %w", err)
	}
	return &stats, nil
}

// UpdateEntryCandidate updates the candidate class and/or category of an entry.
func (a *App) UpdateEntryCandidate(gameRoot, entryID, candidateClass, category string) error {
	proj, err := core.LoadGameProject(gameRoot)
	if err != nil {
		return fmt.Errorf("load project: %w", err)
	}
	found := false
	for i := range proj.Entries {
		if proj.Entries[i].ID == entryID {
			if candidateClass != "" {
				proj.Entries[i].CandidateClass = candidateClass
			}
			if category != "" {
				proj.Entries[i].Category = category
			}
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("entry not found: %s", entryID)
	}
	projectPath, err := core.ProjectFilePath(gameRoot)
	if err != nil {
		return err
	}
	return proj.Save(projectPath)
}

// Inject runs the injection pipeline.
func (a *App) Inject(info *core.GameInfo) (int, error) {
	if info == nil {
		return 0, fmt.Errorf("game information is required")
	}
	a.emitProgress("inject", "خواندن پروژه و اعتبارسنجی عبارات...", 15, "بررسی متغیرها و تگ‌ها")
	proj, err := core.LoadGameProject(info.GameRoot)
	if err != nil {
		return 0, fmt.Errorf("load project: %w", err)
	}
	if proj.Engine != info.Engine {
		return 0, fmt.Errorf("project engine %q does not match detected engine %q", proj.Engine, info.Engine)
	}
	projectPath, err := core.ProjectFilePath(info.GameRoot)
	if err != nil {
		return 0, err
	}
	proj.ModifiedFiles = nil
	proj.ModifiedFileHashes = nil
	if err := proj.Save(projectPath); err != nil {
		return 0, fmt.Errorf("clear previous staged injection state: %w", err)
	}

	// Validate project translations before injecting: block on hard errors.
	validator := validate.New()
	valErrs := validator.Validate(proj.Entries)
	hardErrors := validate.FilterBySeverity(valErrs, core.SevError)
	if len(hardErrors) > 0 {
		var sample []string
		for i, he := range hardErrors {
			if i >= 5 {
				sample = append(sample, fmt.Sprintf("... and %d more hard error(s)", len(hardErrors)-5))
				break
			}
			sample = append(sample, fmt.Sprintf("[%s] %s: %s", he.EntryID, he.Type, he.Message))
		}
		a.emitProgress("inject", "تزریق به دلیل خطاهای اعتبارسنجی متوقف شد", 0, fmt.Sprintf("%d خطا", len(hardErrors)))
		return 0, fmt.Errorf("injection blocked: %d hard validation error(s) found:\n%s", len(hardErrors), strings.Join(sample, "\n"))
	}

	opts := proj.PersianOpts
	if opts == (core.PersianOptions{}) {
		opts = core.DefaultPersianOptions()
	}

	a.emitProgress("inject", "اعمال تغییرات روی کپی‌های موقت (Staging)...", 50, "پردازش متون و راست‌چین‌سازی")
	modified, err := inject.Run(a.ctx, info, proj, a.toolReg, opts)
	if err != nil {
		a.emitProgress("inject", "خطا در تزریق متون", 0, err.Error())
		return 0, fmt.Errorf("injection failed: %v", err)
	}
	a.emitProgress("inject", "ذخیره‌سازی اطلاعات پچ و هش فایل‌ها...", 90, "نهایی‌سازی استیجینگ")
	if err := proj.Save(projectPath); err != nil {
		return 0, fmt.Errorf("save injection results: %w", err)
	}

	a.emitProgress("inject", "تزریق با موفقیت انجام شد.", 100, fmt.Sprintf("%d فایل آماده پچ شد", len(modified)))
	return len(modified), nil
}

// ValidateProject runs the validation pipeline on all project entries.
func (a *App) ValidateProject(gameRoot string) ([]core.ValidationError, error) {
	proj, err := core.LoadGameProject(gameRoot)
	if err != nil {
		return nil, fmt.Errorf("load project: %w", err)
	}
	validator := validate.New()
	return validator.Validate(proj.Entries), nil
}

// SavePersianOptions persists the user's Persian processing options for the project.
func (a *App) SavePersianOptions(gameRoot string, opts core.PersianOptions) error {
	proj, err := core.LoadGameProject(gameRoot)
	if err != nil {
		return fmt.Errorf("load project: %w", err)
	}
	proj.PersianOpts = opts
	projectPath, err := core.ProjectFilePath(gameRoot)
	if err != nil {
		return err
	}
	return proj.Save(projectPath)
}

// LaunchGame starts the game executable silently (no console window flash).
func (a *App) LaunchGame(gameRoot, exeName string) error {
	if gameRoot == "" {
		return fmt.Errorf("launch: game root cannot be empty")
	}
	if exeName == "" {
		return fmt.Errorf("launch: executable name cannot be empty")
	}
	cleanExe := filepath.Clean(strings.ReplaceAll(exeName, "\\", "/"))
	if cleanExe == "." || cleanExe == ".." || strings.HasPrefix(cleanExe, "../") || filepath.IsAbs(cleanExe) {
		return fmt.Errorf("launch: invalid executable path %q", exeName)
	}
	exePath := filepath.Join(gameRoot, cleanExe)
	absRoot, err := filepath.Abs(gameRoot)
	if err != nil {
		return fmt.Errorf("launch: resolve game root: %w", err)
	}
	absExe, err := filepath.Abs(exePath)
	if err != nil {
		return fmt.Errorf("launch: resolve executable: %w", err)
	}
	rel, err := filepath.Rel(absRoot, absExe)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("launch: executable %q escapes game directory", exeName)
	}
	info, err := os.Stat(absExe)
	if err != nil {
		return fmt.Errorf("game executable not found: %s", absExe)
	}
	if info.IsDir() {
		return fmt.Errorf("launch target is a directory, not an executable: %s", absExe)
	}
	// Use RunSilent to avoid a console window flash on Windows.
	_, err = tools.RunSilent(a.ctx, absRoot, absExe)
	if err != nil {
		return fmt.Errorf("failed to launch game: %v", err)
	}
	return nil
}

// SelectDirectory opens a directory selection dialog
func (a *App) SelectDirectory() (string, error) {
	return wailsruntime.OpenDirectoryDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "انتخاب پوشه بازی",
	})
}

// SelectSaveFile opens a file save dialog for exporting.
func (a *App) SelectSaveFile(defaultFilename string) (string, error) {
	return wailsruntime.SaveFileDialog(a.ctx, wailsruntime.SaveDialogOptions{
		Title:           "ذخیره فایل ترجمه",
		DefaultFilename: defaultFilename,
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Excel Files (*.xlsx)", Pattern: "*.xlsx"},
			{DisplayName: "CSV Files (*.csv)", Pattern: "*.csv"},
		},
	})
}

// SelectOpenFile opens a file selection dialog for importing.
func (a *App) SelectOpenFile() (string, error) {
	return wailsruntime.OpenFileDialog(a.ctx, wailsruntime.OpenDialogOptions{
		Title: "انتخاب فایل ترجمه برای ورود",
		Filters: []wailsruntime.FileFilter{
			{DisplayName: "Supported Exchange Files (*.xlsx; *.csv)", Pattern: "*.xlsx;*.csv"},
			{DisplayName: "Excel Files (*.xlsx)", Pattern: "*.xlsx"},
			{DisplayName: "CSV Files (*.csv)", Pattern: "*.csv"},
		},
	})
}

// ExportTranslations exports the project translations to CSV or XLSX.
func (a *App) ExportTranslations(gameRoot, outputPath string) error {
	proj, err := core.LoadGameProject(gameRoot)
	if err != nil {
		return fmt.Errorf("load project: %w", err)
	}
	return exchange.Export(proj, outputPath)
}

// ImportTranslations imports translations from CSV or XLSX atomically.
func (a *App) ImportTranslations(gameRoot, inputPath string) (*exchange.ImportResult, error) {
	proj, err := core.LoadGameProject(gameRoot)
	if err != nil {
		return nil, fmt.Errorf("load project: %w", err)
	}
	res, err := exchange.Import(proj, inputPath)
	if err != nil {
		return &res, err
	}
	projectPath, err := core.ProjectFilePath(gameRoot)
	if err != nil {
		return &res, err
	}
	if err := proj.Save(projectPath); err != nil {
		return &res, fmt.Errorf("save project after import: %w", err)
	}
	return &res, nil
}

// GetProject loads the project
func (a *App) GetProject(gameRoot string) (*core.Project, error) {
	return core.LoadGameProject(gameRoot)
}

// SearchEntries loads the project saved under gameRoot and returns the
// entries matching the query (case-insensitive substring across ID, Source,
// Translation, File, and Notes) and the optional status filter (a status
// value or "qa" for entries with QA notes; "" disables filtering).
func (a *App) SearchEntries(gameRoot, query, status string) ([]core.StringEntry, error) {
	proj, err := a.GetProject(gameRoot)
	if err != nil {
		return nil, err
	}
	return proj.SearchEntries(query, status), nil
}

// SaveTranslations saves translations
func (a *App) SaveTranslations(gameRoot string, entries []core.StringEntry) error {
	proj, err := core.LoadGameProject(gameRoot)
	if err != nil {
		return err
	}
	projectIndexes := make(map[string]int, len(proj.Entries))
	for i, entry := range proj.Entries {
		if entry.ID == "" {
			return fmt.Errorf("project entry at index %d is missing its stable ID", i)
		}
		if _, exists := projectIndexes[entry.ID]; exists {
			return fmt.Errorf("project contains duplicate entry ID %q", entry.ID)
		}
		projectIndexes[entry.ID] = i
	}
	byID := make(map[string]core.StringEntry, len(entries))
	for _, entry := range entries {
		if entry.ID == "" {
			return fmt.Errorf("translation entry is missing its stable ID")
		}
		if _, exists := byID[entry.ID]; exists {
			return fmt.Errorf("duplicate translation entry ID %q", entry.ID)
		}
		if _, exists := projectIndexes[entry.ID]; !exists {
			return fmt.Errorf("translation entry %q is not part of the current project; reload before saving", entry.ID)
		}
		switch entry.Status {
		case core.StatusUntranslated, core.StatusTranslated, core.StatusApproved, core.StatusSkipped:
		default:
			return fmt.Errorf("invalid translation status %q for entry %q", entry.Status, entry.ID)
		}
		byID[entry.ID] = entry
	}
	for id, incoming := range byID {
		i := projectIndexes[id]
		proj.Entries[i].Translation = incoming.Translation
		proj.Entries[i].Status = incoming.Status
		if proj.Entries[i].Translation == "" && proj.Entries[i].Status != core.StatusSkipped {
			proj.Entries[i].Status = core.StatusUntranslated
		} else if proj.Entries[i].Translation != "" && proj.Entries[i].Status == core.StatusUntranslated {
			proj.Entries[i].Status = core.StatusTranslated
		}
		proj.Entries[i].Notes = textfilter.QANotes(textfilter.QA(proj.Entries[i].Source, proj.Entries[i].Translation))
	}
	projectPath, err := core.ProjectFilePath(gameRoot)
	if err != nil {
		return err
	}
	return proj.Save(projectPath)
}

// ── Patcher build ───────────────────────────────────────────────────

// BuildPatcherResult describes the produced patch package.
type BuildPatcherResult struct {
	OutputDir   string `json:"output_dir"`
	PatchFile   string `json:"patch_file"`
	PatcherExe  string `json:"patcher_exe"`
	TargetCount int    `json:"target_count"`
}

// BuildPatcher builds the final FFP1 patch package using pkg/ffpatch and
// stages FarsiForgePatcher.exe alongside it. The modified files are taken
// from the project's ModifiedFiles list; their patched content is read from
// the project working directory.
func (a *App) BuildPatcher(gameRoot string, credits string) (*BuildPatcherResult, error) {
	proj, err := core.LoadGameProject(gameRoot)
	if err != nil {
		return nil, fmt.Errorf("load project: %v", err)
	}

	// Resolve the patcher exe from the tool registry.
	reg := a.toolReg
	if reg == nil {
		reg, _ = tools.NewRegistry("")
		a.toolReg = reg
	}
	patcherExe := reg.GetPath("patcher")
	if patcherExe == "" || !fileExists(patcherExe) {
		patcherExe = filepath.Join(reg.RootDir, "patcher", "FarsiForgePatcher.exe")
	}

	// Collect patch targets: each modified file's patched version lives in
	// the project working directory (injectors write outputs there). We map
	// the game-relative path to the patched file in workDir.
	workDir := proj.WorkingDir
	if workDir == "" {
		return nil, fmt.Errorf("project working directory is not configured")
	}

	var targets []installer.PatchTarget
	for _, mf := range proj.ModifiedFiles {
		gamePath := filepath.ToSlash(mf)
		if filepath.IsAbs(mf) || strings.Contains(gamePath, "../") || gamePath == ".." {
			return nil, fmt.Errorf("invalid modified game path %q", mf)
		}
		patchedFile := filepath.Join(workDir, "out", filepath.FromSlash(gamePath))
		if !fileExists(patchedFile) {
			return nil, fmt.Errorf("staged patched file missing for %q: %s", mf, patchedFile)
		}
		originalHash := proj.ModifiedFileHashes[mf]
		if originalHash == "" {
			return nil, fmt.Errorf("staged source hash missing for %q; inject again before building the patch", mf)
		}
		targets = append(targets, installer.PatchTarget{
			GamePath:       gamePath,
			PatchedFile:    patchedFile,
			OriginalSHA256: originalHash,
		})
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("no patched files found to build a patch (run injection first)")
	}

	a.emitProgress("patch", "جمع‌آوری فایل‌های آماده پچ...", 15, fmt.Sprintf("%d تارگت", len(targets)))
	fontFile := ""
	if proj.FontPath != "" && fileExists(proj.FontPath) {
		fontFile = proj.FontPath
	}

	outputDir := filepath.Join(proj.ProjectDir, "dist")
	cfg := installer.BuildConfig{
		GameRoot:    gameRoot,
		Targets:     targets,
		PatcherExe:  patcherExe,
		GameExe:     proj.GameExe,
		Engine:      proj.Engine,
		PatchName:   proj.GameName + " — فارسی‌ساز",
		Description: "FarsiForge Persian localization patch",
		Author:      credits,
		OutputDir:   outputDir,
		FontFile:    fontFile,
	}

	a.emitProgress("patch", "بسته‌بندی باینری پچ FFP1 و انتقال فایل‌ها...", 50, "نوشتن در دایرکتوری موقت")
	res, err := installer.Build(cfg)
	if err != nil {
		a.emitProgress("patch", "خطا در بسته‌بندی پچ", 0, err.Error())
		return nil, fmt.Errorf("build patcher: %v", err)
	}

	a.emitProgress("patch", "بسته‌بندی پچ با موفقیت انجام شد.", 100, res.OutputDir)
	return &BuildPatcherResult{
		OutputDir:   res.OutputDir,
		PatchFile:   res.PatchFile,
		PatcherExe:  res.PatcherExe,
		TargetCount: res.TargetCount,
	}, nil
}

// ScanGameFonts runs the game font inventory scan and returns detected direct fonts,
// engine font assets, and bitmap atlases.
func (a *App) ScanGameFonts(gameRoot string) (*fonts.FontInventoryReport, error) {
	proj, err := core.LoadGameProject(gameRoot)
	engine := ""
	if err == nil && proj != nil {
		engine = proj.Engine
	}
	if engine == "" {
		if info, err := a.DetectEngine(gameRoot); err == nil && info != nil {
			engine = info.Engine
		}
	}
	return fonts.ScanGameFonts(gameRoot, engine)
}

// StageFontReplacement stages a custom replacement font under the project's working directory,
// capturing hashes and registering it for patch building.
func (a *App) StageFontReplacement(gameRoot, targetFontRelPath, customFontPath string) (*fonts.FontReplacementResult, error) {
	proj, err := core.LoadGameProject(gameRoot)
	if err != nil {
		return nil, fmt.Errorf("load project: %w", err)
	}
	if proj.WorkingDir == "" {
		return nil, fmt.Errorf("project working directory is not initialized")
	}

	res, err := fonts.StageFontReplacement(gameRoot, proj.WorkingDir, targetFontRelPath, customFontPath)
	if err != nil {
		return nil, err
	}

	// Register staged font in project state
	proj.FontPath = res.StagedPath
	if proj.ModifiedFileHashes == nil {
		proj.ModifiedFileHashes = make(map[string]string)
	}
	proj.ModifiedFileHashes[res.TargetRelPath] = res.SourceSHA256

	alreadyTracked := false
	for _, mf := range proj.ModifiedFiles {
		if filepath.ToSlash(mf) == res.TargetRelPath {
			alreadyTracked = true
			break
		}
	}
	if !alreadyTracked {
		proj.ModifiedFiles = append(proj.ModifiedFiles, res.TargetRelPath)
	}

	projectPath, err := core.ProjectFilePath(gameRoot)
	if err != nil {
		return nil, err
	}
	if err := proj.Save(projectPath); err != nil {
		return nil, fmt.Errorf("save project after font replacement: %w", err)
	}

	return res, nil
}

// BrowseGameAssets returns a hierarchical tree view of game assets, containers, and resources.
func (a *App) BrowseGameAssets(gameRoot string) (*assetbrowser.AssetBrowserView, error) {
	proj, err := core.LoadGameProject(gameRoot)
	engine := ""
	if err == nil && proj != nil {
		engine = proj.Engine
	}
	if engine == "" {
		if info, err := a.DetectEngine(gameRoot); err == nil && info != nil {
			engine = info.Engine
		}
	}
	return assetbrowser.BrowseGameAssets(gameRoot, engine)
}

// InspectGameAsset inspects serialized resources and objects within a single game file or container.
func (a *App) InspectGameAsset(gameRoot, relPath string, page, pageSize int) (*assetbrowser.AssetInspectionResult, error) {
	proj, err := core.LoadGameProject(gameRoot)
	engine := ""
	if err == nil && proj != nil {
		engine = proj.Engine
	}
	if engine == "" {
		if info, err := a.DetectEngine(gameRoot); err == nil && info != nil {
			engine = info.Engine
		}
	}
	return assetbrowser.InspectGameAsset(a.ctx, gameRoot, relPath, engine, page, pageSize, a.toolReg)
}

// ── helpers ─────────────────────────────────────────────────────────

// fileExists is a local helper (app package) to avoid importing scanner here.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
