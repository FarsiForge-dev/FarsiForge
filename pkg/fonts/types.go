package fonts

// FontAssetType classifies the physical or engine representation of a game font.
type FontAssetType string

const (
	// FontAssetDirectFile is a standard loose font file (TTF, OTF, TTC, WOFF).
	FontAssetDirectFile FontAssetType = "direct_file"

	// FontAssetEngineResource is a compiled/serialized engine font (e.g. Unreal Font/FontFace, Unity Font/TextMeshPro, Godot FontFile).
	FontAssetEngineResource FontAssetType = "engine_resource"

	// FontAssetBitmapAtlas is a bitmap glyph sheet or BMFont (e.g. AngelCode .fnt + .png, custom texture atlas).
	FontAssetBitmapAtlas FontAssetType = "bitmap_atlas"

	// FontAssetCustomArchive is an engine-specific proprietary archive (e.g. FromSoftware BHD/BDT glyph atlas, SAGE font big).
	FontAssetCustomArchive FontAssetType = "custom_archive"
)

// GlyphCoverageReport details character and shaping readiness for Persian localization.
type GlyphCoverageReport struct {
	SupportsPersian       bool     `json:"supports_persian"`
	SupportsArabic        bool     `json:"supports_arabic"`
	SupportsPersianDigits bool     `json:"supports_persian_digits"`
	HasOpenTypeShaping    bool     `json:"has_opentype_shaping"` // True if GSUB features (init, medi, fina, isol) are present
	TotalTestedRunes      int      `json:"total_tested_runes"`
	SupportedRunesCount   int      `json:"supported_runes_count"`
	MissingRunes          []string `json:"missing_runes,omitempty"`
	CoveragePercent       float64  `json:"coverage_percent"`
}

// FontInventoryItem represents one detected font or glyph asset in the game installation.
type FontInventoryItem struct {
	Path              string               `json:"path"`                // Game-relative asset path
	Container         string               `json:"container,omitempty"`// Archive/container containing this font, if not loose
	ResourceID        string               `json:"resource_id,omitempty"`
	Engine            string               `json:"engine"`
	DetectedFormat    string               `json:"detected_format"`    // TTF, OTF, TTC, BMFont, TextMeshPro, etc.
	AssetType         FontAssetType        `json:"asset_type"`
	Family            string               `json:"family,omitempty"`
	Style             string               `json:"style,omitempty"`
	AtlasTexture      string               `json:"atlas_texture,omitempty"`
	Confidence        float64              `json:"confidence"`
	CanExport         bool                 `json:"can_export"`
	CanReplace        bool                 `json:"can_replace"`
	UnsupportedReason string               `json:"unsupported_reason,omitempty"`
	GlyphCoverage     *GlyphCoverageReport `json:"glyph_coverage,omitempty"`
	FileSize          int64                `json:"file_size"`
}

// FontInventoryReport is the complete result of a game font scan.
type FontInventoryReport struct {
	GameRoot            string              `json:"game_root"`
	Engine              string              `json:"engine"`
	TotalFontsFound     int                 `json:"total_fonts_found"`
	DirectFontFiles     int                 `json:"direct_font_files"`
	EngineResources     int                 `json:"engine_resources"`
	BitmapAtlases       int                 `json:"bitmap_atlases"`
	PersianReadyCount   int                 `json:"persian_ready_count"`
	Items               []FontInventoryItem `json:"items"`
	DiagnosticSummary   string              `json:"diagnostic_summary"`
	RecommendedAction   string              `json:"recommended_action"`
}

// FontReplacementResult records a staged font replacement operation.
type FontReplacementResult struct {
	TargetRelPath string `json:"target_rel_path"`
	StagedPath    string `json:"staged_path"`
	SourceSHA256  string `json:"source_sha256"`
	TargetSHA256  string `json:"target_sha256"`
	FontFamily    string `json:"font_family"`
}
