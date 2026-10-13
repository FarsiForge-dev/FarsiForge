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
  Wrench
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Progress } from "@/components/ui/progress";
import { Badge } from "@/components/ui/badge";
import { SelectDirectory, DetectEngine, Extract, GetProject, SaveTranslations, Inject, BuildPatcher, GetToolStatus } from "../../wailsjs/go/main/App";
import { core, main } from "../../wailsjs/go/models";

type Step = "detect" | "extract" | "translate" | "inject" | "patch" | "success";

export default function FarsiForgeDashboard() {
  const [currentStep, setCurrentStep] = useState<Step>("detect");
  const [dirPath, setDirPath] = useState("");
  const [isProcessing, setIsProcessing] = useState(false);
  const [progress, setProgress] = useState(0);
  const [gameInfo, setGameInfo] = useState<core.GameInfo | null>(null);
  const [translations, setTranslations] = useState<core.StringEntry[]>([]);
  const [patchResult, setPatchResult] = useState<Awaited<ReturnType<typeof BuildPatcher>> | null>(null);
  const [patchCredits, setPatchCredits] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [toolStatus, setToolStatus] = useState<main.ToolStatus | null>(null);
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
  }, []);

  const engineInfo = {
    engine: gameInfo ? gameInfo.engine : "---",
    backend: gameInfo ? gameInfo.backend : "---",
    confidence: gameInfo ? (gameInfo.confidence * 100).toFixed(0) + "%" : "---",
    files: translations.length > 0 
      ? `${translations.length} متن` 
      : (gameInfo?.evidence?.length ? `${gameInfo.evidence.length} نشانه شناسایی` : "---")
  };

  const isDetectionOnly = gameInfo ? ["ue3", "source"].includes(gameInfo.engine.toLowerCase()) : false;
  const isExtractOnly = gameInfo ? ["sage", "goldsrc", "source2", "factorio", "zomboid", "rage"].includes(gameInfo.engine.toLowerCase()) : false;

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

  // Cap the rendered list so huge projects (100k+ entries) stay responsive.
  const MAX_VISIBLE = 200;
  const visibleEntries = filteredEntries.slice(0, MAX_VISIBLE);

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
    setProgress(50);
    try {
      if (next === "extract") {
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
      alert(String(e));
    } finally {
      setIsProcessing(false);
      setProgress(0);
    }
  };

  return (
    <div className="min-h-screen p-6 flex items-center justify-center relative overflow-hidden">
      <div className="absolute inset-0 bg-[radial-gradient(#312e81_1px,transparent_1px)] [background-size:24px_24px] opacity-20 pointer-events-none" />
      
      <div className="max-w-5xl w-full grid grid-cols-1 lg:grid-cols-12 gap-8 z-10">
        
        {/* Sidebar / Info Panel */}
        <div className="lg:col-span-4 flex flex-col gap-5">
          <div className="flex items-center gap-4 mb-2">
            <div className="p-3 bg-primary/10 rounded-2xl border border-primary/20 backdrop-blur-md">
              <Cpu className="w-8 h-8 text-primary" />
            </div>
            <div>
              <h1 className="text-3xl font-black tracking-tight text-white">FarsiForge</h1>
              <p className="text-zinc-400 font-medium text-sm tracking-widest uppercase">Persian Game Engine</p>
            </div>
          </div>

          <Card className="bg-zinc-950/40 backdrop-blur-xl border-zinc-800/50 shadow-2xl">
            <CardHeader className="py-3 px-4">
              <CardTitle className="text-base flex items-center gap-2">
                <Box className="w-4 h-4 text-indigo-400" /> 
                مشخصات بازی
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 px-4 pb-4 pt-0 text-sm">
              <div className="flex justify-between items-center py-1.5 border-b border-zinc-800/50">
                <span className="text-zinc-400">موتور بازی</span>
                <Badge variant="outline" className="bg-indigo-500/10 text-indigo-300 border-indigo-500/20">{currentStep !== "detect" ? engineInfo.engine : "---"}</Badge>
              </div>
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
                    onClick={() => setPersianOpts((prev) => ({ ...prev, [setting.id]: !prev[setting.id as keyof typeof prev] }))}
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
                          <p className="text-xs text-zinc-500 text-center animate-pulse">در حال استخراج متون و دارایی‌های بازی...</p>
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
                    className="w-full h-full flex flex-col space-y-6"
                  >
                    <div className="flex items-center gap-3">
                      <div className="p-2 bg-blue-500/10 rounded-lg border border-blue-500/20">
                        <Languages className="w-6 h-6 text-blue-400" />
                      </div>
                      <h2 className="text-xl font-bold text-white">مدیریت ترجمه‌ها</h2>
                    </div>

                    <div className="flex gap-2 items-center">
                      <Input
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="bg-zinc-900 border-zinc-700 text-sm text-zinc-200"
                        placeholder="جستجو در متن‌ها (متن، ترجمه، فایل، شناسه، یادداشت)..."
                      />
                      <select
                        value={statusFilter}
                        onChange={(e) => setStatusFilter(e.target.value)}
                        className="bg-zinc-900 border border-zinc-700 rounded-md px-3 py-2 text-sm text-zinc-200 shrink-0"
                      >
                        <option value="all">همه</option>
                        <option value="untranslated">بی‌ترجمه</option>
                        <option value="translated">ترجمه‌شده</option>
                        <option value="approved">تأییدشده</option>
                        <option value="skipped">ردشده</option>
                        <option value="qa">هشدار QA</option>
                      </select>
                    </div>
                    <p className="text-xs text-zinc-500">
                      نمایش {visibleEntries.length} از {filteredEntries.length} مورد
                      {filteredEntries.length !== translations.length && ` (کل: ${translations.length})`}
                      {filteredEntries.length > MAX_VISIBLE && " — برای دیدن موارد بیشتر، جستجو را دقیق‌تر کنید"}
                    </p>

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
                    
                    <div className="flex justify-between items-center pt-2">
                      {isExtractOnly ? (
                        <span className="text-xs text-amber-400">
                          ℹ موتور {engineInfo.engine} در وضعیت فقط-استخراج (Extract-Only) است؛ تزریق خودکار غیرفعال است.
                        </span>
                      ) : <span />}
                      <Button onClick={() => handleNextStep("inject")} disabled={isExtractOnly} size="lg" className="bg-blue-600 hover:bg-blue-700">
                        {isExtractOnly ? "تزریق در دسترس نیست" : "آماده‌سازی برای تزریق"}
                      </Button>
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
                    </div>
                    
                    <div className="w-full max-w-md flex flex-col gap-4">
                      {isProcessing ? (
                        <div className="space-y-2 w-full">
                          <Progress value={progress} className="h-2 bg-zinc-800" />
                          <p className="text-xs text-zinc-500 text-center animate-pulse">در حال آماده‌سازی فایل‌های پچ ...</p>
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
                          <p className="text-xs text-zinc-500 text-center animate-pulse">در حال ساخت نصاب...</p>
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
    </div>
  );
}
