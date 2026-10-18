"use client";

import { useEffect, useMemo, useState } from "react";
import { motion, AnimatePresence } from "framer-motion";
import { 
  FolderSearch, 
  Settings2, 
  Box, 
  Rocket, 
  CheckCircle2, 
  XCircle,
  Cpu, 
  FolderOpen,
  ArrowRight,
  Database,
  Languages,
  Code2,
  Wrench,
  Download,
  Upload,
  Save,
  ChevronLeft,
  ChevronRight,
  AlertTriangle,
  RotateCcw,
  Copy,
  Check,
  X,
  Info,
  Lock,
  Filter,
  Sparkles,
  Type,
  Archive,
  Folder,
  FolderTree,
  FileText,
  ChevronDown,
  Search,
  FileCode,
  Layers,
  Eye,
  ExternalLink,
  RefreshCw
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Progress } from "@/components/ui/progress";
import { Badge } from "@/components/ui/badge";
import { 
  SelectDirectory, 
  DetectEngine, 
  Extract, 
  GetProject, 
  SaveTranslations, 
  Inject, 
  BuildPatcher, 
  GetToolStatus,
  ExportTranslations,
  ImportTranslations,
  SelectSaveFile,
  SelectOpenFile,
  SavePersianOptions,
  ValidateProject,
  GetEngineCapabilities,
  GetEngineCapability,
  GetSettings,
  SaveSettings,
  UpdateEntryCandidate,
  ClassifyProjectEntries,
  ScanGameFonts,
  StageFontReplacement,
  BrowseGameAssets,
  InspectGameAsset
} from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import { core, main, fonts, assetbrowser } from "../../wailsjs/go/models";

type Step = "detect" | "extract" | "translate" | "inject" | "patch" | "success";

export default function FarsiForgeDashboard() {
  const [currentStep, setCurrentStep] = useState<Step>("detect");
  const [dirPath, setDirPath] = useState("");
  const [isProcessing, setIsProcessing] = useState(false);
  const [progress, setProgress] = useState(0);
  const [stageName, setStageName] = useState("");
  const [stageDetail, setStageDetail] = useState("");
  const [gameInfo, setGameInfo] = useState<core.GameInfo | null>(null);
  const [translations, setTranslations] = useState<core.StringEntry[]>([]);
  const [patchResult, setPatchResult] = useState<Awaited<ReturnType<typeof BuildPatcher>> | null>(null);
  const [patchCredits, setPatchCredits] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [currentPage, setCurrentPage] = useState(1);
  const [toolStatus, setToolStatus] = useState<main.ToolStatus | null>(null);
  const [engineCaps, setEngineCaps] = useState<core.EngineCapability[]>([]);
  const [isSaving, setIsSaving] = useState(false);
  const [notification, setNotification] = useState<{
    type: "success" | "error" | "info" | "warning";
    title: string;
    message: string;
    diagnostic?: string;
  } | null>(null);
  const [copiedDiag, setCopiedDiag] = useState(false);
  const [showSettingsModal, setShowSettingsModal] = useState(false);
  const [settingsData, setSettingsData] = useState<any>(null);
  const [fontReport, setFontReport] = useState<fonts.FontInventoryReport | null>(null);
  const [isScanningFonts, setIsScanningFonts] = useState(false);
  const [stagedFonts, setStagedFonts] = useState<Record<string, fonts.FontReplacementResult>>({});
  const [showFontModal, setShowFontModal] = useState(false);
  const [assetView, setAssetView] = useState<assetbrowser.AssetBrowserView | null>(null);
  const [isBrowsingAssets, setIsBrowsingAssets] = useState(false);
  const [showAssetBrowserModal, setShowAssetBrowserModal] = useState(false);
  const [assetFilter, setAssetFilter] = useState<string>("all");
  const [assetSearch, setAssetSearch] = useState<string>("");
  const [expandedNodes, setExpandedNodes] = useState<Record<string, boolean>>({});
  const [selectedAssetNode, setSelectedAssetNode] = useState<assetbrowser.AssetNode | null>(null);
  const [persianOpts, setPersianOpts] = useState({
    rtl: true,
    shaping: true,
    digits: true,
    vazirmatn: true,
  });

  // WinRAR-style Archive Explorer state
  const [selectedArchiveFile, setSelectedArchiveFile] = useState<string>("all");
  const [selectedEntryId, setSelectedEntryId] = useState<string | null>(null);
  const [activeArchiveTab, setActiveArchiveTab] = useState<"all" | "dialogue" | "ui" | "quest">("all");
  const [archiveSearch, setArchiveSearch] = useState<string>("");

  useEffect(() => {
    GetToolStatus()
      .then((status) => setToolStatus(status))
      .catch((err) => console.error("GetToolStatus error:", err));

    GetEngineCapabilities()
      .then((caps) => setEngineCaps(caps))
      .catch((err) => console.error("GetEngineCapabilities error:", err));

    const cancelProgress = EventsOn("pipeline:progress", (evt: any) => {
      if (evt && typeof evt.percent === "number") {
        setProgress(evt.percent);
        if (evt.stage_fa) setStageName(evt.stage_fa);
        if (evt.detail) setStageDetail(evt.detail);
      }
    });

    return () => {
      if (cancelProgress) cancelProgress();
    };
  }, []);

  const currentCap = useMemo(() => {
    if (!gameInfo?.engine) return null;
    const match = engineCaps.find(c => c.engine.toLowerCase() === gameInfo.engine.toLowerCase());
    if (match) return match;
    return {
      engine: gameInfo.engine,
      display_name: gameInfo.engine,
      can_detect: true,
      can_extract: true,
      can_inject: true,
      archive_rebuild: false,
      font_support: false,
      patch_install: true,
      status: "custom",
      extraction_notes: "موتور اختصاصی یا شناسایی‌شده بر مبنای قوانین عمومی",
      injection_notes: "تزریق بر مبنای فایل‌های متنی پشتیبانی می‌شود",
      required_tools: []
    } as core.EngineCapability;
  }, [gameInfo, engineCaps]);

  const engineInfo = {
    engine: gameInfo ? (currentCap?.display_name || gameInfo.engine) : "---",
    backend: gameInfo ? gameInfo.backend : "---",
    confidence: gameInfo ? (gameInfo.confidence * 100).toFixed(0) + "%" : "---",
    files: translations.length > 0 
      ? `${translations.length} متن` 
      : (gameInfo?.evidence?.length ? `${gameInfo.evidence.length} نشانه شناسایی` : "---")
  };

  const isDetectionOnly = currentCap ? !currentCap.can_extract : false;
  const isExtractOnly = currentCap ? (currentCap.can_extract && !currentCap.can_inject) : false;

  const [workbenchView, setWorkbenchView] = useState<"candidates" | "explorer">("candidates");
  const [candidateClassFilter, setCandidateClassFilter] = useState("verified");
  const [categoryFilter, setCategoryFilter] = useState("all");

  const [inspectedAssetPath, setInspectedAssetPath] = useState<string>("");
  const [assetInspection, setAssetInspection] = useState<assetbrowser.AssetInspectionResult | null>(null);
  const [isLoadingAssetInspection, setIsLoadingAssetInspection] = useState(false);
  const [inspectionPage, setInspectionPage] = useState(1);
  const [expandedObjectIds, setExpandedObjectIds] = useState<Record<string, boolean>>({});
  const [inspectedFilterTextOnly, setInspectedFilterTextOnly] = useState(false);
  const [inspectedObjectSearch, setInspectedObjectSearch] = useState("");

  const classificationCounts = useMemo(() => {
    let verified = 0, review = 0, raw = 0, tech = 0, readOnly = 0;
    for (const t of translations) {
      if (t.read_only) {
        readOnly++;
      } else if (t.candidate_class === "high-confidence") {
        verified++;
      }
      if (t.candidate_class === "needs-review") review++;
      if (t.candidate_class === "raw-heuristic") raw++;
      if (t.candidate_class === "technical") tech++;
    }
    return { verified, review, raw, tech, readOnly, total: translations.length };
  }, [translations]);

  const handleUpdateCandidate = async (entryID: string, newClass?: string, newCategory?: string) => {
    if (!gameInfo?.game_root) return;
    try {
      await UpdateEntryCandidate(gameInfo.game_root, entryID, newClass || "", newCategory || "");
      setTranslations((prev) =>
        prev.map((e) =>
          e.id === entryID
            ? {
                ...e,
                ...(newClass ? { candidate_class: newClass } : {}),
                ...(newCategory ? { category: newCategory } : {}),
              }
            : e
        )
      );
    } catch (err) {
      console.error("Failed to update candidate:", err);
    }
  };

  // Group translations by file/container into an archive file tree (WinRAR-like)
  const archiveFiles = useMemo(() => {
    const fileMap: Record<string, {
      file: string;
      name: string;
      dir: string;
      total: number;
      dialogues: number;
      translated: number;
      categories: Record<string, number>;
    }> = {};

    for (const t of translations) {
      const f = t.file || "Other";
      if (!fileMap[f]) {
        const normalized = f.replace(/\\/g, "/");
        const parts = normalized.split("/");
        const name = parts[parts.length - 1];
        const dir = parts.slice(0, -1).join("/") || "Root";
        fileMap[f] = {
          file: f,
          name,
          dir,
          total: 0,
          dialogues: 0,
          translated: 0,
          categories: {},
        };
      }
      fileMap[f].total++;
      // Genuine dialogue evidence: category is dialogue or explicit dialogue structure in path
      if (t.category === "dialogue" || (t.path && t.path.toLowerCase().includes("dialogue"))) {
        fileMap[f].dialogues++;
      }
      if (t.translation && t.translation.trim() !== "") {
        fileMap[f].translated++;
      }
      const cat = t.category || "unclassified";
      fileMap[f].categories[cat] = (fileMap[f].categories[cat] || 0) + 1;
    }

    return Object.values(fileMap).sort((a, b) => b.total - a.total);
  }, [translations]);

  // Filtered entry list for the translate step: case-insensitive search
  // across id/source/translation/file/notes plus status, candidate class, and category filters.
  const filteredEntries = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    return translations
      .map((item, idx) => ({ item, idx }))
      .filter(({ item }) => {
        // Filter by archive file selected in the left WinRAR tree (exact or directory prefix)
        if (selectedArchiveFile !== "all") {
          const normItem = item.file.replace(/\\/g, "/");
          const normSelected = selectedArchiveFile.replace(/\\/g, "/");
          if (normItem !== normSelected && !normItem.startsWith(normSelected + "/")) {
            return false;
          }
        }

        // Quick archive tab filter (dialogue, ui, quest)
        // Genuine dialogue requires category === "dialogue" or explicit dialogue path context (never blindly TextAsset)
        if (activeArchiveTab === "dialogue") {
          const isDial = item.category === "dialogue" || (item.path && item.path.toLowerCase().includes("dialogue"));
          if (!isDial) return false;
        } else if (activeArchiveTab === "ui") {
          if (item.category !== "ui") return false;
        } else if (activeArchiveTab === "quest") {
          if (item.category !== "quest" && item.category !== "item") return false;
        }

        if (statusFilter === "qa") {
          if (!(item.notes || "").trim()) return false;
        } else if (statusFilter !== "all" && item.status !== statusFilter) {
          return false;
        }

        // Truthful candidate classification filter
        if (candidateClassFilter === "verified") {
          // Must satisfy BOTH conditions: high-confidence linguistic/structural evidence AND verified write-back path
          if (item.read_only || item.candidate_class !== "high-confidence") return false;
        } else if (candidateClassFilter === "review") {
          if (item.candidate_class !== "needs-review") return false;
        } else if (candidateClassFilter === "raw") {
          if (item.candidate_class !== "raw-heuristic") return false;
        } else if (candidateClassFilter === "read-only") {
          if (!item.read_only) return false;
        } else if (candidateClassFilter !== "all") {
          const itemClass = item.candidate_class || "high-confidence";
          if (itemClass !== candidateClassFilter) return false;
        }

        if (categoryFilter !== "all") {
          const itemCat = item.category || "unclassified";
          if (itemCat !== categoryFilter) return false;
        }

        if (q) {
          const hay = [
            item.id,
            item.source,
            item.translation,
            item.file,
            item.notes,
            item.category,
            item.candidate_class,
            item.read_only_reason
          ]
            .filter(Boolean)
            .join("\n")
            .toLowerCase();
          if (!hay.includes(q)) return false;
        }
        return true;
      });
  }, [translations, searchQuery, statusFilter, candidateClassFilter, categoryFilter, selectedArchiveFile, activeArchiveTab]);

  // Reset page when search or filter changes
  useEffect(() => {
    setCurrentPage(1);
  }, [searchQuery, statusFilter, candidateClassFilter, categoryFilter, selectedArchiveFile, activeArchiveTab]);

  const qaIssuesCount = useMemo(() => {
    return translations.filter((t) => (t.notes || "").trim() !== "").length;
  }, [translations]);

  // Paginate filtered entries (50 per page) so huge projects (100k+ entries) remain fully accessible and smooth.
  const PAGE_SIZE = 50;
  const totalPages = Math.max(1, Math.ceil(filteredEntries.length / PAGE_SIZE));
  const visibleEntries = useMemo(() => {
    const start = (currentPage - 1) * PAGE_SIZE;
    return filteredEntries.slice(start, start + PAGE_SIZE);
  }, [filteredEntries, currentPage]);

  const activeEntry = useMemo(() => {
    if (selectedEntryId) {
      const found = translations.find((t) => t.id === selectedEntryId);
      if (found) return found;
    }
    return visibleEntries.length > 0 ? visibleEntries[0].item : null;
  }, [translations, selectedEntryId, visibleEntries]);

  const filteredArchiveFiles = useMemo(() => {
    if (!archiveSearch.trim()) return archiveFiles;
    const s = archiveSearch.toLowerCase();
    return archiveFiles.filter(
      (a) =>
        a.name.toLowerCase().includes(s) ||
        a.dir.toLowerCase().includes(s) ||
        a.file.toLowerCase().includes(s)
    );
  }, [archiveFiles, archiveSearch]);

  const handleSelectPrev = () => {
    if (!activeEntry || filteredEntries.length === 0) return;
    const idx = filteredEntries.findIndex((e) => e.item.id === activeEntry.id);
    if (idx > 0) {
      const prevEntry = filteredEntries[idx - 1].item;
      setSelectedEntryId(prevEntry.id);
      const targetPage = Math.floor((idx - 1) / PAGE_SIZE) + 1;
      if (targetPage !== currentPage) setCurrentPage(targetPage);
    }
  };

  const handleSelectNext = () => {
    if (!activeEntry || filteredEntries.length === 0) return;
    const idx = filteredEntries.findIndex((e) => e.item.id === activeEntry.id);
    if (idx >= 0 && idx < filteredEntries.length - 1) {
      const nextEntry = filteredEntries[idx + 1].item;
      setSelectedEntryId(nextEntry.id);
      const targetPage = Math.floor((idx + 1) / PAGE_SIZE) + 1;
      if (targetPage !== currentPage) setCurrentPage(targetPage);
    }
  };

  const handleUpdateTranslation = (entryID: string, newText: string) => {
    setTranslations((prev) =>
      prev.map((t) =>
        t.id === entryID
          ? {
              ...t,
              translation: newText,
              status: newText.trim() !== "" ? (t.status === "approved" ? "approved" : "translated") : "untranslated",
            }
          : t
      )
    );
  };

  const handleInspectCompiledAsset = async (relPath: string, page = 1) => {
    const root = gameInfo?.game_root || dirPath;
    if (!root || !relPath) return;
    setIsLoadingAssetInspection(true);
    setInspectedAssetPath(relPath);
    setInspectionPage(page);
    try {
      const res = await InspectGameAsset(root, relPath, page, 50);
      setAssetInspection(res);
      if (res?.objects) {
        const toExpand: Record<string, boolean> = {};
        for (const obj of res.objects) {
          if (obj.text_bearing) {
            toExpand[String(obj.path_id)] = true;
          }
        }
        setExpandedObjectIds((prev) => ({ ...prev, ...toExpand }));
      }
    } catch (e) {
      console.error("InspectGameAsset error:", e);
      setNotification({
        type: "error",
        title: "خطا در بررسی ساختار باینری فایل کامپایل‌شده",
        message: String(e),
      });
    } finally {
      setIsLoadingAssetInspection(false);
    }
  };

  const handleNavigateToAssetExplorer = async (item: core.StringEntry) => {
    if (!item.file) return;
    setWorkbenchView("explorer");
    if (!assetView) {
      await handleBrowseAssets(false);
    }
    await handleInspectCompiledAsset(item.file, 1);
    if (item.path) {
      setInspectedObjectSearch(item.path);
    }
  };

  const handleNavigateToCandidate = (filePath: string, propPath?: string) => {
    setWorkbenchView("candidates");
    setSelectedArchiveFile(filePath);
    if (propPath) {
      setSearchQuery(propPath);
    }
  };

  const renderAssetTree = (nodes?: assetbrowser.AssetNode[], depth = 0, onNodeClick?: (node: assetbrowser.AssetNode) => void): any => {
    if (!nodes || nodes.length === 0) return null;
    return nodes
      .filter((n) => {
        if (assetSearch) {
          const s = assetSearch.toLowerCase();
          const match = n.name.toLowerCase().includes(s) || n.rel_path.toLowerCase().includes(s);
          if (!match && (!n.children || n.children.length === 0)) return false;
        }
        if (assetFilter === "containers") {
          return n.type === "container_archive" || (n.children && n.children.length > 0);
        }
        if (assetFilter === "extractable") {
          return n.can_extract || n.text_count > 0 || (n.children && n.children.length > 0);
        }
        if (assetFilter === "readonly") {
          return n.read_only || (n.children && n.children.length > 0);
        }
        return true;
      })
      .map((node) => {
        const isExpanded = expandedNodes[node.id] ?? (depth < 2);
        const isSelected = selectedAssetNode?.id === node.id;
        const hasChildren = node.children && node.children.length > 0;

        return (
          <div key={node.id} className="space-y-0.5">
            <div
              onClick={() => {
                setSelectedAssetNode(node);
                if (onNodeClick) {
                  onNodeClick(node);
                } else if (node.type !== "folder") {
                  handleInspectCompiledAsset(node.rel_path, 1);
                }
              }}
              className={`flex items-center justify-between gap-2 px-2 py-1 rounded cursor-pointer text-xs transition-colors ${
                isSelected 
                  ? "bg-indigo-600/30 text-white border border-indigo-500/40" 
                  : "hover:bg-zinc-800/60 text-zinc-300"
              }`}
              style={{ marginRight: `${depth * 14}px` }}
            >
              <div className="flex items-center gap-1.5 truncate">
                {hasChildren ? (
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      setExpandedNodes((prev) => ({ ...prev, [node.id]: !isExpanded }));
                    }}
                    className="p-0.5 text-zinc-500 hover:text-white"
                  >
                    {isExpanded ? "▼" : "◀"}
                  </button>
                ) : (
                  <span className="w-3" />
                )}
                {node.type === "folder" ? (
                  <FolderOpen className="w-3.5 h-3.5 text-indigo-400 shrink-0" />
                ) : node.type === "container_archive" ? (
                  <Box className="w-3.5 h-3.5 text-purple-400 shrink-0" />
                ) : node.type === "parsed_resource" ? (
                  <Database className="w-3.5 h-3.5 text-emerald-400 shrink-0" />
                ) : (
                  <Code2 className="w-3.5 h-3.5 text-sky-400 shrink-0" />
                )}
                <span className="truncate font-medium">{node.name}</span>
                {node.format && node.type !== "folder" && (
                  <span className="text-[10px] text-zinc-500 uppercase font-mono">
                    .{node.format}
                  </span>
                )}
              </div>

              <div className="flex items-center gap-1 shrink-0">
                {node.text_count > 0 && (
                  <Badge variant="outline" className="text-[10px] px-1.5 py-0 bg-emerald-500/10 text-emerald-400 border-emerald-500/20">
                    {node.text_count} متن
                  </Badge>
                )}
                {node.read_only ? (
                  <span title={node.read_only_reason}><Lock className="w-3 h-3 text-amber-400" /></span>
                ) : node.can_inject ? (
                  <span title="قابل تزریق"><Check className="w-3 h-3 text-emerald-400" /></span>
                ) : null}
              </div>
            </div>

            {hasChildren && isExpanded && (
              <div>{renderAssetTree(node.children, depth + 1, onNodeClick)}</div>
            )}
          </div>
        );
      });
  };

  const handleTogglePersian = async (key: string) => {
    const next = { ...persianOpts, [key]: !persianOpts[key as keyof typeof persianOpts] };
    setPersianOpts(next);
    if (gameInfo?.game_root) {
      try {
        await SavePersianOptions(gameInfo.game_root, {
          bidi_reorder: next.rtl,
          reshape: next.shaping,
          persian_digits: next.digits,
          fix_yeh: true,
          convert_punct: true,
          drop_diacritics: false,
        } as any);
      } catch (e) {
        console.error("SavePersianOptions error:", e);
      }
    }
  };

  const openSettings = async () => {
    try {
      const s = await GetSettings();
      setSettingsData(s);
      setShowSettingsModal(true);
    } catch (err) {
      setNotification({
        type: "error",
        title: "خطا در بارگذاری تنظیمات",
        message: String(err),
      });
    }
  };

  const handleSaveSettings = async () => {
    if (!settingsData) return;
    try {
      await SaveSettings(settingsData);
      setShowSettingsModal(false);
      setNotification({
        type: "success",
        title: "تنظیمات ذخیره شد",
        message: "تنظیمات پیکربندی برنامه با موفقیت ذخیره شد.",
      });
      const status = await GetToolStatus();
      setToolStatus(status);
    } catch (err) {
      setNotification({
        type: "error",
        title: "خطا در ذخیره تنظیمات",
        message: String(err),
      });
    }
  };

  const handleScanFonts = async () => {
    const targetDir = gameInfo?.game_root || dirPath;
    if (!targetDir) {
      setNotification({
        type: "warning",
        title: "مسیر بازی مشخص نیست",
        message: "لطفاً ابتدا مسیر بازی را انتخاب کنید تا فونت‌ها بررسی شوند.",
      });
      return;
    }
    setIsScanningFonts(true);
    try {
      const rep = await ScanGameFonts(targetDir);
      setFontReport(rep);
      setShowFontModal(true);
      setNotification({
        type: "success",
        title: "تحلیل فونت‌های بازی کامل شد",
        message: `تعداد ${rep.total_fonts_found} فونت شناسایی شد (${rep.persian_ready_count} مورد دارای پشتیبانی کامل از حروف فارسی).`,
      });
    } catch (e) {
      console.error(e);
      setNotification({
        type: "error",
        title: "خطا در بررسی فونت‌ها",
        message: String(e),
      });
    } finally {
      setIsScanningFonts(false);
    }
  };

  const handleStageFontReplacement = async (targetRelPath: string) => {
    const targetDir = gameInfo?.game_root || dirPath;
    if (!targetDir) return;
    try {
      const customFontPath = await SelectOpenFile();
      if (!customFontPath) return;
      const res = await StageFontReplacement(targetDir, targetRelPath, customFontPath);
      setStagedFonts((prev) => ({ ...prev, [targetRelPath]: res }));
      setNotification({
        type: "success",
        title: "جایگزینی فونت با موفقیت آماده‌سازی شد (Staged)",
        message: `فونت "${res.font_family || "Custom"}" در مسیر ایمن WorkingDir/out استیج شد. فایل‌های نصب بازی بدون تغییر مانده‌اند و این تغییر در پچ نهایی اعمال خواهد شد.`,
        diagnostic: `Original SHA256: ${res.source_sha256}\nTarget SHA256: ${res.target_sha256}\nTarget Relative: ${res.target_rel_path}\nStaged Location: ${res.staged_path}`,
      });
    } catch (e) {
      console.error(e);
      setNotification({
        type: "error",
        title: "خطا در جایگزینی فونت",
        message: String(e),
      });
    }
  };

  const handleBrowseAssets = async (openModal = true) => {
    const targetDir = gameInfo?.game_root || dirPath;
    if (!targetDir) {
      if (openModal) {
        setNotification({
          type: "warning",
          title: "مسیر بازی مشخص نیست",
          message: "لطفاً ابتدا مسیر بازی را انتخاب کنید تا فایل‌ها و دارایی‌ها کاوش شوند.",
        });
      }
      return;
    }
    setIsBrowsingAssets(true);
    try {
      const view = await BrowseGameAssets(targetDir);
      setAssetView(view);
      if (openModal) {
        setShowAssetBrowserModal(true);
      }
      if (view.root_nodes && view.root_nodes.length > 0 && !selectedAssetNode) {
        setSelectedAssetNode(view.root_nodes[0]);
      }
      if (openModal) {
        setNotification({
          type: "success",
          title: "کاوش ساختار دارایی‌ها کامل شد",
          message: `تعداد ${view.total_files} فایل (${view.total_containers} کانتینر آرشیو، ${view.total_parsed_resources} منبع متنی کامپایل‌شده) شناسایی شد.`,
        });
      }
      return view;
    } catch (e) {
      console.error(e);
      if (openModal) {
        setNotification({
          type: "error",
          title: "خطا در کاوش دارایی‌ها",
          message: String(e),
        });
      }
    } finally {
      setIsBrowsingAssets(false);
    }
  };

  const handleExport = async () => {
    if (!gameInfo?.game_root) return;
    try {
      const defaultName = `${gameInfo.game_name || "game"}_translations.xlsx`;
      const path = await SelectSaveFile(defaultName);
      if (!path) return;
      await ExportTranslations(gameInfo.game_root, path);
      setNotification({
        type: "success",
        title: "صادر شد",
        message: `فایل ترجمه با موفقیت ذخیره شد:\n${path}`,
      });
    } catch (e) {
      console.error(e);
      setNotification({
        type: "error",
        title: "خطا در صدور فایل",
        message: String(e),
        diagnostic: `Export error: ${String(e)}\nGame: ${gameInfo?.game_name}\nRoot: ${gameInfo?.game_root}`,
      });
    }
  };

  const handleImport = async () => {
    if (!gameInfo?.game_root) return;
    try {
      const path = await SelectOpenFile();
      if (!path) return;
      const res = await ImportTranslations(gameInfo.game_root, path);
      const proj = await GetProject(gameInfo.game_root);
      setTranslations(proj.entries || []);
      let msg = `عملیات ورود فایل انجام شد:\n• اعمال‌شده: ${res.Accepted}\n• نادیده‌گرفته‌شده: ${res.Skipped}\n• ردشده: ${res.Rejected}`;
      if (res.Errors && res.Errors.length > 0) {
        msg += `\n\nخطاها و هشدارهای اعتبارسنجی:\n` + res.Errors.slice(0, 5).join("\n");
        if (res.Errors.length > 5) {
          msg += `\n... و ${res.Errors.length - 5} خطای دیگر`;
        }
      }
      setNotification({
        type: res.Rejected > 0 ? "warning" : "success",
        title: res.Rejected > 0 ? "ورود فایل با خطا یا موارد رد شده" : "فایل با موفقیت وارد شد",
        message: msg,
        diagnostic: res.Errors && res.Errors.length > 0 ? res.Errors.join("\n") : undefined,
      });
    } catch (e) {
      console.error(e);
      setNotification({
        type: "error",
        title: "خطا در ورود فایل",
        message: String(e),
        diagnostic: `Import error: ${String(e)}\nGame: ${gameInfo?.game_name}\nRoot: ${gameInfo?.game_root}`,
      });
    }
  };

  const handleSaveAll = async () => {
    if (!gameInfo?.game_root) return;
    setIsSaving(true);
    try {
      await SaveTranslations(gameInfo.game_root, translations);
      const proj = await GetProject(gameInfo.game_root);
      setTranslations(proj.entries || []);
      setNotification({
        type: "success",
        title: "ذخیره شد",
        message: "تمام ترجمه‌ها و وضعیت‌های QA با موفقیت ذخیره شدند.",
      });
    } catch (e) {
      console.error(e);
      setNotification({
        type: "error",
        title: "خطا در ذخیره ترجمه‌ها",
        message: String(e),
        diagnostic: `Save error: ${String(e)}\nGame: ${gameInfo?.game_name}`,
      });
    } finally {
      setIsSaving(false);
    }
  };

  const handleSelectDir = async () => {
    try {
      const path = await SelectDirectory();
      if (path) setDirPath(path);
    } catch (e) { console.error(e); }
  };

  const handleNextStep = async (next: Step) => {
    const selectedGame = gameInfo;
    if (next === "extract" && !dirPath.trim()) {
      return;
    }
    if (next !== "extract" && !selectedGame) {
      return;
    }
    setIsProcessing(true);
    setProgress(10);
    setStageName("در حال آماده‌سازی عملیات...");
    setStageDetail("");
    try {
      if (next === "extract") {
        setStageName("در حال شناسایی موتور بازی...");
        const info = await DetectEngine(dirPath);
        setGameInfo(info);
      } else if (next === "translate") {
        await Extract(selectedGame!);
        const proj = await GetProject(selectedGame!.game_root);
        setTranslations(proj.entries || []);
        BrowseGameAssets(selectedGame!.game_root).then((view) => setAssetView(view)).catch(() => {});
      } else if (next === "inject") {
        await SaveTranslations(selectedGame!.game_root, translations);
      } else if (next === "patch") {
        await Inject(selectedGame!);
      } else if (next === "success") {
        const result = await BuildPatcher(selectedGame!.game_root, patchCredits);
        setPatchResult(result);
      }
      setProgress(100);
      setCurrentStep(next);
    } catch (e) {
      console.error(e);
      const diagReport = [
        `Step: ${next}`,
        `Engine: ${selectedGame?.engine || "N/A"}`,
        `Game Name: ${selectedGame?.game_name || "N/A"}`,
        `Game Root: ${selectedGame?.game_root || dirPath || "N/A"}`,
        `Backend: ${selectedGame?.backend || "N/A"}`,
        `Error: ${String(e)}`,
      ].join("\n");

      setNotification({
        type: "error",
        title: `خطا در مرحله ${next}`,
        message: String(e),
        diagnostic: diagReport,
      });
    } finally {
      setIsProcessing(false);
      setProgress(0);
      setStageName("");
      setStageDetail("");
    }
  };

  return (
    <div className="min-h-screen p-6 flex items-center justify-center relative overflow-hidden">
      <div className="absolute inset-0 bg-[radial-gradient(#312e81_1px,transparent_1px)] [background-size:24px_24px] opacity-20 pointer-events-none" />
      
      <div className={`w-full grid grid-cols-1 lg:grid-cols-12 gap-8 z-10 transition-all duration-300 ${currentStep === "translate" ? "max-w-[1600px]" : "max-w-5xl"}`}>
        
        {/* Sidebar / Info Panel */}
        <div className={`${currentStep === "translate" ? "lg:col-span-3 xl:col-span-3" : "lg:col-span-4"} flex flex-col gap-5`}>
          <div className="flex items-center justify-between gap-4 mb-2">
            <div className="flex items-center gap-4">
              <div className="p-3 bg-primary/10 rounded-2xl border border-primary/20 backdrop-blur-md">
                <Cpu className="w-8 h-8 text-primary" />
              </div>
              <div>
                <h1 className="text-3xl font-black tracking-tight text-white">FarsiForge</h1>
                <p className="text-zinc-400 font-medium text-sm tracking-widest uppercase">Persian Game Engine</p>
              </div>
            </div>
            <Button
              variant="outline"
              size="icon"
              onClick={openSettings}
              title="تنظیمات برنامه"
              className="border-zinc-800 bg-zinc-900/60 hover:bg-zinc-800 text-zinc-300 hover:text-white shrink-0"
            >
              <Settings2 className="w-4 h-4" />
            </Button>
          </div>

          <Card className="bg-zinc-950/40 backdrop-blur-xl border-zinc-800/50 shadow-2xl">
            <CardHeader className="py-3 px-4">
              <CardTitle className="text-base flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Box className="w-4 h-4 text-indigo-400" /> 
                  <span>مشخصات بازی</span>
                </div>
                {currentCap && currentStep !== "detect" && (
                  <Badge variant="outline" className={`text-xs ${
                    currentCap.status === "full"
                      ? "bg-emerald-500/10 text-emerald-300 border-emerald-500/20"
                      : currentCap.status === "extract-only"
                      ? "bg-amber-500/10 text-amber-300 border-amber-500/20"
                      : currentCap.status === "detection-only"
                      ? "bg-rose-500/10 text-rose-300 border-rose-500/20"
                      : "bg-blue-500/10 text-blue-300 border-blue-500/20"
                  }`}>
                    {currentCap.status === "full" ? "پشتیبانی کامل" : currentCap.status === "extract-only" ? "فقط استخراج" : currentCap.status === "detection-only" ? "فقط شناسایی" : "موتور عمومی"}
                  </Badge>
                )}
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 px-4 pb-4 pt-0 text-sm">
              <div className="flex justify-between items-center py-1.5 border-b border-zinc-800/50">
                <span className="text-zinc-400">موتور بازی</span>
                <Badge variant="outline" className="bg-indigo-500/10 text-indigo-300 border-indigo-500/20">{currentStep !== "detect" ? engineInfo.engine : "---"}</Badge>
              </div>
              {currentCap && currentStep !== "detect" && (
                <div className="py-2 border-b border-zinc-800/50 space-y-1.5">
                  <span className="text-xs text-zinc-400 block">قابلیت‌های ماژول</span>
                  <div className="flex flex-wrap gap-1.5">
                    <span className={`text-[11px] px-1.5 py-0.5 rounded border ${currentCap.can_extract ? "bg-emerald-500/10 text-emerald-400 border-emerald-500/20" : "bg-zinc-800/50 text-zinc-500 border-zinc-700/50"}`}>
                      {currentCap.can_extract ? "✓ استخراج" : "✗ استخراج"}
                    </span>
                    <span className={`text-[11px] px-1.5 py-0.5 rounded border ${currentCap.can_inject ? "bg-emerald-500/10 text-emerald-400 border-emerald-500/20" : "bg-zinc-800/50 text-zinc-500 border-zinc-700/50"}`}>
                      {currentCap.can_inject ? "✓ تزریق" : "✗ تزریق"}
                    </span>
                    <span className={`text-[11px] px-1.5 py-0.5 rounded border ${currentCap.archive_rebuild ? "bg-purple-500/10 text-purple-400 border-purple-500/20" : "bg-zinc-800/50 text-zinc-500 border-zinc-700/50"}`}>
                      {currentCap.archive_rebuild ? "✓ بازسازی آرشیو" : "فقط فایل باز (Loose)"}
                    </span>
                    <span className={`text-[11px] px-1.5 py-0.5 rounded border ${currentCap.font_support ? "bg-indigo-500/10 text-indigo-400 border-indigo-500/20" : "bg-zinc-800/50 text-zinc-500 border-zinc-700/50"}`}>
                      {currentCap.font_support ? "✓ فونت" : "✗ فونت"}
                    </span>
                  </div>
                </div>
              )}
              <div className="flex justify-between items-center py-1.5 border-b border-zinc-800/50">
                <span className="text-zinc-400">پلتفرم پردازشی</span>
                <span className="font-mono text-zinc-300 text-xs">{currentStep !== "detect" ? engineInfo.backend : "---"}</span>
              </div>
              <div className="flex justify-between items-center py-1.5 border-b border-zinc-800/50">
                <span className="text-zinc-400">دقت شناسایی</span>
                <span className="text-green-400 font-bold">{currentStep !== "detect" ? engineInfo.confidence : "---"}</span>
              </div>
              <div className="flex justify-between items-center py-1.5">
                <span className="text-zinc-400">تعداد محتوا</span>
                <span className="font-mono text-zinc-300 text-xs">{currentStep !== "detect" ? engineInfo.files : "---"}</span>
              </div>
            </CardContent>
          </Card>

          {/* Tools Status Card */}
          <Card className="bg-zinc-950/40 backdrop-blur-xl border-zinc-800/50 shadow-2xl">
            <CardHeader className="py-3 px-4">
              <CardTitle className="text-base flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Wrench className="w-4 h-4 text-emerald-400" />
                  <span>ابزارهای کمکی</span>
                </div>
                {toolStatus && (
                  <Badge variant="outline" className="text-xs bg-emerald-500/10 text-emerald-400 border-emerald-500/20">
                    سیستم ابزارها
                  </Badge>
                )}
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 px-4 pb-4 pt-0 text-xs">
              {[
                { name: "موتور fftools", ok: toolStatus?.ff_tools_ok },
                { name: "آنریل repak", ok: toolStatus?.repak_ok },
                { name: "آنریل retoc", ok: toolStatus?.retoc_ok },
                { name: "گودو gdre_tools", ok: toolStatus?.gdre_tools_ok },
                { name: "محیط Python", ok: toolStatus?.python_ok },
                { name: "پچ‌ساز (Patcher)", ok: toolStatus?.patcher_ok },
              ].map((tool) => (
                <div key={tool.name} className="flex items-center justify-between py-1 border-b border-zinc-900/60 last:border-0">
                  <span className="text-zinc-400">{tool.name}</span>
                  {tool.ok ? (
                    <span className="flex items-center gap-1 text-emerald-400 font-medium">
                      <CheckCircle2 className="w-3.5 h-3.5" /> آماده
                    </span>
                  ) : (
                    <span className="flex items-center gap-1 text-zinc-500">
                      <XCircle className="w-3.5 h-3.5" /> ناموجود
                    </span>
                  )}
                </div>
              ))}
            </CardContent>
          </Card>

          <Card className="bg-zinc-950/40 backdrop-blur-xl border-zinc-800/50">
            <CardHeader className="py-3 px-4">
              <CardTitle className="text-base flex items-center gap-2">
                <Settings2 className="w-4 h-4 text-purple-400" />
                تنظیمات بومی‌سازی
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2 px-4 pb-4 pt-0">
              {[
                { id: "rtl", label: "راست‌چین‌سازی (RTL)" },
                { id: "shaping", label: "اتصال حروف فارسی" },
                { id: "digits", label: "اعداد فارسی" },
                { id: "vazirmatn", label: "فونت Vazirmatn" },
              ].map((setting) => {
                const isChecked = persianOpts[setting.id as keyof typeof persianOpts];
                return (
                  <div
                    key={setting.id}
                    onClick={() => handleTogglePersian(setting.id)}
                    className="flex items-center gap-2.5 cursor-pointer select-none py-1 hover:text-white transition-colors"
                  >
                    <div
                      className={`w-4 h-4 rounded-sm border flex items-center justify-center transition-colors ${
                        isChecked
                          ? "bg-primary border-primary text-primary-foreground"
                          : "border-zinc-700 bg-zinc-900 text-transparent"
                      }`}
                    >
                      <CheckCircle2 className="w-3 h-3" />
                    </div>
                    <span className="text-xs text-zinc-300">{setting.label}</span>
                  </div>
                );
              })}
            </CardContent>
          </Card>

          {/* Font Inventory & Diagnostics Card */}
          <Card className="bg-zinc-950/40 backdrop-blur-xl border-zinc-800/50 shadow-2xl">
            <CardHeader className="py-3 px-4">
              <CardTitle className="text-base flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <Type className="w-4 h-4 text-sky-400" />
                  <span>فونت و تایپوگرافی</span>
                </div>
                {fontReport && (
                  <Badge variant="outline" className="text-xs bg-sky-500/10 text-sky-400 border-sky-500/20">
                    {fontReport.total_fonts_found} فونت
                  </Badge>
                )}
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2.5 px-4 pb-4 pt-0 text-xs">
              {fontReport ? (
                <div className="space-y-2">
                  <div className="flex justify-between items-center text-zinc-300">
                    <span className="text-zinc-400">سازگار با فارسی:</span>
                    <span className="font-bold text-emerald-400">{fontReport.persian_ready_count} از {fontReport.total_fonts_found}</span>
                  </div>
                  <div className="flex justify-between items-center text-zinc-400 text-[11px]">
                    <span>فایل‌های مستقیم TTF/OTF:</span>
                    <span className="font-mono text-zinc-300">{fontReport.direct_font_files}</span>
                  </div>
                  <div className="flex justify-between items-center text-zinc-400 text-[11px]">
                    <span>ریسورس‌های کامپایل‌شده:</span>
                    <span className="font-mono text-zinc-300">{fontReport.engine_resources}</span>
                  </div>
                  {Object.keys(stagedFonts).length > 0 && (
                    <div className="p-1.5 rounded bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 text-[11px]">
                      ✓ {Object.keys(stagedFonts).length} فونت جایگزین در صف پچ
                    </div>
                  )}
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setShowFontModal(true)}
                    className="w-full text-xs border-sky-500/30 text-sky-300 hover:bg-sky-500/10"
                  >
                    مشاهده جزئیات و جایگزینی
                  </Button>
                </div>
              ) : (
                <div className="space-y-2">
                  <p className="text-zinc-400 text-[11px] leading-relaxed">
                    تحلیل فونت‌های بازی، سنجش پوشش گلیف‌های فارسی (گ چ پ ژ) و اتصال حروف OpenType.
                  </p>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={isScanningFonts || (!dirPath && !gameInfo?.game_root)}
                    onClick={handleScanFonts}
                    className="w-full text-xs border-zinc-700 hover:border-sky-500 hover:text-sky-300"
                  >
                    {isScanningFonts ? "در حال تحلیل فونت‌ها..." : "بررسی فونت‌های بازی"}
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>
        </div>

        {/* Main Work Area */}
        <div className={`${currentStep === "translate" ? "lg:col-span-9 xl:col-span-9" : "lg:col-span-8"} flex flex-col`}>
          <Card className="flex-1 bg-zinc-950/60 backdrop-blur-2xl border-zinc-800/60 shadow-2xl relative overflow-hidden flex flex-col">
            {/* Dynamic header gradient */}
            <div className="absolute top-0 inset-x-0 h-1 bg-gradient-to-r from-transparent via-primary to-transparent opacity-50" />
            
            <CardHeader className="border-b border-zinc-800/50 pb-6 pt-8">
              <div className="flex justify-between items-center">
                <CardTitle className="text-2xl font-bold">پایپ‌لاین بومی‌سازی</CardTitle>
                <div className="flex gap-2">
                  <Badge variant={currentStep === "detect" ? "default" : "outline"}>شناسایی</Badge>
                  <ArrowRight className="w-4 h-4 text-zinc-600 mt-0.5" />
                  <Badge variant={currentStep === "extract" ? "default" : "outline"}>استخراج</Badge>
                  <ArrowRight className="w-4 h-4 text-zinc-600 mt-0.5" />
                  <Badge variant={currentStep === "inject" ? "default" : "outline"}>تزریق</Badge>
                </div>
              </div>
            </CardHeader>

            <CardContent className="flex-1 flex flex-col justify-center p-8 relative">
              <AnimatePresence mode="wait">
                
                {/* STEP 1: DETECT */}
                {currentStep === "detect" && (
                  <motion.div 
                    key="detect"
                    initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -10 }}
                    className="flex flex-col items-center text-center space-y-8"
                  >
                    <div className="w-24 h-24 rounded-full bg-indigo-500/10 flex items-center justify-center border border-indigo-500/20 shadow-[0_0_50px_rgba(99,102,241,0.1)]">
                      <FolderSearch className="w-12 h-12 text-indigo-400" />
                    </div>
                    <div className="space-y-2">
                      <h2 className="text-2xl font-bold text-white">انتخاب مسیر بازی</h2>
                      <p className="text-zinc-400 max-w-md mx-auto">برای شروع عملیات، مسیر فولدر بازی را انتخاب کنید تا موتور بازی و فایل‌های قابل ترجمه شناسایی شوند.</p>
                    </div>
                    
                    <div className="w-full max-w-md flex flex-col gap-4">
                      <div className="flex gap-2 relative">
                        <div className="relative flex-1">
                          <FolderOpen className="absolute right-3 top-1/2 -translate-y-1/2 w-5 h-5 text-zinc-500" />
                          <Input 
                            value={dirPath}
                            onChange={(e) => setDirPath(e.target.value)}
                            placeholder="مسیر پوشهٔ بازی را انتخاب کنید"
                            className="pl-4 pr-10 py-6 bg-zinc-900/50 border-zinc-800 text-left font-mono text-lg focus-visible:ring-indigo-500"
                            dir="ltr"
                          />
                        </div>
                        <Button onClick={handleSelectDir} className="h-[52px] px-6 bg-zinc-800 hover:bg-zinc-700">انتخاب مسیر</Button>
                      </div>
                      
                      {isProcessing ? (
                        <div className="space-y-2 w-full">
                          <Progress value={progress} className="h-2" />
                          <p className="text-xs text-zinc-500 text-center animate-pulse">در حال تحلیل ساختار فایل‌ها...</p>
                        </div>
                      ) : (
                        <Button onClick={() => handleNextStep("extract")} disabled={!dirPath.trim()} size="lg" className="w-full text-lg h-14 bg-indigo-600 hover:bg-indigo-700 shadow-[0_0_20px_rgba(79,70,229,0.3)] transition-all">
                          شناسایی موتور
                        </Button>
                      )}
                    </div>
                  </motion.div>
                )}

                {/* STEP 2: EXTRACT */}
                {currentStep === "extract" && (
                  <motion.div 
                    key="extract"
                    initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -10 }}
                    className="flex flex-col items-center text-center space-y-8"
                  >
                    <div className="w-24 h-24 rounded-full bg-purple-500/10 flex items-center justify-center border border-purple-500/20 shadow-[0_0_50px_rgba(168,85,247,0.1)]">
                      <Database className="w-12 h-12 text-purple-400" />
                    </div>
                    <div className="space-y-2">
                      <h2 className="text-2xl font-bold text-white">استخراج دیتابیس متن‌ها</h2>
                      <p className="text-zinc-400 max-w-md mx-auto">موتور {engineInfo.engine} با موفقیت شناسایی شد. اکنون فایل‌های متنی و دارایی‌ها (Assets) را استخراج می‌کنیم.</p>
                      {currentCap?.extraction_notes && (
                        <div className="p-3 bg-indigo-500/10 border border-indigo-500/20 rounded-lg text-indigo-300 text-xs max-w-md mx-auto text-right">
                          ℹ <strong>راهنمای ماژول:</strong> {currentCap.extraction_notes}
                        </div>
                      )}
                      {isDetectionOnly && (
                        <div className="p-3 bg-amber-500/10 border border-amber-500/30 rounded-lg text-amber-300 text-sm max-w-md mx-auto">
                          ⚠ موتور {engineInfo.engine} در وضعیت فقط-شناسایی (Detection-Only) است؛ ماژول استخراج اختصاصی برای آن در دسترس نیست.
                        </div>
                      )}
                    </div>
                    
                    <div className="w-full max-w-md flex flex-col gap-4">
                      {isProcessing ? (
                        <div className="space-y-4 w-full p-4 rounded-xl bg-zinc-900/60 border border-zinc-800 text-right">
                          <div className="flex items-center justify-between text-xs text-zinc-400">
                            <span className="font-mono text-purple-400 font-bold text-sm">{Math.round(progress)}%</span>
                            <span className="flex items-center gap-1.5 text-purple-300 font-medium">
                              <Database className="w-4 h-4 animate-spin text-purple-400" />
                              {stageName || "در حال کاوش و استخراج متون و دارایی‌های بازی..."}
                            </span>
                          </div>
                          <Progress value={progress} className="h-2.5 bg-zinc-800" />
                          <div className="p-3 rounded-lg bg-zinc-950/80 border border-zinc-800/80 font-mono text-xs text-zinc-300 text-left space-y-1.5" dir="ltr">
                            <div className="flex items-center gap-2 text-zinc-500 text-[11px]">
                              <FolderOpen className="w-3.5 h-3.5 text-purple-400 shrink-0" />
                              <span className="truncate">Root: {gameInfo?.game_root || dirPath}</span>
                            </div>
                            {stageDetail && (
                              <div className="text-indigo-300 font-medium truncate flex items-center gap-2">
                                <span className="text-zinc-500 shrink-0">📦 Unpacking:</span>
                                <span className="truncate">{stageDetail}</span>
                              </div>
                            )}
                          </div>
                        </div>
                      ) : (
                        <Button onClick={() => handleNextStep("translate")} disabled={isDetectionOnly} size="lg" className="w-full text-lg h-14 bg-purple-600 hover:bg-purple-700 shadow-[0_0_20px_rgba(168,85,247,0.3)] transition-all">
                          {isDetectionOnly ? "استخراج پشتیبانی نمی‌شود" : "شروع استخراج"}
                        </Button>
                      )}
                    </div>
                  </motion.div>
                )}

                {/* STEP 3: TRANSLATE MOCK */}
                {currentStep === "translate" && (
                  <motion.div 
                    key="translate"
                    initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -10 }}
                    className="w-full h-full flex flex-col space-y-4"
                  >
                    <div className="flex items-center justify-between gap-3">
                      <div className="flex items-center gap-3">
                        <div className="p-2 bg-blue-500/10 rounded-lg border border-blue-500/20">
                          <Languages className="w-6 h-6 text-blue-400" />
                        </div>
                        <div>
                          <h2 className="text-xl font-bold text-white">میز کار ترجمه</h2>
                          <p className="text-xs text-zinc-400">ویرایش، اعتبارسنجی خودکار و تبادل فایل‌های ترجمه</p>
                        </div>
                      </div>
                      
                      {/* Action buttons: Asset Browser, Fonts, Export, Import, Save */}
                      <div className="flex items-center gap-2">
                        <Button 
                          onClick={() => {
                            if (!assetView) {
                              handleBrowseAssets();
                            } else {
                              setShowAssetBrowserModal(true);
                            }
                          }}
                          variant="outline" 
                          size="sm" 
                          className="h-9 px-3 bg-zinc-900 border-zinc-700 text-zinc-200 hover:bg-zinc-800 flex items-center gap-1.5"
                          title="کاوشگر ساختار فایل‌ها، کانتینرها و اصالت منابع"
                        >
                          <FolderSearch className="w-4 h-4 text-emerald-400" />
                          <span>کاوشگر دارایی‌ها {assetView ? `(${assetView.total_files})` : ""}</span>
                        </Button>
                        <Button 
                          onClick={() => {
                            if (!fontReport) {
                              handleScanFonts();
                            } else {
                              setShowFontModal(true);
                            }
                          }}
                          variant="outline" 
                          size="sm" 
                          className="h-9 px-3 bg-zinc-900 border-zinc-700 text-zinc-200 hover:bg-zinc-800 flex items-center gap-1.5"
                          title="تحلیل و جایگزینی ایمن فونت‌های بازی"
                        >
                          <Type className="w-4 h-4 text-sky-400" />
                          <span>فونت‌های بازی {fontReport ? `(${fontReport.total_fonts_found})` : ""}</span>
                        </Button>
                        <Button 
                          onClick={handleExport} 
                          variant="outline" 
                          size="sm" 
                          className="h-9 px-3 bg-zinc-900 border-zinc-700 text-zinc-200 hover:bg-zinc-800 flex items-center gap-1.5"
                          title="خروجی به اکسل یا CSV برای ترجمه خارج از برنامه"
                        >
                          <Download className="w-4 h-4 text-emerald-400" />
                          <span>خروجی Excel/CSV</span>
                        </Button>
                        <Button 
                          onClick={handleImport} 
                          variant="outline" 
                          size="sm" 
                          className="h-9 px-3 bg-zinc-900 border-zinc-700 text-zinc-200 hover:bg-zinc-800 flex items-center gap-1.5"
                          title="ورود فایل ترجمه‌شده از اکسل یا CSV"
                        >
                          <Upload className="w-4 h-4 text-indigo-400" />
                          <span>ورود Excel/CSV</span>
                        </Button>
                        <Button 
                          onClick={handleSaveAll} 
                          disabled={isSaving}
                          size="sm" 
                          className="h-9 px-3 bg-zinc-800 hover:bg-zinc-700 text-white flex items-center gap-1.5"
                          title="ذخیره تغییرات در دیتابیس پروژه"
                        >
                          <Save className="w-4 h-4 text-amber-400" />
                          <span>{isSaving ? "در حال ذخیره..." : "ذخیره تغییرات"}</span>
                        </Button>
                      </div>
                    </div>

                    {/* View Switcher: Translation Candidates Workbench vs Real Compiled Asset Explorer */}
                    <div className="flex flex-wrap items-center justify-between gap-3 bg-zinc-950 p-1.5 rounded-xl border border-zinc-800 shrink-0">
                      <div className="flex items-center gap-1.5">
                        <button
                          type="button"
                          onClick={() => setWorkbenchView("candidates")}
                          className={`px-3.5 py-1.5 rounded-lg text-xs font-medium flex items-center gap-2 transition-all ${
                            workbenchView === "candidates"
                              ? "bg-indigo-600 text-white shadow"
                              : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900"
                          }`}
                        >
                          <Languages className="w-4 h-4 text-indigo-300" />
                          <span>میزکار ترجمه و کاندیداها</span>
                          <span className="text-[10px] font-mono px-1.5 py-0.2 rounded bg-indigo-950/80 border border-indigo-400/30 text-indigo-200">
                            {classificationCounts.verified} مورد تأییدشده
                          </span>
                        </button>
                        <button
                          type="button"
                          onClick={() => {
                            setWorkbenchView("explorer");
                            if (!assetView) {
                              handleBrowseAssets(false);
                            }
                          }}
                          className={`px-3.5 py-1.5 rounded-lg text-xs font-medium flex items-center gap-2 transition-all ${
                            workbenchView === "explorer"
                              ? "bg-emerald-600 text-white shadow"
                              : "text-zinc-400 hover:text-zinc-200 hover:bg-zinc-900"
                          }`}
                        >
                          <FolderSearch className="w-4 h-4 text-emerald-300" />
                          <span>کاوشگر دارایی‌های کامپایل‌شده (Compiled Asset Explorer)</span>
                          {assetView && (
                            <span className="text-[10px] font-mono px-1.5 py-0.2 rounded bg-emerald-950/80 border border-emerald-400/30 text-emerald-200">
                              {assetView.total_files} فایل
                            </span>
                          )}
                        </button>
                      </div>

                      <div className="flex items-center gap-2 text-xs text-zinc-400">
                        {workbenchView === "candidates" ? (
                          <span className="text-[11px] text-zinc-400 flex items-center gap-1">
                            <CheckCircle2 className="w-3.5 h-3.5 text-emerald-400" />
                            پیش‌فرض: فقط کاندیداهای با اصالت ساختاری و مسیر تزریق امن
                          </span>
                        ) : (
                          <span className="text-[11px] text-zinc-400 flex items-center gap-1">
                            <Database className="w-3.5 h-3.5 text-sky-400" />
                            بررسی مستقیم Serialized Objects باینری، TypeTree و پراپرتی‌ها
                          </span>
                        )}
                      </div>
                    </div>

                    {workbenchView === "candidates" && (
                    <div className="flex flex-col space-y-3.5">
                    {/* WinRAR Breadcrumb & Archive Metrics Bar */}
                    <div className="flex flex-wrap items-center justify-between gap-3 p-2.5 rounded-xl bg-zinc-900/90 border border-zinc-800 text-xs font-mono">
                      <div className="flex items-center gap-2 min-w-0 max-w-full">
                        <FolderOpen className="w-4 h-4 text-amber-400 shrink-0" />
                        <span className="text-zinc-500 font-sans">موقعیت:</span>
                        <span className="text-zinc-200 font-semibold truncate select-all" dir="ltr">
                          {gameInfo?.game_root ? gameInfo.game_root.replace(/\\/g, "/") : "Game_Root"}
                          {selectedArchiveFile !== "all" ? ` / ${selectedArchiveFile.replace(/\\/g, "/")}` : " / [تمام آرشیوها و کانتینرها]"}
                        </span>
                      </div>
                      <div className="flex flex-wrap items-center gap-1.5 shrink-0 font-sans">
                        <Badge variant="outline" className="text-[11px] bg-zinc-800/80 text-zinc-300 border-zinc-700">
                          📦 {archiveFiles.length} بسته آرشیو
                        </Badge>
                        <Badge variant="outline" className="text-[11px] bg-purple-500/10 text-purple-300 border-purple-500/30">
                          💬 {archiveFiles.reduce((acc, f) => acc + f.dialogues, 0)} گفت‌وگو
                        </Badge>
                        <Badge variant="outline" className="text-[11px] bg-emerald-500/10 text-emerald-400 border-emerald-500/30">
                          ✓ {translations.filter(t => t.translation?.trim()).length} ترجمه‌شده
                        </Badge>
                        {qaIssuesCount > 0 && (
                          <Badge 
                            onClick={() => setStatusFilter("qa")}
                            variant="outline" 
                            className="cursor-pointer text-[11px] bg-amber-500/10 text-amber-300 border-amber-500/30 flex items-center gap-1 hover:bg-amber-500/20"
                          >
                            <AlertTriangle className="w-3 h-3 text-amber-400" />
                            {qaIssuesCount} خطای QA
                          </Badge>
                        )}
                      </div>
                    </div>

                    {/* Split-View: Left Archive Tree + Right WinRAR Table & Dialogue Inspector */}
                    <div className="flex flex-col lg:flex-row gap-3.5 h-[590px]">
                      
                      {/* Left Panel: Archive Files & Packages Tree */}
                      <div className="w-full lg:w-72 xl:w-80 shrink-0 bg-zinc-950/70 border border-zinc-800/80 rounded-xl flex flex-col overflow-hidden shadow-xl">
                        <div className="p-2.5 border-b border-zinc-800 bg-zinc-900/60 flex items-center justify-between">
                          <div className="flex items-center gap-2 text-xs font-semibold text-zinc-200">
                            <Archive className="w-4 h-4 text-purple-400" />
                            <span>بسته‌ها و فایل‌ها ({archiveFiles.length})</span>
                          </div>
                          <span className="text-[10px] text-zinc-400 font-mono font-medium">WinRAR View</span>
                        </div>
                        
                        <div className="p-2 border-b border-zinc-800/70 bg-zinc-950/40">
                          <Input
                            value={archiveSearch}
                            onChange={(e) => setArchiveSearch(e.target.value)}
                            placeholder="فیلتر فایل‌ها و بسته‌ها..."
                            className="h-8 text-xs bg-zinc-900/80 border-zinc-800 placeholder:text-zinc-600"
                          />
                        </div>

                        <div className="flex-1 overflow-y-auto p-2 space-y-1 divide-y divide-zinc-900/40">
                          <button
                            type="button"
                            onClick={() => setSelectedArchiveFile("all")}
                            className={`w-full text-right p-2 rounded-lg transition-all flex items-center justify-between gap-2 text-xs ${
                              selectedArchiveFile === "all"
                                ? "bg-indigo-600/20 border border-indigo-500/40 text-indigo-200 shadow-sm"
                                : "hover:bg-zinc-900/60 text-zinc-300 border border-transparent"
                            }`}
                          >
                            <div className="flex items-center gap-2 min-w-0">
                              <FolderTree className="w-4 h-4 text-indigo-400 shrink-0" />
                              <div className="truncate font-medium">تمام آرشیوها و بسته‌ها</div>
                            </div>
                            <Badge variant="outline" className="text-[10px] bg-zinc-900 border-zinc-700 text-zinc-400 font-mono">
                              {translations.length}
                            </Badge>
                          </button>

                          {filteredArchiveFiles.map((f) => {
                            const isSelected = selectedArchiveFile === f.file;
                            const ext = f.name.includes(".") ? f.name.split(".").pop()?.toLowerCase() : "";
                            const icon = ext === "assets" ? (
                              <Archive className="w-4 h-4 text-amber-400 shrink-0" />
                            ) : ext === "bundle" ? (
                              <Box className="w-4 h-4 text-purple-400 shrink-0" />
                            ) : ext === "locres" ? (
                              <Languages className="w-4 h-4 text-emerald-400 shrink-0" />
                            ) : (
                              <FileCode className="w-4 h-4 text-sky-400 shrink-0" />
                            );

                            const percent = f.total > 0 ? Math.round((f.translated / f.total) * 100) : 0;

                            return (
                              <button
                                key={f.file}
                                type="button"
                                onClick={() => setSelectedArchiveFile(f.file)}
                                className={`w-full text-right p-2 rounded-lg transition-all flex flex-col gap-1 text-xs ${
                                  isSelected
                                    ? "bg-indigo-600/20 border border-indigo-500/40 text-indigo-200 shadow-sm"
                                    : "hover:bg-zinc-900/60 text-zinc-300 border border-transparent"
                                }`}
                              >
                                <div className="flex items-center justify-between gap-1.5 w-full">
                                  <div className="flex items-center gap-2 min-w-0">
                                    {icon}
                                    <span className="truncate font-mono font-medium text-xs text-left" dir="ltr" title={f.name}>
                                      {f.name}
                                    </span>
                                  </div>
                                  <span className="text-[10px] font-mono px-1.5 py-0.2 rounded bg-zinc-900 border border-zinc-800 text-zinc-400 shrink-0">
                                    {f.total}
                                  </span>
                                </div>
                                <div className="flex items-center justify-between text-[10px] text-zinc-500 pr-6">
                                  <span className="truncate text-left font-mono" dir="ltr" title={f.dir}>
                                    {f.dir}
                                  </span>
                                  <div className="flex items-center gap-1.5 shrink-0">
                                    {f.dialogues > 0 && (
                                      <span className="text-purple-400 font-medium">💬 {f.dialogues}</span>
                                    )}
                                    {percent > 0 ? (
                                      <span className="text-emerald-400 font-mono font-semibold">{percent}%</span>
                                    ) : null}
                                  </div>
                                </div>
                              </button>
                            );
                          })}
                        </div>
                      </div>

                      {/* Right Panel: WinRAR Entries Explorer & Dialogue Inspector */}
                      <div className="flex-1 flex flex-col gap-3 min-w-0 h-full">
                        
                        {/* Filter & Search Toolbar */}
                        <div className="flex flex-col gap-2 shrink-0">
                          <div className="flex flex-wrap items-center justify-between gap-2">
                            <div className="flex items-center gap-1 p-1 bg-zinc-900/70 border border-zinc-800 rounded-lg text-xs">
                              <button
                                type="button"
                                onClick={() => setActiveArchiveTab("all")}
                                className={`px-2.5 py-1 rounded transition-colors ${activeArchiveTab === "all" ? "bg-indigo-600 text-white font-medium" : "text-zinc-400 hover:text-white"}`}
                              >
                                همه ({filteredEntries.length})
                              </button>
                              <button
                                type="button"
                                onClick={() => setActiveArchiveTab("dialogue")}
                                className={`px-2.5 py-1 rounded flex items-center gap-1 transition-colors ${activeArchiveTab === "dialogue" ? "bg-purple-600 text-white font-medium" : "text-zinc-400 hover:text-white"}`}
                              >
                                <span>💬 گفت‌وگوها</span>
                              </button>
                              <button
                                type="button"
                                onClick={() => setActiveArchiveTab("ui")}
                                className={`px-2.5 py-1 rounded flex items-center gap-1 transition-colors ${activeArchiveTab === "ui" ? "bg-sky-600 text-white font-medium" : "text-zinc-400 hover:text-white"}`}
                              >
                                <span>🔤 رابط کاربری</span>
                              </button>
                              <button
                                type="button"
                                onClick={() => setActiveArchiveTab("quest")}
                                className={`px-2.5 py-1 rounded flex items-center gap-1 transition-colors ${activeArchiveTab === "quest" ? "bg-amber-600 text-white font-medium" : "text-zinc-400 hover:text-white"}`}
                              >
                                <span>📜 مأموریت و آیتم</span>
                              </button>
                            </div>

                            <div className="flex items-center gap-2 flex-1 min-w-[240px]">
                              <div className="relative flex-1">
                                <Search className="w-3.5 h-3.5 text-zinc-500 absolute right-2.5 top-2.5" />
                                <Input
                                  value={searchQuery}
                                  onChange={(e) => setSearchQuery(e.target.value)}
                                  className="pr-8 h-8 text-xs bg-zinc-900 border-zinc-700 text-zinc-200 placeholder:text-zinc-500"
                                  placeholder="جستجو در متن انگلیسی، ترجمه، شناسه، فایل..."
                                />
                              </div>
                              <select
                                value={statusFilter}
                                onChange={(e) => setStatusFilter(e.target.value)}
                                className="h-8 bg-zinc-900 border border-zinc-700 rounded-md px-2 text-xs text-zinc-200 shrink-0"
                              >
                                <option value="all">همه وضعیت‌ها</option>
                                <option value="untranslated">بی‌ترجمه</option>
                                <option value="translated">ترجمه‌شده</option>
                                <option value="approved">تأییدشده</option>
                                <option value="qa">هشدار QA ({qaIssuesCount})</option>
                              </select>
                            </div>
                          </div>

                          <div className="flex flex-wrap items-center gap-1.5 text-[11px]">
                            <span className="text-zinc-500 flex items-center gap-1 pl-1">
                              <Filter className="w-3 h-3 text-indigo-400" />
                              فیلتر اصالت و کیفیت:
                            </span>
                            <Badge
                              onClick={() => setCandidateClassFilter("verified")}
                              variant={candidateClassFilter === "verified" ? "default" : "outline"}
                              className={`cursor-pointer text-[11px] py-0.5 px-2.5 transition-all ${candidateClassFilter === "verified" ? "bg-emerald-600 text-white" : "text-emerald-400 hover:bg-emerald-500/10 border-emerald-500/30"}`}
                              title="نامزدهای دارای شواهد متنی ساختاری و مسیر تزریق تضمین‌شده"
                            >
                              ✓ تأییدشده و قابل‌تزریق ({classificationCounts.verified})
                            </Badge>
                            <Badge
                              onClick={() => setCandidateClassFilter("review")}
                              variant={candidateClassFilter === "review" ? "default" : "outline"}
                              className={`cursor-pointer text-[11px] py-0.5 px-2.5 transition-all ${candidateClassFilter === "review" ? "bg-amber-600 text-white" : "text-amber-400 hover:bg-amber-500/10 border-amber-500/30"}`}
                              title="نیازمند بازبینی انسانی قبل از ترجمه"
                            >
                              ? بازبینی ({classificationCounts.review})
                            </Badge>
                            <Badge
                              onClick={() => setCandidateClassFilter("raw")}
                              variant={candidateClassFilter === "raw" ? "default" : "outline"}
                              className={`cursor-pointer text-[11px] py-0.5 px-2.5 transition-all ${candidateClassFilter === "raw" ? "bg-sky-600 text-white" : "text-sky-400 hover:bg-sky-500/10 border-sky-500/30"}`}
                              title="کاندیداهای اسکن خام و غیرقطعی"
                            >
                              ⚡ اسکن خام ({classificationCounts.raw})
                            </Badge>
                            <Badge
                              onClick={() => setCandidateClassFilter("read-only")}
                              variant={candidateClassFilter === "read-only" ? "default" : "outline"}
                              className={`cursor-pointer text-[11px] py-0.5 px-2.5 transition-all ${candidateClassFilter === "read-only" ? "bg-rose-900/80 text-white" : "text-rose-400 hover:bg-rose-500/10 border-rose-500/30"}`}
                              title="متونی که فاقد مسیر تزریق امن هستند و تغییر داده نمی‌شوند"
                            >
                              🔒 فقط‌خواندنی ({classificationCounts.readOnly})
                            </Badge>
                            <Badge
                              onClick={() => setCandidateClassFilter("all")}
                              variant={candidateClassFilter === "all" ? "default" : "outline"}
                              className="cursor-pointer text-[11px] py-0.5 px-2 transition-all"
                            >
                              همه ({classificationCounts.total})
                            </Badge>
                          </div>
                        </div>

                        {/* WinRAR File Table */}
                        <div className="flex-1 overflow-auto rounded-xl border border-zinc-800/80 bg-zinc-950/80">
                          <table className="w-full text-right text-xs border-collapse">
                            <thead className="sticky top-0 bg-zinc-900/95 backdrop-blur border-b border-zinc-800 text-zinc-400 font-medium z-10">
                              <tr>
                                <th className="py-2 px-3 text-right w-[230px]">مسیر و شناسه در آرشیو</th>
                                <th className="py-2 px-2 text-center w-[85px]">دسته</th>
                                <th className="py-2 px-3 text-left font-mono" dir="ltr">متن اصلی (Source - Monospace)</th>
                                <th className="py-2 px-3 text-right">ترجمه فارسی (RTL)</th>
                                <th className="py-2 px-2 text-center w-[75px]">وضعیت</th>
                              </tr>
                            </thead>
                            <tbody className="divide-y divide-zinc-900/60">
                              {visibleEntries.length === 0 ? (
                                <tr>
                                  <td colSpan={5} className="py-12 text-center text-zinc-500">
                                    هیچ متنی با فیلترهای انتخابی یافت نشد.
                                  </td>
                                </tr>
                              ) : (
                                visibleEntries.map(({ item, idx }) => {
                                  const isSelected = activeEntry?.id === item.id;
                                  const hasTranslation = Boolean(item.translation?.trim());

                                  return (
                                    <tr
                                      key={item.id || idx}
                                      onClick={() => setSelectedEntryId(item.id)}
                                      className={`cursor-pointer transition-colors ${
                                        isSelected
                                          ? "bg-indigo-950/60 border-r-2 border-indigo-500 text-white"
                                          : "hover:bg-zinc-900/40 text-zinc-300"
                                      }`}
                                    >
                                      <td className="py-2 px-3 align-top font-mono text-[11px] max-w-[230px]">
                                        <div className="flex items-center gap-1.5">
                                          {item.read_only ? (
                                            <span title={item.read_only_reason || "فقط‌خواندنی"}>
                                              <Lock className="w-3 h-3 text-amber-400 shrink-0" />
                                            </span>
                                          ) : (
                                            <FileText className="w-3 h-3 text-zinc-500 shrink-0" />
                                          )}
                                          <span className="truncate text-indigo-300" dir="ltr" title={item.file}>
                                            {item.file}
                                          </span>
                                        </div>
                                        {item.path ? (
                                          <div className="truncate text-[10px] text-zinc-500 pr-4" dir="ltr" title={item.path}>
                                            {item.path}
                                          </div>
                                        ) : null}
                                      </td>
                                      <td className="py-2 px-2 align-top text-center">
                                        <span className={`inline-block px-1.5 py-0.5 rounded text-[10px] ${
                                          item.category === "dialogue" ? "bg-purple-500/10 text-purple-300 border border-purple-500/20" :
                                          item.category === "ui" ? "bg-sky-500/10 text-sky-300 border border-sky-500/20" :
                                          item.category === "quest" ? "bg-amber-500/10 text-amber-300 border border-amber-500/20" :
                                          item.category === "item" ? "bg-emerald-500/10 text-emerald-300 border border-emerald-500/20" :
                                          "bg-zinc-800/60 text-zinc-400"
                                        }`}>
                                          {item.category === "dialogue" ? "گفت‌وگو" :
                                           item.category === "ui" ? "UI" :
                                           item.category === "quest" ? "مأموریت" :
                                           item.category === "item" ? "آیتم" : "عمومی"}
                                        </span>
                                      </td>
                                      <td className="py-2 px-3 align-top font-mono text-[11px] text-zinc-300 text-left" dir="ltr">
                                        <div className="line-clamp-2 select-text" title={item.source}>
                                          {item.source}
                                        </div>
                                      </td>
                                      <td className="py-2 px-3 align-top text-xs text-zinc-100 text-right" dir="rtl">
                                        {hasTranslation ? (
                                          <div className="line-clamp-2 text-emerald-300 font-medium" title={item.translation}>
                                            {item.translation}
                                          </div>
                                        ) : (
                                          <span className="text-zinc-600 italic">بدون ترجمه</span>
                                        )}
                                      </td>
                                      <td className="py-2 px-2 align-top text-center">
                                        {item.read_only ? (
                                          <span className="text-[10px] text-amber-400 bg-amber-500/10 px-1 py-0.5 rounded border border-amber-500/20">قفل</span>
                                        ) : hasTranslation ? (
                                          <span className="text-[10px] text-emerald-400 bg-emerald-500/10 px-1 py-0.5 rounded border border-emerald-500/20">ترجمه‌شده</span>
                                        ) : (
                                          <span className="text-[10px] text-zinc-500 bg-zinc-800/40 px-1 py-0.5 rounded">خام</span>
                                        )}
                                      </td>
                                    </tr>
                                  );
                                })
                              )}
                            </tbody>
                          </table>
                        </div>

                        {/* Bottom Panel: WinRAR Inspector / Dialogue Editor */}
                        <div className="h-[195px] shrink-0 bg-zinc-950/95 rounded-xl border border-zinc-800 p-3 flex flex-col justify-between shadow-2xl">
                          {activeEntry ? (
                            <div className="h-full flex flex-col justify-between gap-2">
                              {/* Top details of active entry */}
                              <div className="flex items-center justify-between gap-2 border-b border-zinc-800/70 pb-1.5 text-xs">
                                <div className="flex items-center gap-2 truncate min-w-0">
                                  <span className="font-mono text-indigo-400 font-medium truncate" dir="ltr" title={`${activeEntry.file} · ${activeEntry.path || activeEntry.context}`}>
                                    📦 {activeEntry.file} {activeEntry.path ? `· ${activeEntry.path}` : ""}
                                  </span>
                                  {activeEntry.confidence ? (
                                    <span className="text-[10px] font-mono px-1.5 py-0.2 rounded bg-zinc-900 border border-zinc-700 text-zinc-400" title="امتیاز اطمینان استخراج">
                                      {(activeEntry.confidence * 100).toFixed(0)}%
                                    </span>
                                  ) : null}
                                  {activeEntry.read_only && (
                                    <span className="text-[10px] text-amber-300 bg-amber-500/10 px-1.5 py-0.2 rounded border border-amber-500/30 flex items-center gap-1">
                                      <Lock className="w-2.5 h-2.5" /> فقط‌خواندنی
                                    </span>
                                  )}
                                  {activeEntry.notes && (
                                    <span className="text-[10px] text-amber-400 truncate" title={activeEntry.notes}>
                                      ⚠ {activeEntry.notes}
                                    </span>
                                  )}
                                </div>

                                <div className="flex items-center gap-1.5 shrink-0">
                                  <Button
                                    onClick={handleSelectPrev}
                                    disabled={filteredEntries.findIndex(e => e.item.id === activeEntry.id) <= 0}
                                    variant="outline"
                                    size="sm"
                                    className="h-7 px-2 text-xs bg-zinc-900 border-zinc-800 text-zinc-300 hover:text-white flex items-center gap-1"
                                    title="مورد قبلی"
                                  >
                                    <ChevronRight className="w-3.5 h-3.5" />
                                    <span>قبلی</span>
                                  </Button>
                                  <span className="text-[11px] text-zinc-400 font-mono px-1">
                                    {filteredEntries.findIndex(e => e.item.id === activeEntry.id) + 1} / {filteredEntries.length}
                                  </span>
                                  <Button
                                    onClick={handleSelectNext}
                                    disabled={filteredEntries.findIndex(e => e.item.id === activeEntry.id) >= filteredEntries.length - 1}
                                    variant="outline"
                                    size="sm"
                                    className="h-7 px-2 text-xs bg-zinc-900 border-zinc-800 text-zinc-300 hover:text-white flex items-center gap-1"
                                    title="مورد بعدی"
                                  >
                                    <span>بعدی</span>
                                    <ChevronLeft className="w-3.5 h-3.5" />
                                  </Button>
                                </div>
                              </div>

                              {/* 2-column editor */}
                              <div className="grid grid-cols-1 md:grid-cols-2 gap-3 flex-1 min-h-0">
                                {/* Source Box */}
                                <div className="flex flex-col gap-1 min-h-0">
                                  <div className="flex items-center justify-between text-[11px] text-zinc-400">
                                    <span className="flex items-center gap-1">
                                      <FileText className="w-3 h-3 text-indigo-400" />
                                      متن اصلی انگلیسی (Source):
                                    </span>
                                    <button
                                      type="button"
                                      onClick={() => {
                                        navigator.clipboard.writeText(activeEntry.source);
                                        setNotification({ type: "info", title: "کپی شد", message: "متن اصلی کپی شد." });
                                      }}
                                      className="text-[10px] text-zinc-400 hover:text-white flex items-center gap-1"
                                      title="کپی متن اصلی به کلیپ‌بورد"
                                    >
                                      <Copy className="w-3 h-3" />
                                      کپی
                                    </button>
                                  </div>
                                  <div
                                    className="flex-1 p-2 rounded-lg bg-zinc-900/90 border border-zinc-800/80 font-mono text-xs text-zinc-300 overflow-y-auto select-text text-left leading-relaxed"
                                    dir="ltr"
                                  >
                                    {activeEntry.source}
                                  </div>
                                </div>

                                {/* Persian Translation Textarea */}
                                <div className="flex flex-col gap-1 min-h-0">
                                  <div className="flex items-center justify-between text-[11px] text-zinc-400">
                                    <span className="flex items-center gap-1">
                                      <Languages className="w-3 h-3 text-emerald-400" />
                                      ترجمه فارسی (Persian - Vazirmatn):
                                    </span>
                                    <button
                                      type="button"
                                      onClick={() => handleUpdateTranslation(activeEntry.id, activeEntry.source)}
                                      disabled={activeEntry.read_only}
                                      className="text-[10px] text-indigo-400 hover:text-indigo-300 disabled:opacity-50 flex items-center gap-1"
                                      title="کپی کردن متن انگلیسی به فیلد ترجمه جهت ویرایش"
                                    >
                                      <Copy className="w-3 h-3" />
                                      کپی از منبع
                                    </button>
                                  </div>
                                  <textarea
                                    value={activeEntry.translation || ""}
                                    disabled={activeEntry.read_only}
                                    onChange={(e) => handleUpdateTranslation(activeEntry.id, e.target.value)}
                                    className={`flex-1 p-2 rounded-lg text-xs leading-relaxed transition-all resize-none ${
                                      activeEntry.read_only
                                        ? "bg-zinc-900/30 border-zinc-800 text-zinc-500 cursor-not-allowed"
                                        : "bg-zinc-900/90 border border-zinc-700 text-zinc-100 focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500"
                                    }`}
                                    dir="rtl"
                                    placeholder={activeEntry.read_only ? "رشته فقط‌خواندنی است و به بازی تزریق نخواهد شد." : "ترجمه فارسی را اینجا بنویسید..."}
                                  />
                                </div>
                              </div>

                              {/* Bottom bar of active entry */}
                              <div className="flex items-center justify-between text-[11px] pt-1 border-t border-zinc-900">
                                <div className="flex items-center gap-3">
                                  {activeEntry.candidate_class && activeEntry.candidate_class !== "high-confidence" && (
                                    <button
                                      type="button"
                                      onClick={() => handleUpdateCandidate(activeEntry.id, "high-confidence")}
                                      className="text-emerald-400 hover:text-emerald-300 hover:underline flex items-center gap-1"
                                    >
                                      <Sparkles className="w-3 h-3" />
                                      تأیید به عنوان کاندیدای معتبر
                                    </button>
                                  )}
                                  <button
                                    type="button"
                                    onClick={() => handleNavigateToAssetExplorer(activeEntry)}
                                    className="text-indigo-400 hover:text-indigo-300 hover:underline flex items-center gap-1"
                                    title="مشاهده و ردیابی مستقیم شیء در کاوشگر دارایی‌های کامپایل‌شده"
                                  >
                                    <FolderSearch className="w-3.5 h-3.5" />
                                    <span>کاوش شیء در ساختار دارایی‌ها</span>
                                  </button>
                                </div>
                                <div className="text-[10px] text-zinc-500 font-mono">
                                  {activeEntry.translation?.length || 0} کاراکتر فارسی
                                </div>
                              </div>
                            </div>
                          ) : (
                            <div className="h-full flex items-center justify-center text-zinc-500 text-xs">
                              برای مشاهده و ویرایش متن، یک ردیف از جدول یا پکیج را انتخاب کنید.
                            </div>
                          )}
                        </div>
                      </div>
                      </div>

                      {/* Pagination for Translation Candidates */}
                      <div className="flex items-center gap-1.5 pt-1">
                        <Button 
                          onClick={() => setCurrentPage((p) => Math.max(1, p - 1))} 
                          disabled={currentPage <= 1}
                          variant="outline" 
                          size="sm" 
                          className="h-8 px-2.5 bg-zinc-900 border-zinc-800 text-zinc-300"
                        >
                          <ChevronRight className="w-4 h-4" />
                          <span>قبلی</span>
                        </Button>
                        <span className="text-xs text-zinc-400 px-2 font-mono">
                          صفحه {currentPage} از {totalPages}
                        </span>
                        <Button 
                          onClick={() => setCurrentPage((p) => Math.min(totalPages, p + 1))} 
                          disabled={currentPage >= totalPages}
                          variant="outline" 
                          size="sm" 
                          className="h-8 px-2.5 bg-zinc-900 border-zinc-800 text-zinc-300"
                        >
                          <span>بعدی</span>
                          <ChevronLeft className="w-4 h-4" />
                        </Button>
                      </div>
                    </div>
                    )}

                    {/* View B: Real Compiled Asset & Serialized Object Explorer */}
                    {workbenchView === "explorer" && (
                      <div className="flex flex-col lg:flex-row gap-3.5 h-[620px]">
                        {/* Left Column: Real Asset & Container Tree */}
                        <div className="w-full lg:w-72 xl:w-80 shrink-0 bg-zinc-950/70 border border-zinc-800/80 rounded-xl flex flex-col overflow-hidden shadow-xl">
                          <div className="p-2.5 border-b border-zinc-800 bg-zinc-900/60 flex items-center justify-between">
                            <div className="flex items-center gap-2 text-xs font-semibold text-zinc-200">
                              <FolderSearch className="w-4 h-4 text-emerald-400" />
                              <span>درخت دارایی‌های بازی</span>
                            </div>
                            <Button
                              onClick={() => handleBrowseAssets(false)}
                              disabled={isBrowsingAssets}
                              variant="ghost"
                              size="sm"
                              className="h-6 px-1.5 text-[10px] text-zinc-400 hover:text-white"
                            >
                              <RefreshCw className={`w-3 h-3 ${isBrowsingAssets ? "animate-spin" : ""}`} />
                            </Button>
                          </div>

                          <div className="p-2 border-b border-zinc-800/70 bg-zinc-950/40">
                            <Input
                              value={assetSearch}
                              onChange={(e) => setAssetSearch(e.target.value)}
                              placeholder="جستجو در نام فایل‌ها و پوشه‌ها..."
                              className="h-8 text-xs bg-zinc-900/80 border-zinc-800 placeholder:text-zinc-600"
                            />
                          </div>

                          <div className="flex-1 overflow-y-auto p-2 space-y-1">
                            {assetView ? (
                              renderAssetTree(assetView.root_nodes, 0)
                            ) : (
                              <div className="p-4 text-center text-xs text-zinc-500 space-y-2">
                                <p>درخت دارایی‌ها هنوز کاوش نشده است.</p>
                                <Button
                                  onClick={() => handleBrowseAssets(false)}
                                  disabled={isBrowsingAssets}
                                  size="sm"
                                  className="h-7 text-xs bg-emerald-600 hover:bg-emerald-500 text-white"
                                >
                                  {isBrowsingAssets ? "در حال کاوش..." : "کاوش دارایی‌های بازی"}
                                </Button>
                              </div>
                            )}
                          </div>
                        </div>

                        {/* Right Column: Compiled Object & Property Inspector */}
                        <div className="flex-1 flex flex-col gap-3 min-w-0 bg-zinc-950/70 border border-zinc-800/80 rounded-xl p-3 shadow-xl overflow-hidden">
                          {selectedAssetNode ? (
                            <div className="flex-1 flex flex-col gap-2.5 min-h-0">
                              {/* Selected Asset Header */}
                              <div className="flex flex-wrap items-center justify-between gap-2 p-2.5 rounded-lg bg-zinc-900/80 border border-zinc-800 text-xs shrink-0">
                                <div className="flex items-center gap-2 min-w-0">
                                  <Box className="w-4 h-4 text-purple-400 shrink-0" />
                                  <span className="font-bold text-white truncate" title={selectedAssetNode.name}>
                                    {selectedAssetNode.name}
                                  </span>
                                  <span className="text-[10px] font-mono text-zinc-400 truncate" dir="ltr" title={selectedAssetNode.rel_path}>
                                    ({selectedAssetNode.rel_path})
                                  </span>
                                </div>

                                <div className="flex items-center gap-2 shrink-0">
                                  <Badge variant="outline" className="text-[10px] font-mono uppercase bg-zinc-800 text-indigo-300 border-zinc-700">
                                    {selectedAssetNode.format || "Asset"}
                                  </Badge>
                                  {selectedAssetNode.read_only ? (
                                    <Badge variant="outline" className="text-[10px] bg-amber-500/10 text-amber-300 border-amber-500/30 flex items-center gap-1">
                                      <Lock className="w-2.5 h-2.5" /> فقط‌خواندنی
                                    </Badge>
                                  ) : (
                                    <Badge variant="outline" className="text-[10px] bg-emerald-500/10 text-emerald-400 border-emerald-500/30 flex items-center gap-1">
                                      <Check className="w-2.5 h-2.5" /> قابل تزریق
                                    </Badge>
                                  )}
                                  <Button
                                    onClick={() => handleInspectCompiledAsset(selectedAssetNode.rel_path, 1)}
                                    disabled={isLoadingAssetInspection}
                                    variant="outline"
                                    size="sm"
                                    className="h-7 px-2 text-xs bg-zinc-800 border-zinc-700 text-zinc-200 hover:text-white flex items-center gap-1"
                                    title="بازخوانی ساختار اشیای باینری فایل"
                                  >
                                    <RefreshCw className={`w-3 h-3 ${isLoadingAssetInspection ? "animate-spin text-emerald-400" : ""}`} />
                                    <span>بررسی اشیا</span>
                                  </Button>
                                </div>
                              </div>

                              {/* Search & Filter Toolbar for Objects */}
                              <div className="flex flex-wrap items-center justify-between gap-2 shrink-0">
                                <div className="flex items-center gap-2 flex-1 min-w-[200px]">
                                  <div className="relative flex-1">
                                    <Search className="w-3.5 h-3.5 text-zinc-500 absolute right-2.5 top-2.5" />
                                    <Input
                                      value={inspectedObjectSearch}
                                      onChange={(e) => setInspectedObjectSearch(e.target.value)}
                                      placeholder="جستجو در نام شیء، نوع Unity، شناسه یا پراپرتی..."
                                      className="pr-8 h-8 text-xs bg-zinc-900 border-zinc-700 text-zinc-200 placeholder:text-zinc-500"
                                    />
                                  </div>
                                  <label className="flex items-center gap-1.5 text-xs text-zinc-300 cursor-pointer select-none shrink-0 bg-zinc-900 px-2 py-1 rounded border border-zinc-800">
                                    <input
                                      type="checkbox"
                                      checked={inspectedFilterTextOnly}
                                      onChange={(e) => setInspectedFilterTextOnly(e.target.checked)}
                                      className="rounded bg-zinc-800 border-zinc-700 text-indigo-600 focus:ring-0"
                                    />
                                    <span>فقط اشیای متنی</span>
                                  </label>
                                </div>

                                {assetInspection && (
                                  <div className="flex items-center gap-2 text-xs text-zinc-400 font-mono shrink-0">
                                    <span>کل اشیا: {assetInspection.total_objects}</span>
                                    <span>•</span>
                                    <span className="text-emerald-400">
                                      متنی در این صفحه: {assetInspection.objects?.filter((o) => o.text_bearing).length || 0}
                                    </span>
                                  </div>
                                )}
                              </div>

                              {/* Objects Table / List */}
                              <div className="flex-1 overflow-auto rounded-xl border border-zinc-800/80 bg-zinc-950/80 p-2 space-y-2">
                                {isLoadingAssetInspection ? (
                                  <div className="h-full flex flex-col items-center justify-center text-zinc-400 text-xs py-16 gap-2">
                                    <RefreshCw className="w-6 h-6 animate-spin text-indigo-400" />
                                    <span>در حال بازخوانی ساختار اشیا به صورت صفحه‌بندی‌شده...</span>
                                  </div>
                                ) : !assetInspection || !assetInspection.objects || assetInspection.objects.length === 0 ? (
                                  <div className="h-full flex flex-col items-center justify-center text-zinc-500 text-xs py-16 gap-3">
                                    <Database className="w-8 h-8 text-zinc-600" />
                                    <span>برای مشاهده Serialized Objects این فایل، دکمه «بررسی اشیا» را بزنید.</span>
                                    <Button
                                      onClick={() => handleInspectCompiledAsset(selectedAssetNode.rel_path, 1)}
                                      size="sm"
                                      className="bg-indigo-600 hover:bg-indigo-500 text-white text-xs"
                                    >
                                      کاوش اشیای باینری فایل
                                    </Button>
                                  </div>
                                ) : (
                                  assetInspection.objects
                                    .filter((obj) => {
                                      if (inspectedFilterTextOnly && !obj.text_bearing) return false;
                                      if (inspectedObjectSearch) {
                                        const s = inspectedObjectSearch.toLowerCase();
                                        const match =
                                          obj.name.toLowerCase().includes(s) ||
                                          obj.type.toLowerCase().includes(s) ||
                                          String(obj.path_id).includes(s) ||
                                          (obj.properties && obj.properties.some((p) => p.path.toLowerCase().includes(s) || (p.value && p.value.toLowerCase().includes(s))));
                                        if (!match) return false;
                                      }
                                      return true;
                                    })
                                    .map((obj) => {
                                      const objKey = String(obj.path_id);
                                      const isExpanded = expandedObjectIds[objKey] ?? obj.text_bearing;
                                      return (
                                        <div key={objKey} className="rounded-lg border border-zinc-800 bg-zinc-900/60 overflow-hidden text-xs">
                                          {/* Object Header */}
                                          <div
                                            onClick={() => setExpandedObjectIds((prev) => ({ ...prev, [objKey]: !isExpanded }))}
                                            className="p-2.5 flex items-center justify-between gap-2 cursor-pointer hover:bg-zinc-800/40 transition-colors"
                                          >
                                            <div className="flex items-center gap-2 min-w-0">
                                              <button type="button" className="text-zinc-500 hover:text-white text-xs">
                                                {isExpanded ? "▼" : "◀"}
                                              </button>
                                              <span className="font-mono text-indigo-400 font-semibold" dir="ltr">
                                                #{obj.path_id}
                                              </span>
                                              <Badge variant="outline" className={`text-[10px] font-mono px-1.5 py-0 ${
                                                obj.type === "MonoBehaviour" ? "bg-purple-500/10 text-purple-300 border-purple-500/30" :
                                                obj.type === "TextAsset" ? "bg-emerald-500/10 text-emerald-300 border-emerald-500/30" :
                                                obj.type === "GameObject" ? "bg-amber-500/10 text-amber-300 border-amber-500/30" :
                                                "bg-zinc-800 text-zinc-300 border-zinc-700"
                                              }`}>
                                                {obj.type}
                                              </Badge>
                                              <span className="font-medium text-zinc-200 truncate" title={obj.name}>
                                                {obj.name || "(بدون نام)"}
                                              </span>
                                            </div>

                                            <div className="flex items-center gap-2 shrink-0">
                                              {obj.text_bearing && (
                                                <Badge variant="outline" className="text-[10px] bg-emerald-500/10 text-emerald-400 border-emerald-500/20">
                                                  دارای متن
                                                </Badge>
                                              )}
                                              {obj.read_only ? (
                                                <span className="text-[10px] text-amber-400 bg-amber-500/10 px-1.5 py-0.5 rounded border border-amber-500/20" title={obj.read_only_reason}>
                                                  فقط‌خواندنی
                                                </span>
                                              ) : obj.can_inject ? (
                                                <span className="text-[10px] text-emerald-400 bg-emerald-500/10 px-1.5 py-0.5 rounded border border-emerald-500/20">
                                                  قابل تزریق
                                                </span>
                                              ) : null}
                                              <span className="text-[10px] text-zinc-500 font-mono">
                                                {(obj.size_bytes / 1024).toFixed(1)} KB
                                              </span>
                                            </div>
                                          </div>

                                          {/* Properties Tree */}
                                          {isExpanded && obj.properties && obj.properties.length > 0 && (
                                            <div className="border-t border-zinc-800/80 bg-zinc-950/70 p-2.5 space-y-1.5">
                                              <div className="text-[10px] text-zinc-400 font-semibold mb-1">
                                                پراپرتی‌های شیء:
                                              </div>
                                              <div className="space-y-1">
                                                {obj.properties.map((prop, pIdx) => {
                                                  const hasText = Boolean(prop.value && prop.value.trim() !== "");
                                                  return (
                                                    <div
                                                      key={pIdx}
                                                      className={`p-2 rounded border flex flex-col sm:flex-row sm:items-center justify-between gap-2 text-xs ${
                                                        hasText
                                                          ? "bg-zinc-900/90 border-indigo-500/30 text-zinc-200"
                                                          : "bg-zinc-900/40 border-zinc-800/60 text-zinc-400"
                                                      }`}
                                                    >
                                                      <div className="min-w-0 flex-1">
                                                        <div className="flex items-center gap-2 font-mono text-[11px]" dir="ltr">
                                                          <span className="text-indigo-300 font-medium">{prop.path}</span>
                                                          {prop.field_name && (
                                                            <span className="text-zinc-600">({prop.field_name})</span>
                                                          )}
                                                          {prop.category && (
                                                            <span className="px-1 py-0.2 rounded text-[9px] bg-purple-500/20 text-purple-300">
                                                              {prop.category}
                                                            </span>
                                                          )}
                                                        </div>
                                                        {prop.value && (
                                                          <div className="mt-1 font-mono text-[11px] text-zinc-300 line-clamp-2 select-text" dir="ltr" title={prop.value}>
                                                            {prop.value}
                                                          </div>
                                                        )}
                                                        {prop.read_only_reason && (
                                                          <div className="mt-0.5 text-[10px] text-amber-400">
                                                            🔒 {prop.read_only_reason}
                                                          </div>
                                                        )}
                                                      </div>

                                                      <div className="flex items-center gap-1.5 shrink-0">
                                                        {hasText && (
                                                          <Button
                                                            onClick={() => handleNavigateToCandidate(selectedAssetNode.rel_path, prop.path)}
                                                            variant="outline"
                                                            size="sm"
                                                            className="h-6 px-2 text-[10px] bg-indigo-950/60 border-indigo-500/40 text-indigo-200 hover:bg-indigo-900 flex items-center gap-1"
                                                            title="انتقال به میزکار ترجمه جهت ویرایش این فیلد"
                                                          >
                                                            <Languages className="w-3 h-3" />
                                                            <span>ویرایش در میزکار</span>
                                                          </Button>
                                                        )}
                                                      </div>
                                                    </div>
                                                  );
                                                })}
                                              </div>
                                            </div>
                                          )}
                                        </div>
                                      );
                                    })
                                )}
                              </div>

                              {/* Object Pagination Footer */}
                              {assetInspection && assetInspection.total_objects > 50 && (
                                <div className="flex items-center justify-between gap-2 pt-1 border-t border-zinc-800 text-xs shrink-0">
                                  <div className="flex items-center gap-1.5">
                                    <Button
                                      onClick={() => handleInspectCompiledAsset(selectedAssetNode.rel_path, Math.max(1, inspectionPage - 1))}
                                      disabled={inspectionPage <= 1 || isLoadingAssetInspection}
                                      variant="outline"
                                      size="sm"
                                      className="h-7 px-2 text-xs bg-zinc-900 border-zinc-800 text-zinc-300"
                                    >
                                      <ChevronRight className="w-3.5 h-3.5" />
                                      <span>قبلی</span>
                                    </Button>
                                    <span className="text-xs text-zinc-400 px-2 font-mono">
                                      صفحه {inspectionPage} از {Math.ceil(assetInspection.total_objects / 50)}
                                    </span>
                                    <Button
                                      onClick={() => handleInspectCompiledAsset(selectedAssetNode.rel_path, inspectionPage + 1)}
                                      disabled={inspectionPage >= Math.ceil(assetInspection.total_objects / 50) || isLoadingAssetInspection}
                                      variant="outline"
                                      size="sm"
                                      className="h-7 px-2 text-xs bg-zinc-900 border-zinc-800 text-zinc-300"
                                    >
                                      <span>بعدی</span>
                                      <ChevronLeft className="w-3.5 h-3.5" />
                                    </Button>
                                  </div>
                                  <span className="text-[10px] text-zinc-500">
                                    پارس اشیا به صورت تنبل (Lazy) جهت صرفه‌جویی در مصرف رم
                                  </span>
                                </div>
                              )}
                            </div>
                          ) : (
                            <div className="h-full flex flex-col items-center justify-center text-zinc-500 text-xs py-16 gap-2">
                              <FolderSearch className="w-8 h-8 text-zinc-600" />
                              <span>یک فایل از درخت سمت راست انتخاب کنید تا ساختار و اشیای کامپایل‌شده آن نمایش داده شود.</span>
                            </div>
                          )}
                        </div>
                      </div>
                    )}

                    {/* Step Action Buttons (Bottom Bar) */}
                    <div className="flex flex-wrap items-center justify-end gap-3 pt-2">

                      <div className="flex items-center gap-3">
                        <Button 
                          onClick={() => setCurrentStep("extract")} 
                          variant="ghost" 
                          size="sm"
                          className="text-zinc-400 hover:text-white flex items-center gap-1"
                        >
                          <RotateCcw className="w-3.5 h-3.5" />
                          <span>بازگشت</span>
                        </Button>
                        {isExtractOnly ? (
                          <span className="text-xs text-amber-400">
                            ℹ موتور {engineInfo.engine} در وضعیت فقط-استخراج است؛ تزریق غیرفعال است.
                          </span>
                        ) : null}
                        <Button onClick={() => handleNextStep("inject")} disabled={isExtractOnly} size="lg" className="bg-blue-600 hover:bg-blue-700">
                          {isExtractOnly ? "تزریق در دسترس نیست" : "آماده‌سازی برای تزریق"}
                        </Button>
                      </div>
                    </div>
                  </motion.div>
                )}

                {/* STEP 4: INJECT */}
                {currentStep === "inject" && (
                  <motion.div 
                    key="inject"
                    initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -10 }}
                    className="flex flex-col items-center text-center space-y-8"
                  >
                    <div className="w-24 h-24 rounded-full bg-emerald-500/10 flex items-center justify-center border border-emerald-500/20 shadow-[0_0_50px_rgba(16,185,129,0.1)] relative">
                      <Code2 className="w-12 h-12 text-emerald-400 relative z-10" />
                      {isProcessing && <div className="absolute inset-0 rounded-full border-t-2 border-emerald-500 animate-spin" />}
                    </div>
                    <div className="space-y-2">
                      <h2 className="text-2xl font-bold text-white">تزریق و اعمال تغییرات</h2>
                      <p className="text-zinc-400 max-w-md mx-auto">ترجمه‌ها روی کپی‌های موقت فایل‌های قابل پشتیبانی اعمال می‌شوند؛ فایل اصلی بازی در این مرحله تغییر نمی‌کند.</p>
                      {currentCap?.injection_notes && (
                        <div className="p-3 bg-emerald-500/10 border border-emerald-500/20 rounded-lg text-emerald-300 text-xs max-w-md mx-auto text-right">
                          ℹ <strong>راهنمای ماژول:</strong> {currentCap.injection_notes}
                        </div>
                      )}
                    </div>
                    
                    <div className="w-full max-w-md flex flex-col gap-4">
                      {isProcessing ? (
                        <div className="space-y-2 w-full">
                          <Progress value={progress} className="h-2 bg-zinc-800" />
                          <div className="text-center space-y-1">
                            <p className="text-xs text-emerald-300 font-medium animate-pulse">{stageName || "در حال آماده‌سازی فایل‌های پچ ..."}</p>
                            {stageDetail && <p className="text-[11px] text-zinc-500 font-mono">{stageDetail}</p>}
                          </div>
                        </div>
                      ) : (
                        <Button onClick={() => handleNextStep("patch")} size="lg" className="w-full text-lg h-14 bg-emerald-600 hover:bg-emerald-700 shadow-[0_0_20px_rgba(16,185,129,0.3)] transition-all">
                          شروع تزریق به بازی
                        </Button>
                      )}
                    </div>
                  </motion.div>
                )}

                {/* STEP 5: PATCH */}
                {currentStep === "patch" && (
                  <motion.div 
                    key="patch"
                    initial={{ opacity: 0, y: 10 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -10 }}
                    className="flex flex-col items-center text-center space-y-8"
                  >
                    <div className="w-24 h-24 rounded-full bg-orange-500/10 flex items-center justify-center border border-orange-500/20 shadow-[0_0_50px_rgba(249,115,22,0.1)]">
                      <Box className="w-12 h-12 text-orange-400" />
                    </div>
                    <div className="space-y-2">
                      <h2 className="text-2xl font-bold text-white">ساخت نصاب و پچ</h2>
                      <p className="text-zinc-400 max-w-md mx-auto">تزریق با موفقیت انجام شد! برای انتشار، می‌توانید نام مترجمان را وارد کنید تا نصاب نهایی ساخته شود.</p>
                    </div>
                    
                    <div className="w-full max-w-md flex flex-col gap-4">
                      <Input 
                        value={patchCredits} 
                        onChange={(e) => setPatchCredits(e.target.value)} 
                        className="py-6 bg-zinc-900/50 border-zinc-800 text-center text-lg focus-visible:ring-orange-500"
                        placeholder="نام مترجم / تیم (مثلا: تیم ترجمه فلان)"
                      />
                      
                      {isProcessing ? (
                        <div className="space-y-2 w-full">
                          <Progress value={progress} className="h-2 bg-zinc-800" />
                          <div className="text-center space-y-1">
                            <p className="text-xs text-orange-300 font-medium animate-pulse">{stageName || "در حال ساخت نصاب..."}</p>
                            {stageDetail && <p className="text-[11px] text-zinc-500 font-mono">{stageDetail}</p>}
                          </div>
                        </div>
                      ) : (
                        <Button onClick={() => handleNextStep("success")} size="lg" className="w-full text-lg h-14 bg-orange-600 hover:bg-orange-700 shadow-[0_0_20px_rgba(249,115,22,0.3)] transition-all">
                          ساخت پچ
                        </Button>
                      )}
                    </div>
                  </motion.div>
                )}

                {/* STEP 6: SUCCESS */}
                {currentStep === "success" && (
                  <motion.div 
                    key="success"
                    initial={{ opacity: 0, scale: 0.95 }} animate={{ opacity: 1, scale: 1 }}
                    className="flex flex-col items-center text-center space-y-8"
                  >
                    <div className="w-24 h-24 rounded-full bg-green-500/20 flex items-center justify-center border border-green-500/30 shadow-[0_0_80px_rgba(34,197,94,0.2)]">
                      <Rocket className="w-12 h-12 text-green-400" />
                    </div>
                    <div className="space-y-2">
                      <h2 className="text-3xl font-black text-white">بستهٔ فارسی‌ساز آماده شد</h2>
                      <p className="text-zinc-400 max-w-md mx-auto text-lg">بسته ساخته شده است؛ برای اعمال آن، فایل نصب‌کننده را اجرا کنید و پوشهٔ بازی را انتخاب کنید.</p>
                      {patchResult && (
                        <div className="max-w-md mx-auto rounded-lg border border-zinc-800 bg-zinc-900/70 p-3 text-sm text-zinc-300">
                          <div>{patchResult.target_count} فایل در پچ قرار گرفت.</div>
                          <div className="mt-1 break-all font-mono text-xs text-zinc-500" dir="ltr">{patchResult.output_dir}</div>
                        </div>
                      )}
                    </div>
                    
                    <div className="flex gap-4 mt-4">
                      <Button onClick={() => setCurrentStep("detect")} variant="outline" className="border-zinc-700 hover:bg-zinc-800">
                        شروع پروژه‌ی جدید
                      </Button>
                    </div>
                  </motion.div>
                )}

              </AnimatePresence>
            </CardContent>
          </Card>
        </div>
      </div>

      {/* In-app Notification / Toast Banner */}
      <AnimatePresence>
        {notification && (
          <motion.div
            initial={{ opacity: 0, y: -20, scale: 0.98 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -20, scale: 0.98 }}
            className={`fixed top-6 left-1/2 -translate-x-1/2 z-50 max-w-xl w-[90%] p-4 rounded-xl shadow-2xl backdrop-blur-xl border flex items-start gap-3.5 ${
              notification.type === "success"
                ? "bg-emerald-950/90 border-emerald-500/30 text-emerald-200"
                : notification.type === "error"
                ? "bg-rose-950/90 border-rose-500/30 text-rose-200"
                : notification.type === "warning"
                ? "bg-amber-950/90 border-amber-500/30 text-amber-200"
                : "bg-blue-950/90 border-blue-500/30 text-blue-200"
            }`}
            dir="rtl"
          >
            {notification.type === "success" && <CheckCircle2 className="w-5 h-5 text-emerald-400 shrink-0 mt-0.5" />}
            {notification.type === "error" && <XCircle className="w-5 h-5 text-rose-400 shrink-0 mt-0.5" />}
            {notification.type === "warning" && <AlertTriangle className="w-5 h-5 text-amber-400 shrink-0 mt-0.5" />}
            {notification.type === "info" && <Info className="w-5 h-5 text-blue-400 shrink-0 mt-0.5" />}

            <div className="flex-1 text-sm space-y-1">
              <div className="font-bold">{notification.title}</div>
              <div className="text-xs whitespace-pre-wrap opacity-90 leading-relaxed">{notification.message}</div>
              {notification.diagnostic && (
                <div className="pt-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => {
                      if (notification.diagnostic) {
                        navigator.clipboard.writeText(notification.diagnostic);
                        setCopiedDiag(true);
                        setTimeout(() => setCopiedDiag(false), 2000);
                      }
                    }}
                    className="text-xs h-7 gap-1.5 border-zinc-700 bg-zinc-900/60 hover:bg-zinc-800 text-zinc-200"
                  >
                    {copiedDiag ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                    <span>{copiedDiag ? "گزارش خطا کپی شد" : "کپی گزارش کامل خطا (Diagnostics)"}</span>
                  </Button>
                </div>
              )}
            </div>

            <button
              onClick={() => setNotification(null)}
              className="text-zinc-400 hover:text-white transition-colors p-1"
              title="بستن"
            >
              <X className="w-4 h-4" />
            </button>
          </motion.div>
        )}
      </AnimatePresence>

      {/* Settings Modal */}
      <AnimatePresence>
        {showSettingsModal && settingsData && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm"
          >
            <motion.div
              initial={{ scale: 0.95, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.95, opacity: 0 }}
              className="bg-zinc-950 border border-zinc-800 rounded-2xl max-w-lg w-full p-6 shadow-2xl space-y-5 text-right"
              dir="rtl"
            >
              <div className="flex items-center justify-between border-b border-zinc-800 pb-3">
                <div className="flex items-center gap-2">
                  <Settings2 className="w-5 h-5 text-indigo-400" />
                  <h3 className="text-lg font-bold text-white">تنظیمات FarsiForge</h3>
                </div>
                <button
                  onClick={() => setShowSettingsModal(false)}
                  className="text-zinc-400 hover:text-white"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              <div className="space-y-4 text-sm">
                <div className="space-y-1.5">
                  <label className="text-xs text-zinc-400">مسیر دایرکتوری ابزارها (Tools):</label>
                  <div className="flex gap-2">
                    <Input
                      value={settingsData.tools_dir || ""}
                      onChange={(e) => setSettingsData({ ...settingsData, tools_dir: e.target.value })}
                      placeholder="مسیر پیش‌فرض ابزارها (اختیاری)"
                      className="bg-zinc-900 border-zinc-800 text-xs text-left font-mono"
                      dir="ltr"
                    />
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={async () => {
                        const dir = await SelectDirectory();
                        if (dir) setSettingsData({ ...settingsData, tools_dir: dir });
                      }}
                    >
                      انتخاب
                    </Button>
                  </div>
                </div>

                <div className="space-y-1.5">
                  <label className="text-xs text-zinc-400">مسیر خروجی پچ‌ها (Output):</label>
                  <div className="flex gap-2">
                    <Input
                      value={settingsData.output_dir || ""}
                      onChange={(e) => setSettingsData({ ...settingsData, output_dir: e.target.value })}
                      placeholder="مسیر خروجی پیش‌فرض (اختیاری)"
                      className="bg-zinc-900 border-zinc-800 text-xs text-left font-mono"
                      dir="ltr"
                    />
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={async () => {
                        const dir = await SelectDirectory();
                        if (dir) setSettingsData({ ...settingsData, output_dir: dir });
                      }}
                    >
                      انتخاب
                    </Button>
                  </div>
                </div>

                <div className="space-y-1.5">
                  <label className="text-xs text-zinc-400">نام پیش‌فرض مترجم / تیم (Author):</label>
                  <Input
                    value={settingsData.default_author || ""}
                    onChange={(e) => setSettingsData({ ...settingsData, default_author: e.target.value })}
                    placeholder="مثال: FarsiForge Team"
                    className="bg-zinc-900 border-zinc-800 text-xs"
                  />
                </div>
              </div>

              <div className="flex justify-end gap-3 pt-3 border-t border-zinc-800">
                <Button
                  variant="ghost"
                  onClick={() => setShowSettingsModal(false)}
                  className="text-zinc-400 hover:text-white"
                >
                  انصراف
                </Button>
                <Button
                  onClick={handleSaveSettings}
                  className="bg-indigo-600 hover:bg-indigo-700 text-white"
                >
                  ذخیره تنظیمات
                </Button>
              </div>
            </motion.div>
          </motion.div>
        )}

        {/* Font Inventory & Diagnostics Modal */}
        {showFontModal && fontReport && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/75 backdrop-blur-sm"
          >
            <motion.div
              initial={{ scale: 0.95, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.95, opacity: 0 }}
              className="bg-zinc-950 border border-zinc-800 rounded-2xl max-w-4xl w-full p-6 shadow-2xl space-y-5 text-right flex flex-col max-h-[90vh]"
              dir="rtl"
            >
              {/* Header */}
              <div className="flex items-center justify-between border-b border-zinc-800 pb-3 shrink-0">
                <div className="flex items-center gap-2.5">
                  <Type className="w-5 h-5 text-sky-400" />
                  <div>
                    <h3 className="text-lg font-bold text-white">مدیریت و تحلیل فونت‌های بازی (Font Inventory)</h3>
                    <p className="text-xs text-zinc-400">تحلیل گلیف‌های فارسی، پشتیبانی از اتصالات GSUB و جایگزینی امن فونت</p>
                  </div>
                </div>
                <button
                  onClick={() => setShowFontModal(false)}
                  className="text-zinc-400 hover:text-white p-1 rounded-lg hover:bg-zinc-800"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              {/* Summary Stats */}
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 shrink-0">
                <div className="bg-zinc-900/60 p-3 rounded-xl border border-zinc-800/80">
                  <span className="text-[11px] text-zinc-400 block">کل فونت‌های کشف‌شده</span>
                  <span className="text-xl font-bold text-white">{fontReport.total_fonts_found}</span>
                </div>
                <div className="bg-zinc-900/60 p-3 rounded-xl border border-zinc-800/80">
                  <span className="text-[11px] text-zinc-400 block">سازگار با حروف فارسی</span>
                  <span className="text-xl font-bold text-emerald-400">{fontReport.persian_ready_count}</span>
                </div>
                <div className="bg-zinc-900/60 p-3 rounded-xl border border-zinc-800/80">
                  <span className="text-[11px] text-zinc-400 block">فایل‌های مستقیم TTF/OTF</span>
                  <span className="text-xl font-bold text-sky-400">{fontReport.direct_font_files}</span>
                </div>
                <div className="bg-zinc-900/60 p-3 rounded-xl border border-zinc-800/80">
                  <span className="text-[11px] text-zinc-400 block">ریسورس‌های باینری موتور</span>
                  <span className="text-xl font-bold text-purple-400">{fontReport.engine_resources}</span>
                </div>
              </div>

              {/* Diagnostics & Guidance Banner */}
              <div className="p-3 bg-sky-500/10 border border-sky-500/20 rounded-xl space-y-1 text-xs text-sky-300 shrink-0">
                <div className="font-semibold flex items-center gap-1.5">
                  <Info className="w-4 h-4 text-sky-400 shrink-0" />
                  <span>{fontReport.diagnostic_summary}</span>
                </div>
                <p className="text-[11px] text-sky-200/80 mr-5">
                  {fontReport.recommended_action}
                </p>
              </div>

              {/* Font Items List */}
              <div className="flex-1 overflow-y-auto space-y-3 pr-1">
                {fontReport.items.length === 0 ? (
                  <p className="text-center text-zinc-500 py-8">هیچ فونتی در مسیر بازی یافت نشد.</p>
                ) : (
                  fontReport.items.map((item, idx) => {
                    const isStaged = stagedFonts[item.path];
                    const cov = item.glyph_coverage;
                    return (
                      <div
                        key={item.path || idx}
                        className="bg-zinc-900/50 border border-zinc-800 rounded-xl p-4 space-y-3 transition-colors hover:border-zinc-700"
                      >
                        <div className="flex flex-wrap items-center justify-between gap-2">
                          <div className="flex items-center gap-2">
                            <span className="font-mono text-xs px-2 py-0.5 rounded bg-zinc-800 text-zinc-200 uppercase font-semibold">
                              {item.detected_format}
                            </span>
                            <span className="font-medium text-sm text-white">
                              {item.family || (item.path.split(/[/\\]/).pop() || "Font")}
                            </span>
                            {item.style && (
                              <span className="text-xs text-zinc-400">({item.style})</span>
                            )}
                            <Badge
                              variant="outline"
                              className={`text-[10px] ${
                                item.asset_type === "direct_file"
                                  ? "bg-emerald-500/10 text-emerald-300 border-emerald-500/20"
                                  : item.asset_type === "bitmap_atlas"
                                  ? "bg-amber-500/10 text-amber-300 border-amber-500/20"
                                  : "bg-purple-500/10 text-purple-300 border-purple-500/20"
                              }`}
                            >
                              {item.asset_type === "direct_file"
                                ? "فایل مستقیم وکتور"
                                : item.asset_type === "bitmap_atlas"
                                ? "اطلس بیت‌مپ (BMFont)"
                                : "ریسورس کامپایل‌شده موتور"}
                            </Badge>
                          </div>

                          <div className="flex items-center gap-2">
                            {item.file_size > 0 && (
                              <span className="text-[11px] font-mono text-zinc-500">
                                {(item.file_size / 1024).toFixed(1)} KB
                              </span>
                            )}
                          </div>
                        </div>

                        <div className="text-[11px] font-mono text-zinc-400 text-left truncate" dir="ltr" title={item.path}>
                          {item.path}
                        </div>

                        {/* Glyph coverage details */}
                        {cov ? (
                          <div className="bg-zinc-950/60 rounded-lg p-2.5 border border-zinc-800/60 flex flex-wrap items-center justify-between gap-2 text-xs">
                            <div className="flex flex-wrap items-center gap-3">
                              <span className={`flex items-center gap-1 font-medium ${cov.supports_persian ? "text-emerald-400" : "text-amber-400"}`}>
                                {cov.supports_persian ? <CheckCircle2 className="w-3.5 h-3.5" /> : <AlertTriangle className="w-3.5 h-3.5" />}
                                {cov.supports_persian ? "پشتیبانی کامل حروف فارسی" : "فاقد حروف اختصاصی فارسی (گ چ پ ژ)"}
                              </span>
                              <span className={`text-[11px] ${cov.has_opentype_shaping ? "text-emerald-400" : "text-zinc-500"}`}>
                                {cov.has_opentype_shaping ? "✓ اتصالات GSUB فعال" : "فاقد GSUB Shaping"}
                              </span>
                              <span className="text-[11px] text-zinc-400">
                                {cov.supports_persian_digits ? "✓ ارقام فارسی" : "ارقام لاتین"}
                              </span>
                            </div>
                            <div className="flex items-center gap-2">
                              <span className="text-[11px] text-zinc-400">پوشش گلیف‌ها:</span>
                              <span className="font-mono font-bold text-xs text-white">
                                {cov.coverage_percent.toFixed(0)}%
                              </span>
                            </div>
                          </div>
                        ) : null}

                        {/* Staged or Replace Action */}
                        <div className="flex flex-wrap items-center justify-between gap-2 pt-1 border-t border-zinc-800/60">
                          {isStaged ? (
                            <div className="flex items-center gap-2 text-xs text-emerald-400">
                              <CheckCircle2 className="w-4 h-4 text-emerald-400" />
                              <span>فونت جایگزین استیج شد: <strong>{isStaged.font_family || "Custom Font"}</strong></span>
                              <span className="text-[10px] font-mono text-zinc-500">
                                (SHA: {isStaged.target_sha256.slice(0, 8)}...)
                              </span>
                            </div>
                          ) : (
                            <div className="text-xs text-zinc-400">
                              {item.can_replace ? (
                                <span className="text-zinc-400">
                                  این فونت قابلیت جایگزینی مستقیم با فونت سفارشی فارسی (مانند Vazirmatn) را دارد.
                                </span>
                              ) : (
                                <span className="text-amber-400/90 flex items-center gap-1">
                                  <AlertTriangle className="w-3.5 h-3.5" />
                                  {item.unsupported_reason || "جایگزینی مستقیم این فرمت توسط موتور مسدود است."}
                                </span>
                              )}
                            </div>
                          )}

                          {item.can_replace && (
                            <Button
                              size="sm"
                              variant={isStaged ? "outline" : "default"}
                              onClick={() => handleStageFontReplacement(item.path)}
                              className={`text-xs h-8 ${
                                isStaged
                                  ? "border-emerald-500/40 text-emerald-300 hover:bg-emerald-500/10"
                                  : "bg-sky-600 hover:bg-sky-700 text-white"
                              }`}
                            >
                              <Type className="w-3.5 h-3.5 ml-1.5" />
                              {isStaged ? "تغییر فونت انتخابی..." : "انتخاب فونت جایگزین (TTF/OTF)..."}
                            </Button>
                          )}
                        </div>
                      </div>
                    );
                  })
                )}
              </div>

              {/* Safety Footer & Close */}
              <div className="flex flex-wrap items-center justify-between gap-3 pt-3 border-t border-zinc-800 shrink-0">
                <div className="flex items-center gap-1.5 text-xs text-zinc-500">
                  <CheckCircle2 className="w-4 h-4 text-emerald-400 shrink-0" />
                  <span>
                    امنیت تراکنشی تضمین‌شده: فونت‌های جایگزین منحصراً در پوشه WorkingDir/out آماده‌سازی می‌شوند و فایل‌های بازی دست‌نخورده باقی می‌مانند.
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleScanFonts}
                    disabled={isScanningFonts}
                    className="text-xs border-zinc-700 text-zinc-300 hover:bg-zinc-800"
                  >
                    {isScanningFonts ? "در حال اسکن مجدد..." : "اسکن مجدد"}
                  </Button>
                  <Button
                    onClick={() => setShowFontModal(false)}
                    className="bg-zinc-800 hover:bg-zinc-700 text-white text-xs"
                  >
                    بستن پنجره
                  </Button>
                </div>
              </div>
            </motion.div>
          </motion.div>
        )}

        {/* Compiled Asset Browser & Provenance Modal */}
        {showAssetBrowserModal && assetView && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/80 backdrop-blur-md"
          >
            <motion.div
              initial={{ scale: 0.95, opacity: 0 }}
              animate={{ scale: 1, opacity: 1 }}
              exit={{ scale: 0.95, opacity: 0 }}
              className="bg-zinc-950 border border-zinc-800 rounded-2xl max-w-6xl w-full p-6 shadow-2xl space-y-4 text-right flex flex-col max-h-[92vh]"
              dir="rtl"
            >
              {/* Header */}
              <div className="flex items-center justify-between border-b border-zinc-800 pb-3 shrink-0">
                <div className="flex items-center gap-2.5">
                  <FolderSearch className="w-5 h-5 text-emerald-400" />
                  <div>
                    <h3 className="text-lg font-bold text-white">کاوشگر سلسله‌مراتبی دارایی‌های بازی (Compiled Asset Browser)</h3>
                    <p className="text-xs text-zinc-400">شناسایی فایل‌ها، کانتینرها، منابع کامپایل‌شده و منشأ اصیل متن‌ها (Source Provenance)</p>
                  </div>
                </div>
                <button
                  onClick={() => setShowAssetBrowserModal(false)}
                  className="text-zinc-400 hover:text-white p-1 rounded-lg hover:bg-zinc-800"
                >
                  <X className="w-5 h-5" />
                </button>
              </div>

              {/* Summary Stats */}
              <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 shrink-0">
                <div className="bg-zinc-900/60 p-3 rounded-xl border border-zinc-800/80">
                  <span className="text-[11px] text-zinc-400 block">کل فایل‌های کاوش‌شده</span>
                  <span className="text-xl font-bold text-white">{assetView.total_files}</span>
                </div>
                <div className="bg-zinc-900/60 p-3 rounded-xl border border-zinc-800/80">
                  <span className="text-[11px] text-zinc-400 block">کانتینرها و آرشیوها</span>
                  <span className="text-xl font-bold text-purple-400">{assetView.total_containers}</span>
                </div>
                <div className="bg-zinc-900/60 p-3 rounded-xl border border-zinc-800/80">
                  <span className="text-[11px] text-zinc-400 block">منابع با پارسر معتبر</span>
                  <span className="text-xl font-bold text-emerald-400">{assetView.total_parsed_resources}</span>
                </div>
                <div className="bg-zinc-900/60 p-3 rounded-xl border border-zinc-800/80">
                  <span className="text-[11px] text-zinc-400 block">برآورد متون قابل‌استخراج</span>
                  <span className="text-xl font-bold text-sky-400">{assetView.total_text_records}</span>
                </div>
              </div>

              {/* Filter and Search Bar */}
              <div className="flex flex-wrap items-center justify-between gap-3 shrink-0">
                <div className="flex flex-wrap items-center gap-1.5">
                  <Badge
                    onClick={() => setAssetFilter("all")}
                    variant={assetFilter === "all" ? "default" : "outline"}
                    className="cursor-pointer text-xs"
                  >
                    همه ({assetView.total_files})
                  </Badge>
                  <Badge
                    onClick={() => setAssetFilter("containers")}
                    variant={assetFilter === "containers" ? "default" : "outline"}
                    className={`cursor-pointer text-xs ${assetFilter === "containers" ? "bg-purple-600 text-white" : "text-purple-400 hover:bg-purple-500/10 border-purple-500/30"}`}
                  >
                    کانتینرها و آرشیوها ({assetView.total_containers})
                  </Badge>
                  <Badge
                    onClick={() => setAssetFilter("extractable")}
                    variant={assetFilter === "extractable" ? "default" : "outline"}
                    className={`cursor-pointer text-xs ${assetFilter === "extractable" ? "bg-emerald-600 text-white" : "text-emerald-400 hover:bg-emerald-500/10 border-emerald-500/30"}`}
                  >
                    منابع دارای متن ({assetView.total_parsed_resources})
                  </Badge>
                  <Badge
                    onClick={() => setAssetFilter("readonly")}
                    variant={assetFilter === "readonly" ? "default" : "outline"}
                    className={`cursor-pointer text-xs ${assetFilter === "readonly" ? "bg-amber-600 text-white" : "text-amber-400 hover:bg-amber-500/10 border-amber-500/30"}`}
                  >
                    فقط‌خواندنی
                  </Badge>
                </div>
                <Input
                  value={assetSearch}
                  onChange={(e) => setAssetSearch(e.target.value)}
                  placeholder="جستجو در نام فایل‌ها و مسیرها..."
                  className="bg-zinc-900 border-zinc-800 text-xs w-64"
                />
              </div>

              {/* Main Split View: Tree / List on Right, Inspector on Left */}
              <div className="flex-1 grid grid-cols-1 md:grid-cols-12 gap-4 overflow-hidden min-h-[380px]">
                {/* Tree View (Col 7) */}
                <div className="md:col-span-7 bg-zinc-900/40 border border-zinc-800 rounded-xl p-3 overflow-y-auto space-y-1.5">
                  {renderAssetTree(assetView.root_nodes)}
                </div>

                {/* Details & Provenance Inspector (Col 5) */}
                <div className="md:col-span-5 bg-zinc-900/60 border border-zinc-800 rounded-xl p-4 overflow-y-auto flex flex-col justify-between">
                  {selectedAssetNode ? (
                    <div className="space-y-4">
                      <div className="border-b border-zinc-800 pb-3">
                        <span className="text-[10px] text-zinc-500 uppercase font-mono block">اطلاعات دارایی انتخابی</span>
                        <h4 className="text-base font-bold text-white break-all">{selectedAssetNode.name}</h4>
                        <div className="text-[11px] font-mono text-zinc-400 text-left break-all mt-1" dir="ltr">
                          {selectedAssetNode.rel_path}
                        </div>
                      </div>

                      <div className="space-y-2 text-xs">
                        <div className="flex justify-between items-center py-1 border-b border-zinc-800/60">
                          <span className="text-zinc-400">نوع دارایی:</span>
                          <span className="font-medium text-zinc-200">
                            {selectedAssetNode.type === "container_archive" ? "آرشیو کانتینر" :
                             selectedAssetNode.type === "parsed_resource" ? "منبع استخراج‌شده" :
                             selectedAssetNode.type === "raw_candidate" ? "کاندیدای اسکن خام" :
                             selectedAssetNode.type === "folder" ? "پوشه" : "فایل دارایی"}
                          </span>
                        </div>
                        <div className="flex justify-between items-center py-1 border-b border-zinc-800/60">
                          <span className="text-zinc-400">موتور / فرمت:</span>
                          <span className="font-mono text-indigo-300 uppercase">{selectedAssetNode.format || "N/A"}</span>
                        </div>
                        {selectedAssetNode.size_bytes > 0 && (
                          <div className="flex justify-between items-center py-1 border-b border-zinc-800/60">
                            <span className="text-zinc-400">حجم فایل:</span>
                            <span className="font-mono text-zinc-300">
                              {(selectedAssetNode.size_bytes / 1024).toFixed(1)} KB
                            </span>
                          </div>
                        )}
                        {selectedAssetNode.container && (
                          <div className="flex justify-between items-center py-1 border-b border-zinc-800/60">
                            <span className="text-zinc-400">کانتینر والد:</span>
                            <span className="font-mono text-purple-300 text-[11px]" dir="ltr">{selectedAssetNode.container}</span>
                          </div>
                        )}
                        {selectedAssetNode.text_count > 0 && (
                          <div className="flex justify-between items-center py-1 border-b border-zinc-800/60">
                            <span className="text-zinc-400">تعداد متون استخراجی:</span>
                            <span className="font-bold text-emerald-400">{selectedAssetNode.text_count}</span>
                          </div>
                        )}
                      </div>

                      {/* Write-Back Safety Contract Box */}
                      <div className={`p-3 rounded-xl border space-y-1.5 text-xs ${
                        selectedAssetNode.read_only 
                          ? "bg-amber-500/10 border-amber-500/20 text-amber-300"
                          : "bg-emerald-500/10 border-emerald-500/20 text-emerald-300"
                      }`}>
                        <div className="font-semibold flex items-center gap-1.5">
                          {selectedAssetNode.read_only ? (
                            <Lock className="w-4 h-4 text-amber-400 shrink-0" />
                          ) : (
                            <CheckCircle2 className="w-4 h-4 text-emerald-400 shrink-0" />
                          )}
                          <span>
                            {selectedAssetNode.read_only ? "قرارداد فقط‌خواندنی (Read-Only)" : "قابلیت استخراج و تزریق مستقیم (Writable)"}
                          </span>
                        </div>
                        <p className="text-[11px] leading-relaxed">
                          {selectedAssetNode.read_only 
                            ? (selectedAssetNode.read_only_reason || "این دارایی فاقد مسیر بازنویسی امن است و مستقیماً تغییر داده نمی‌شود.")
                            : "این فایل دارای پارسر معتبر است و رشته‌های آن با امنیت کامل به WorkingDir/out تزریق می‌شوند."}
                        </p>
                      </div>

                      {/* Metadata tags */}
                      {selectedAssetNode.metadata && Object.keys(selectedAssetNode.metadata).length > 0 && (
                        <div className="space-y-1 text-xs">
                          <span className="text-zinc-400 text-[11px]">ویژگی‌های کانتینر / منبع:</span>
                          <div className="flex flex-wrap gap-1">
                            {Object.entries(selectedAssetNode.metadata).map(([k, v]) => (
                              <span key={k} className="px-2 py-0.5 rounded bg-zinc-800 text-zinc-300 text-[10px]">
                                {k}: {v}
                              </span>
                            ))}
                          </div>
                        </div>
                      )}
                    </div>
                  ) : (
                    <div className="h-full flex items-center justify-center text-center text-zinc-500 text-xs py-10">
                      یک فایل یا پوشه را از درخت انتخاب کنید تا جزئیات و اصالت منبع آن نمایش داده شود.
                    </div>
                  )}

                  <div className="pt-3 border-t border-zinc-800 text-[10px] text-zinc-500">
                    ℹ تضمین اصالت: هر متن قابل‌ویرایش به یک منبع مشخص گره خورده و رشته‌های ناشناخته هاردکد نخواهند شد.
                  </div>
                </div>
              </div>

              {/* Safety Footer & Close */}
              <div className="flex flex-wrap items-center justify-between gap-3 pt-3 border-t border-zinc-800 shrink-0">
                <div className="flex items-center gap-1.5 text-xs text-zinc-500">
                  <CheckCircle2 className="w-4 h-4 text-emerald-400 shrink-0" />
                  <span>
                    کاوشگر دارایی‌ها ساختار فایل‌ها را بدون نیاز به استخراج اولیه به صورت درختی نمایش می‌دهد.
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleBrowseAssets(true)}
                    disabled={isBrowsingAssets}
                    className="text-xs border-zinc-700 text-zinc-300 hover:bg-zinc-800"
                  >
                    {isBrowsingAssets ? "در حال کاوش مجدد..." : "کاوش مجدد"}
                  </Button>
                  <Button
                    onClick={() => setShowAssetBrowserModal(false)}
                    className="bg-zinc-800 hover:bg-zinc-700 text-white text-xs"
                  >
                    بستن پنجره
                  </Button>
                </div>
              </div>
            </motion.div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
