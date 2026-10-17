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
  Info
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
  SaveSettings
} from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime";
import { core, main } from "../../wailsjs/go/models";

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
  const [persianOpts, setPersianOpts] = useState({
    rtl: true,
    shaping: true,
    digits: true,
    vazirmatn: true,
  });

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

  // Filtered entry list for the translate step: case-insensitive search
  // across id/source/translation/file/notes plus a status filter ("qa"
  // matches entries carrying QA notes). Original indices are preserved so
  // edits map back onto the full translations array.
  const filteredEntries = useMemo(() => {
    const q = searchQuery.trim().toLowerCase();
    return translations
      .map((item, idx) => ({ item, idx }))
      .filter(({ item }) => {
        if (statusFilter === "qa") {
          if (!(item.notes || "").trim()) return false;
        } else if (statusFilter !== "all" && item.status !== statusFilter) {
          return false;
        }
        if (q) {
          const hay = [item.id, item.source, item.translation, item.file, item.notes]
            .filter(Boolean)
            .join("\n")
            .toLowerCase();
          if (!hay.includes(q)) return false;
        }
        return true;
      });
  }, [translations, searchQuery, statusFilter]);

  // Reset page when search or filter changes
  useEffect(() => {
    setCurrentPage(1);
  }, [searchQuery, statusFilter]);

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
      
      <div className="max-w-5xl w-full grid grid-cols-1 lg:grid-cols-12 gap-8 z-10">
        
        {/* Sidebar / Info Panel */}
        <div className="lg:col-span-4 flex flex-col gap-5">
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
        </div>

        {/* Main Work Area */}
        <div className="lg:col-span-8 flex flex-col">
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
                        <div className="space-y-2 w-full">
                          <Progress value={progress} className="h-2 bg-zinc-800" />
                          <div className="text-center space-y-1">
                            <p className="text-xs text-purple-300 font-medium animate-pulse">{stageName || "در حال استخراج متون و دارایی‌های بازی..."}</p>
                            {stageDetail && <p className="text-[11px] text-zinc-500 font-mono">{stageDetail}</p>}
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
                      
                      {/* Action buttons: Export, Import, Save */}
                      <div className="flex items-center gap-2">
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

                    <div className="flex gap-2 items-center">
                      <Input
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="bg-zinc-900 border-zinc-700 text-sm text-zinc-200 flex-1"
                        placeholder="جستجو در متن‌ها (متن اصلی، ترجمه، فایل، شناسه، یادداشت)..."
                      />
                      <select
                        value={statusFilter}
                        onChange={(e) => setStatusFilter(e.target.value)}
                        className="bg-zinc-900 border border-zinc-700 rounded-md px-3 py-2 text-sm text-zinc-200 shrink-0"
                      >
                        <option value="all">همه وضعیت‌ها</option>
                        <option value="untranslated">بی‌ترجمه</option>
                        <option value="translated">ترجمه‌شده</option>
                        <option value="approved">تأییدشده</option>
                        <option value="skipped">ردشده</option>
                        <option value="qa">هشدار QA ({qaIssuesCount})</option>
                      </select>
                    </div>

                    <div className="flex justify-between items-center text-xs text-zinc-500">
                      <div>
                        نمایش {filteredEntries.length === 0 ? 0 : (currentPage - 1) * PAGE_SIZE + 1} تا {Math.min(currentPage * PAGE_SIZE, filteredEntries.length)} از {filteredEntries.length} مورد
                        {filteredEntries.length !== translations.length && ` (کل دیتابیس: ${translations.length})`}
                      </div>
                      {qaIssuesCount > 0 && (
                        <Badge 
                          onClick={() => setStatusFilter("qa")}
                          variant="outline" 
                          className="cursor-pointer bg-amber-500/10 text-amber-300 border-amber-500/30 flex items-center gap-1 hover:bg-amber-500/20"
                        >
                          <AlertTriangle className="w-3 h-3" />
                          {qaIssuesCount} خطای متغیر/ساختار در ترجمه
                        </Badge>
                      )}
                    </div>

                    <div className="flex-1 bg-zinc-900/50 rounded-xl border border-zinc-800 p-4 space-y-3 overflow-y-auto max-h-[300px]">
                      {visibleEntries.length === 0 && <p className="text-center text-zinc-500 py-10">متنی یافت نشد.</p>}
                      {visibleEntries.map(({ item, idx }) => (
                        <div key={item.id || idx} className="grid grid-cols-2 gap-4 p-3 rounded-lg bg-zinc-950/50 border border-zinc-800/50">
                          <div className="space-y-1 min-w-0">
                            <div className="flex items-center justify-between gap-2 text-xs text-zinc-500">
                              <span className="truncate text-indigo-300" title={item.context || "متن عمومی"}>{item.context || "متن عمومی"}</span>
                              <span className="truncate font-mono text-left" dir="ltr" title={`${item.file}${item.path ? ` · ${item.path}` : ""}`}>
                                {item.file}{item.path ? ` · ${item.path}` : ""}
                              </span>
                            </div>
                            {item.context === "MonoBehaviour" && item.path?.startsWith("raw_") && (
                              <p className="text-xs text-amber-400" title="فیلد منبع از روی فایل باینری قابل‌تأیید نیست؛ تزریق این متن پشتیبانی نمی‌شود.">
                                اسکن خام؛ محل فیلد تأییدنشده و فعلاً غیرقابل‌تزریق
                              </p>
                            )}
                            {item.context === "MonoBehaviour raw serialized string (typetree unavailable)" && (
                              <p className="text-xs text-sky-400" title="رشته و offset باینری تأیید شده‌اند؛ نام فیلد بازیابی نشده است.">
                                رشتهٔ سریال‌شده با offset دقیق؛ نام فیلد در دسترس نیست
                              </p>
                            )}
                            {(item.file?.endsWith(".uexp") || item.context?.includes(".uexp")) && (
                              <p className="text-xs text-amber-400" title="این متن در دارایی کامپایل‌شده بلوپرینت/ویجت هاردکد شده است و جدول locres ندارد؛ تزریق باینری دارایی‌های کامپایل‌شده uexp پشتیبانی نمی‌شود.">
                                متن دارایی کامپایل‌شده (.uexp)؛ فاقد جدول locres (فقط‌خواندنی)
                              </p>
                            )}
                            <div className="text-left font-mono text-sm text-zinc-400" dir="ltr">{item.source}</div>
                            {item.notes && (
                              <div className="text-xs text-amber-400 truncate" dir="ltr" title={item.notes}>⚠ {item.notes}</div>
                            )}
                          </div>
                          <Input
                            value={item.translation || ""}
                            onChange={(e) => {
                              const v = e.target.value;
                              setTranslations((prev) => prev.map((t, i) => (i === idx ? { ...t, translation: v } : t)));
                            }}
                            className="bg-zinc-900 border-zinc-700 text-right text-sm text-zinc-200"
                            placeholder="ترجمه..."
                          />
                        </div>
                      ))}
                    </div>

                    {/* Pagination Controls */}
                    <div className="flex justify-between items-center py-1">
                      <div className="flex items-center gap-1">
                        <Button 
                          onClick={() => setCurrentPage((p) => Math.max(1, p - 1))} 
                          disabled={currentPage <= 1}
                          variant="outline" 
                          size="sm" 
                          className="h-8 px-2 bg-zinc-900 border-zinc-800 text-zinc-300"
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
                          className="h-8 px-2 bg-zinc-900 border-zinc-800 text-zinc-300"
                        >
                          <span>بعدی</span>
                          <ChevronLeft className="w-4 h-4" />
                        </Button>
                      </div>

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
      </AnimatePresence>
    </div>
  );
}
