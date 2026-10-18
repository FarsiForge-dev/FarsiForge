// Package assetbrowser provides hierarchical game asset exploration and source provenance.
package assetbrowser

import (
	"farsiforge/pkg/core"
	"farsiforge/pkg/scanner"
)

// AssetNodeType represents the structural kind of an asset node.
type AssetNodeType string

const (
	NodeFolder          AssetNodeType = "folder"
	NodeContainer       AssetNodeType = "container_archive"
	NodeAssetFile       AssetNodeType = "asset_file"
	NodeParsedResource  AssetNodeType = "parsed_resource"
	NodeRawCandidate    AssetNodeType = "raw_candidate"
)

// ParsingStatus describes how deeply the asset has been parsed.
type ParsingStatus string

const (
	StatusParsed          ParsingStatus = "parsed"
	StatusUnsupported     ParsingStatus = "unsupported"
	StatusOpaqueContainer ParsingStatus = "opaque_container"
	StatusRawHeuristic    ParsingStatus = "raw_heuristic"
	StatusNotParsed       ParsingStatus = "not_parsed"
)

// AssetNode is a node in the hierarchical asset browser tree.
type AssetNode struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	RelPath        string            `json:"rel_path"`
	Container      string            `json:"container,omitempty"`
	MemberPath     string            `json:"member_path,omitempty"`
	Type           AssetNodeType     `json:"type"`
	Format         string            `json:"format"`
	Engine         string            `json:"engine"`
	SizeBytes      int64             `json:"size_bytes"`
	TextCount      int               `json:"text_count"`
	ParsingStatus  ParsingStatus     `json:"parsing_status"`
	CanExtract     bool              `json:"can_extract"`
	CanInject      bool              `json:"can_inject"`
	ReadOnly       bool              `json:"read_only"`
	ReadOnlyReason string            `json:"read_only_reason,omitempty"`
	Children       []*AssetNode      `json:"children,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	SampleEntries  []core.StringEntry`json:"sample_entries,omitempty"`
}

// AssetProperty describes one text-bearing property or field in a serialized object.
type AssetProperty struct {
	Path           string `json:"path"`
	FieldName      string `json:"field_name,omitempty"`
	Value          string `json:"value"`
	Category       string `json:"category,omitempty"`
	CanInject      bool   `json:"can_inject"`
	ReadOnly       bool   `json:"read_only"`
	ReadOnlyReason string `json:"read_only_reason,omitempty"`
}

// CompiledObjectNode represents a serialized object inside a compiled asset container.
type CompiledObjectNode struct {
	PathID         int64           `json:"path_id"`
	Type           string          `json:"type"`
	Name           string          `json:"name"`
	SizeBytes      int64           `json:"size_bytes"`
	TextBearing    bool            `json:"text_bearing"`
	ParsingStatus  ParsingStatus   `json:"parsing_status"`
	CanInject      bool            `json:"can_inject"`
	ReadOnly       bool            `json:"read_only"`
	ReadOnlyReason string          `json:"read_only_reason,omitempty"`
	Format         string          `json:"format,omitempty"`
	Properties     []AssetProperty `json:"properties,omitempty"`
}

// AssetInspectionResult holds paginated serialized objects for a single compiled asset file.
type AssetInspectionResult struct {
	FileRelPath  string               `json:"file_rel_path"`
	TotalObjects int                  `json:"total_objects"`
	Page         int                  `json:"page"`
	PageSize     int                  `json:"page_size"`
	HasMore      bool                 `json:"has_more"`
	Objects      []CompiledObjectNode `json:"objects"`
	Error        string               `json:"error,omitempty"`
}

// AssetBrowserView represents the complete asset tree and summary metrics for a game directory.
type AssetBrowserView struct {
	GameRoot             string                  `json:"game_root"`
	Engine               string                  `json:"engine"`
	TotalFiles           int                     `json:"total_files"`
	TotalContainers      int                     `json:"total_containers"`
	TotalParsedResources int                     `json:"total_parsed_resources"`
	TotalTextRecords     int                     `json:"total_text_records"`
	RootNodes            []*AssetNode            `json:"root_nodes"`
	Diagnostics          scanner.ScanDiagnostics `json:"diagnostics"`
}

