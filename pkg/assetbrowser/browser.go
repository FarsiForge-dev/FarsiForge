package assetbrowser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"farsiforge/pkg/core"
	"farsiforge/pkg/extract"
	"farsiforge/pkg/scanner"
)

// BrowseGameAssets walks the game directory, classifies containers and asset files,
// and returns a hierarchical tree view with provenance and editability contracts.
func BrowseGameAssets(gameRoot string, engine string) (*AssetBrowserView, error) {
	absRoot, err := filepath.Abs(gameRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid game root: %w", err)
	}
	info, err := os.Stat(absRoot)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("game root does not exist or is not a directory: %s", absRoot)
	}

	normEngine := strings.ToLower(strings.TrimSpace(engine))
	if normEngine == "" {
		normEngine = "generic"
	}

	allFiles, diag := scanner.WalkDirWithDiagnostics(absRoot, 10, func(p string) bool {
		// Include everything except temp working directories
		rel, err := filepath.Rel(absRoot, p)
		if err != nil {
			return false
		}
		slash := filepath.ToSlash(rel)
		return !strings.HasPrefix(slash, ".farsiforge") &&
			!strings.HasPrefix(slash, "WorkingDir") &&
			!strings.Contains(slash, "/out/")
	})

	view := &AssetBrowserView{
		GameRoot:    absRoot,
		Engine:      normEngine,
		TotalFiles:  len(allFiles),
		Diagnostics: diag,
	}

	// Classify each file into an AssetNode
	classifiedNodes := make([]*AssetNode, 0, len(allFiles))
	for _, fullPath := range allFiles {
		// Enforce root boundary
		rel, err := filepath.Rel(absRoot, fullPath)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		slashRel := filepath.ToSlash(rel)

		node := classifyFile(absRoot, fullPath, slashRel, normEngine)
		if node != nil {
			classifiedNodes = append(classifiedNodes, node)
			if node.Type == NodeContainer {
				view.TotalContainers++
			}
			if node.ParsingStatus == StatusParsed {
				view.TotalParsedResources++
			}
			view.TotalTextRecords += node.TextCount
		}
	}

	// Build folder hierarchy
	view.RootNodes = buildHierarchy(absRoot, classifiedNodes)

	return view, nil
}

// classifyFile analyzes a single file based on engine semantics and known format parsers.
func classifyFile(gameRoot, fullPath, slashRel, engine string) *AssetNode {
	fi, err := os.Stat(fullPath)
	if err != nil || fi.IsDir() {
		return nil
	}

	fileName := filepath.Base(slashRel)
	ext := strings.ToLower(filepath.Ext(fileName))
	size := fi.Size()

	node := &AssetNode{
		ID:            "asset:" + slashRel,
		Name:          fileName,
		RelPath:       slashRel,
		Format:        strings.TrimPrefix(ext, "."),
		Engine:        engine,
		SizeBytes:     size,
		Type:          NodeAssetFile,
		ParsingStatus: StatusNotParsed,
		Metadata:      make(map[string]string),
	}

	switch engine {
	case "unreal", "ue4", "ue5":
		classifyUnrealFile(node, fullPath, slashRel, ext)
	case "fromsoftware":
		classifyFromSoftwareFile(node, fullPath, slashRel, ext)
	case "godot":
		classifyGodotFile(node, fullPath, slashRel, ext)
	case "unity":
		classifyUnityFile(node, fullPath, slashRel, ext)
	default:
		classifyGenericFile(node, fullPath, slashRel, ext)
	}

	return node
}

func classifyUnrealFile(node *AssetNode, fullPath, slashRel, ext string) {
	switch ext {
	case ".pak":
		node.Type = NodeContainer
		node.Format = "pak"
		node.ParsingStatus = StatusOpaqueContainer
		node.CanExtract = true
		node.CanInject = false
		node.ReadOnly = true
		node.ReadOnlyReason = "آرشیو فشرده آنریل (.pak)؛ نیازمند بازسازی آرشیو جهت تزریق"
		node.Metadata["archive_type"] = "Unreal Pak v4/v5/v7"
	case ".utoc", ".ucas":
		node.Type = NodeContainer
		node.Format = "iostore"
		node.ParsingStatus = StatusOpaqueContainer
		node.CanExtract = true
		node.CanInject = false
		node.ReadOnly = true
		node.ReadOnlyReason = "کانتینر UE5 IoStore (.utoc/.ucas)؛ نیازمند استخراج کلید و کاتالوگ zen"
	case ".locres":
		node.Type = NodeAssetFile
		node.Format = "locres"
		node.ParsingStatus = StatusParsed
		node.CanExtract = true
		node.CanInject = true
		node.ReadOnly = false
		node.Metadata["resource_type"] = "Unreal Localization Resource (Loose)"
		// Estimate text count if readable
		node.TextCount = estimateLocresStrings(fullPath)
	case ".uexp":
		node.Type = NodeRawCandidate
		node.Format = "uexp"
		node.ParsingStatus = StatusRawHeuristic
		node.CanExtract = true
		node.CanInject = false
		node.ReadOnly = true
		node.ReadOnlyReason = "دارایی کامپایل‌شده بلوپرینت/ویجت (.uexp)؛ فاقد جدول رسمی locres (فقط‌خواندنی)"
	case ".uasset":
		node.Type = NodeAssetFile
		node.Format = "uasset"
		node.ParsingStatus = StatusUnsupported
		node.CanExtract = false
		node.CanInject = false
		node.ReadOnly = true
		node.ReadOnlyReason = "فایل هدر دارایی آنریل (.uasset)"
	default:
		classifyGenericFile(node, fullPath, slashRel, ext)
	}
}

func classifyFromSoftwareFile(node *AssetNode, fullPath, slashRel, ext string) {
	lower := strings.ToLower(slashRel)
	if strings.HasSuffix(lower, "data0.bhd") || strings.HasSuffix(lower, "data0.bdt") {
		node.Type = NodeContainer
		node.Format = "bhd5"
		node.ParsingStatus = StatusParsed
		node.CanExtract = true
		node.CanInject = false
		node.ReadOnly = true
		node.ReadOnlyReason = "آرشیو BHD5/BDT رمزنگاری‌شده FromSoftware؛ نوشتن مستقیم بدون آسیب به offsets پشتیبانی نمی‌شود"
		node.Metadata["archive_type"] = "FromSoftware BHD5/BDT Container"

		// Expand known message archives inside Data0
		if strings.HasSuffix(lower, "data0.bhd") {
			addFromSoftwareArchiveMembers(node, slashRel)
		}
		return
	}

	if strings.HasSuffix(lower, ".msgbnd.dcx") || strings.HasSuffix(lower, ".msgbnd") ||
		strings.HasSuffix(lower, ".bnd4") || strings.HasSuffix(lower, ".bnd3") {
		node.Type = NodeContainer
		node.Format = "bnd"
		node.ParsingStatus = StatusParsed
		node.CanExtract = true
		node.CanInject = true
		node.ReadOnly = false
		node.Metadata["archive_type"] = "FromSoftware Message Binder"

		// Try parsing inner FMG files
		addLooseBinderMembers(node, fullPath, slashRel)
		return
	}

	if ext == ".fmg" {
		node.Type = NodeAssetFile
		node.Format = "fmg"
		node.ParsingStatus = StatusParsed
		node.CanExtract = true
		node.CanInject = true
		node.ReadOnly = false
		node.TextCount = countFmgStrings(fullPath)
		node.Metadata["resource_type"] = "FromSoftware Text Table (FMG)"
		return
	}

	classifyGenericFile(node, fullPath, slashRel, ext)
}

func addFromSoftwareArchiveMembers(containerNode *AssetNode, containerRel string) {
	// Virtual catalog of standard FromSoftware message binders known to exist in Data0
	knownBins := []struct {
		member   string
		desc     string
		subgroup string
	}{
		{"/msg/engus/item.msgbnd.dcx", "متون و توضیحات آیتم‌ها (انگلیسی)", "item"},
		{"/msg/engus/menu.msgbnd.dcx", "رابط کاربری و پیام‌های سیستم (انگلیسی)", "ui"},
		{"/msg/engus/talk.msgbnd.dcx", "مکالمات و دیالوگ‌های داستانی (انگلیسی)", "dialogue"},
		{"/msg/engus/system.msgbnd.dcx", "اعلان‌ها و تنظیمات سیستمی (انگلیسی)", "system"},
	}

	for _, kb := range knownBins {
		child := &AssetNode{
			ID:             fmt.Sprintf("%s:%s", containerNode.ID, kb.member),
			Name:           filepath.Base(kb.member),
			RelPath:        containerRel + kb.member,
			Container:      containerRel,
			MemberPath:     kb.member,
			Type:           NodeParsedResource,
			Format:         "dcx/fmg",
			Engine:         "fromsoftware",
			ParsingStatus:  StatusParsed,
			CanExtract:     true,
			CanInject:      false,
			ReadOnly:       true,
			ReadOnlyReason: "عضو آرشیو محافظت‌شده BHD5؛ استخراج متون فعال، تزریق مستقیم غیرفعال",
			Metadata: map[string]string{
				"description": kb.desc,
				"category":    kb.subgroup,
			},
		}
		containerNode.Children = append(containerNode.Children, child)
	}
}

func addLooseBinderMembers(containerNode *AssetNode, fullPath, slashRel string) {
	raw, err := os.ReadFile(fullPath)
	if err != nil {
		return
	}
	bndData := raw
	if strings.HasSuffix(strings.ToLower(fullPath), ".dcx") {
		decomp, err := extract.DecompressDCX(raw)
		if err == nil {
			bndData = decomp
		}
	}

	var innerFiles []extract.BND4File
	if len(bndData) >= 4 && string(bndData[:4]) == "BND3" {
		if bf, err := extract.ParseBND3(bndData); err == nil {
			innerFiles = bf
		}
	} else {
		if bf, err := extract.ParseBND4(bndData); err == nil {
			innerFiles = bf
		}
	}

	for _, inner := range innerFiles {
		innerName := filepath.Base(inner.Name)
		textCount := 0
		if strings.HasSuffix(strings.ToLower(innerName), ".fmg") {
			if rows, err := extract.ParseFMG(inner.Data); err == nil {
				textCount = len(rows)
			}
		}

		child := &AssetNode{
			ID:            fmt.Sprintf("%s:%s", containerNode.ID, innerName),
			Name:          innerName,
			RelPath:       slashRel + "/" + innerName,
			Container:     slashRel,
			MemberPath:    innerName,
			Type:          NodeParsedResource,
			Format:        "fmg",
			Engine:        "fromsoftware",
			SizeBytes:     int64(len(inner.Data)),
			TextCount:     textCount,
			ParsingStatus: StatusParsed,
			CanExtract:    true,
			CanInject:     true,
			ReadOnly:      false,
		}
		containerNode.Children = append(containerNode.Children, child)
		containerNode.TextCount += textCount
	}
}

func classifyGodotFile(node *AssetNode, fullPath, slashRel, ext string) {
	switch ext {
	case ".pck":
		node.Type = NodeContainer
		node.Format = "pck"
		node.ParsingStatus = StatusParsed
		node.CanExtract = true
		node.CanInject = true
		node.ReadOnly = false
		node.Metadata["archive_type"] = "Godot PCK Package"
	case ".translation":
		node.Type = NodeAssetFile
		node.Format = "translation"
		node.ParsingStatus = StatusParsed
		node.CanExtract = true
		node.CanInject = true
		node.ReadOnly = false
		node.Metadata["resource_type"] = "Godot Compiled Translation Dictionary"
	case ".tres", ".res":
		node.Type = NodeAssetFile
		node.Format = "tres"
		node.ParsingStatus = StatusParsed
		node.CanExtract = true
		node.CanInject = true
		node.ReadOnly = false
		node.Metadata["resource_type"] = "Godot Resource File"
	case ".tscn", ".scn":
		node.Type = NodeAssetFile
		node.Format = "scene"
		node.ParsingStatus = StatusParsed
		node.CanExtract = true
		node.CanInject = true
		node.ReadOnly = false
		node.Metadata["resource_type"] = "Godot Scene"
	default:
		classifyGenericFile(node, fullPath, slashRel, ext)
	}
}

func classifyUnityFile(node *AssetNode, fullPath, slashRel, ext string) {
	switch ext {
	case ".assets", ".bundle":
		node.Type = NodeContainer
		node.Format = strings.TrimPrefix(ext, ".")
		node.ParsingStatus = StatusParsed
		node.CanExtract = true
		node.CanInject = true
		node.ReadOnly = false
		node.Metadata["archive_type"] = "Unity Serialized AssetBundle"
	case ".resS", ".resource":
		node.Type = NodeAssetFile
		node.Format = "resource"
		node.ParsingStatus = StatusUnsupported
		node.CanExtract = false
		node.CanInject = false
		node.ReadOnly = true
		node.ReadOnlyReason = "جریان داده باینری بزرگ یونیتی (Raw Stream)"
	default:
		classifyGenericFile(node, fullPath, slashRel, ext)
	}
}

// InspectGameAsset inspects serialized resources and objects within a single game file or container.
func InspectGameAsset(ctx context.Context, gameRoot, relPath, engine string, page, pageSize int, reg core.ToolRegistry) (*AssetInspectionResult, error) {
	normEngine := strings.ToLower(strings.TrimSpace(engine))
	switch normEngine {
	case "unity":
		return InspectUnityAsset(ctx, gameRoot, relPath, page, pageSize, reg)
	default:
		cleanRel := filepath.Clean(filepath.ToSlash(relPath))
		fullPath := filepath.Join(gameRoot, filepath.FromSlash(cleanRel))
		fi, err := os.Stat(fullPath)
		if err != nil || fi.IsDir() {
			return nil, fmt.Errorf("file not found: %s", relPath)
		}
		ext := strings.ToLower(filepath.Ext(cleanRel))
		return &AssetInspectionResult{
			FileRelPath:  cleanRel,
			TotalObjects: 0,
			Page:         1,
			PageSize:     pageSize,
			HasMore:      false,
			Objects:      nil,
			Error:        fmt.Sprintf("Engine %q does not have a compiled object inspector for %s files", engine, ext),
		}, nil
	}
}

func classifyGenericFile(node *AssetNode, fullPath, slashRel, ext string) {
	switch ext {
	case ".json", ".xml", ".csv", ".tsv", ".yaml", ".yml", ".ini", ".txt", ".po", ".strings":
		node.Type = NodeAssetFile
		node.Format = strings.TrimPrefix(ext, ".")
		node.ParsingStatus = StatusParsed
		node.CanExtract = true
		node.CanInject = true
		node.ReadOnly = false
		node.Metadata["resource_type"] = "Structured Text File"
	case ".exe", ".dll", ".so", ".dylib":
		node.Type = NodeAssetFile
		node.Format = "binary"
		node.ParsingStatus = StatusUnsupported
		node.CanExtract = false
		node.CanInject = false
		node.ReadOnly = true
		node.ReadOnlyReason = "فایل باینری کد اجرایی"
	case ".png", ".jpg", ".jpeg", ".tga", ".dds", ".wav", ".ogg", ".mp3", ".bik":
		node.Type = NodeAssetFile
		node.Format = "media"
		node.ParsingStatus = StatusUnsupported
		node.CanExtract = false
		node.CanInject = false
		node.ReadOnly = true
		node.ReadOnlyReason = "فایل مدیا (صوت یا تصویر)"
	default:
		node.Type = NodeAssetFile
		node.Format = strings.TrimPrefix(ext, ".")
		node.ParsingStatus = StatusNotParsed
		node.CanExtract = false
		node.CanInject = false
		node.ReadOnly = true
		node.ReadOnlyReason = "فرمت باینری بدون پارسر اختصاصی"
	}
}

func countFmgStrings(path string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	rows, err := extract.ParseFMG(b)
	if err != nil {
		return 0
	}
	count := 0
	for _, r := range rows {
		if strings.TrimSpace(r.Text) != "" {
			count++
		}
	}
	return count
}

func estimateLocresStrings(path string) int {
	fi, err := os.Stat(path)
	if err != nil || fi.Size() < 32 {
		return 0
	}
	// Heuristic approximation from binary size
	return int(fi.Size() / 64)
}

// buildHierarchy arranges classified flat nodes into a folder/container tree.
func buildHierarchy(gameRoot string, nodes []*AssetNode) []*AssetNode {
	dirMap := make(map[string]*AssetNode)
	var rootNodes []*AssetNode

	// Sort nodes by relative path
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].RelPath < nodes[j].RelPath
	})

	for _, node := range nodes {
		parts := strings.Split(node.RelPath, "/")
		if len(parts) == 1 {
			// File is at root
			rootNodes = append(rootNodes, node)
			continue
		}

		// Ensure parent directories exist
		var currentDirNode *AssetNode
		currentDirPath := ""

		for i := 0; i < len(parts)-1; i++ {
			if currentDirPath == "" {
				currentDirPath = parts[i]
			} else {
				currentDirPath += "/" + parts[i]
			}

			dirNode, exists := dirMap[currentDirPath]
			if !exists {
				dirNode = &AssetNode{
					ID:            "folder:" + currentDirPath,
					Name:          parts[i],
					RelPath:       currentDirPath,
					Type:          NodeFolder,
					Format:        "folder",
					ParsingStatus: StatusNotParsed,
					Children:      make([]*AssetNode, 0),
				}
				dirMap[currentDirPath] = dirNode

				if currentDirNode == nil {
					rootNodes = append(rootNodes, dirNode)
				} else {
					currentDirNode.Children = append(currentDirNode.Children, dirNode)
				}
			}
			currentDirNode = dirNode
		}

		if currentDirNode != nil {
			currentDirNode.Children = append(currentDirNode.Children, node)
			currentDirNode.SizeBytes += node.SizeBytes
			currentDirNode.TextCount += node.TextCount
		} else {
			rootNodes = append(rootNodes, node)
		}
	}

	return rootNodes
}
