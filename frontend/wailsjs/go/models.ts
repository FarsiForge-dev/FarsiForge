export namespace core {
	
	export class GameInfo {
	    engine: string;
	    backend?: string;
	    version?: string;
	    game_name: string;
	    game_exe?: string;
	    game_root: string;
	    data_path?: string;
	    platform?: string;
	    arch?: string;
	    confidence: number;
	    evidence?: string[];
	    metadata?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new GameInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.engine = source["engine"];
	        this.backend = source["backend"];
	        this.version = source["version"];
	        this.game_name = source["game_name"];
	        this.game_exe = source["game_exe"];
	        this.game_root = source["game_root"];
	        this.data_path = source["data_path"];
	        this.platform = source["platform"];
	        this.arch = source["arch"];
	        this.confidence = source["confidence"];
	        this.evidence = source["evidence"];
	        this.metadata = source["metadata"];
	    }
	}
	export class EngineCapability {
	    engine: string;
	    display_name: string;
	    can_detect: boolean;
	    can_extract: boolean;
	    can_inject: boolean;
	    archive_rebuild: boolean;
	    font_support: boolean;
	    patch_install: boolean;
	    status: string;
	    extraction_notes?: string;
	    injection_notes?: string;
	    required_tools?: string[];

	    static createFrom(source: any = {}) {
	        return new EngineCapability(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.engine = source["engine"];
	        this.display_name = source["display_name"];
	        this.can_detect = source["can_detect"];
	        this.can_extract = source["can_extract"];
	        this.can_inject = source["can_inject"];
	        this.archive_rebuild = source["archive_rebuild"];
	        this.font_support = source["font_support"];
	        this.patch_install = source["patch_install"];
	        this.status = source["status"];
	        this.extraction_notes = source["extraction_notes"];
	        this.injection_notes = source["injection_notes"];
	        this.required_tools = source["required_tools"];
	    }
	}
	export class PersianOptions {
	    reshape: boolean;
	    bidi_reorder: boolean;
	    fix_yeh: boolean;
	    persian_digits: boolean;
	    convert_punct: boolean;
	    drop_diacritics: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PersianOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.reshape = source["reshape"];
	        this.bidi_reorder = source["bidi_reorder"];
	        this.fix_yeh = source["fix_yeh"];
	        this.persian_digits = source["persian_digits"];
	        this.convert_punct = source["convert_punct"];
	        this.drop_diacritics = source["drop_diacritics"];
	    }
	}
	export class StringEntry {
	    id: string;
	    source: string;
	    translation?: string;
	    file: string;
	    container?: string;
	    path: string;
	    context?: string;
	    speaker?: string;
	    line?: number;
	    status: string;
	    notes?: string;
	    max_length?: number;
	    category?: string;
	    candidate_class?: string;
	    confidence?: number;
	    read_only?: boolean;
	    read_only_reason?: string;
	    source_type?: string;
	
	    static createFrom(source: any = {}) {
	        return new StringEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.source = source["source"];
	        this.translation = source["translation"];
	        this.file = source["file"];
	        this.container = source["container"];
	        this.path = source["path"];
	        this.context = source["context"];
	        this.speaker = source["speaker"];
	        this.line = source["line"];
	        this.status = source["status"];
	        this.notes = source["notes"];
	        this.max_length = source["max_length"];
	        this.category = source["category"];
	        this.candidate_class = source["candidate_class"];
	        this.confidence = source["confidence"];
	        this.read_only = source["read_only"];
	        this.read_only_reason = source["read_only_reason"];
	        this.source_type = source["source_type"];
	    }
	}
	export class ExtractionStats {
	    total_parsed: number;
	    accepted_count: number;
	    high_confidence_count: number;
	    needs_review_count: number;
	    technical_count: number;
	    raw_heuristic_count: number;
	    read_only_count: number;
	    filtered_by_reason?: Record<string, number>;

	    static createFrom(source: any = {}) {
	        return new ExtractionStats(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total_parsed = source["total_parsed"];
	        this.accepted_count = source["accepted_count"];
	        this.high_confidence_count = source["high_confidence_count"];
	        this.needs_review_count = source["needs_review_count"];
	        this.technical_count = source["technical_count"];
	        this.raw_heuristic_count = source["raw_heuristic_count"];
	        this.read_only_count = source["read_only_count"];
	        this.filtered_by_reason = source["filtered_by_reason"];
	    }
	}
	export class ValidationError {
	    entry_id: string;
	    type: string;
	    message: string;
	    severity: string;
	    details?: string;

	    static createFrom(source: any = {}) {
	        return new ValidationError(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entry_id = source["entry_id"];
	        this.type = source["type"];
	        this.message = source["message"];
	        this.severity = source["severity"];
	        this.details = source["details"];
	    }
	}
	export class Project {
	    name: string;
	    game_name: string;
	    game_root: string;
	    engine: string;
	    game_exe?: string;
	    backend?: string;
	    version?: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    entries: StringEntry[];
	    working_dir?: string;
	    project_dir?: string;
	    persian_opts: PersianOptions;
	    font_path?: string;
	    extracted_files?: string[];
	    modified_files?: string[];
	    modified_file_hashes?: Record<string, string>;
	    extraction_stats?: ExtractionStats;
	
	    static createFrom(source: any = {}) {
	        return new Project(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.game_name = source["game_name"];
	        this.game_root = source["game_root"];
	        this.engine = source["engine"];
	        this.game_exe = source["game_exe"];
	        this.backend = source["backend"];
	        this.version = source["version"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.entries = this.convertValues(source["entries"], StringEntry);
	        this.working_dir = source["working_dir"];
	        this.project_dir = source["project_dir"];
	        this.persian_opts = this.convertValues(source["persian_opts"], PersianOptions);
	        this.font_path = source["font_path"];
	        this.extracted_files = source["extracted_files"];
	        this.modified_files = source["modified_files"];
	        this.modified_file_hashes = source["modified_file_hashes"];
	        this.extraction_stats = this.convertValues(source["extraction_stats"], ExtractionStats);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace main {
	
	export class BuildPatcherResult {
	    output_dir: string;
	    patch_file: string;
	    patcher_exe: string;
	    target_count: number;
	
	    static createFrom(source: any = {}) {
	        return new BuildPatcherResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.output_dir = source["output_dir"];
	        this.patch_file = source["patch_file"];
	        this.patcher_exe = source["patcher_exe"];
	        this.target_count = source["target_count"];
	    }
	}
	export class Settings {
	    tools_dir: string;
	    output_dir: string;
	    game_search_paths: string[];
	    default_persian_opts: core.PersianOptions;
	    default_author: string;
	    default_language: string;
	    log_level: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tools_dir = source["tools_dir"];
	        this.output_dir = source["output_dir"];
	        this.game_search_paths = source["game_search_paths"];
	        this.default_persian_opts = this.convertValues(source["default_persian_opts"], core.PersianOptions);
	        this.default_author = source["default_author"];
	        this.default_language = source["default_language"];
	        this.log_level = source["log_level"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ToolStatus {
	    tools_dir: string;
	    python: string;
	    python_version: string;
	    python_ok: boolean;
	    unity_py: boolean;
	    ff_tools: string;
	    ff_tools_ok: boolean;
	    repak: string;
	    repak_ok: boolean;
	    retoc: string;
	    retoc_ok: boolean;
	    gdre_tools: string;
	    gdre_tools_ok: boolean;
	    patcher_exe: string;
	    patcher_ok: boolean;
	    available: Record<string, boolean>;
	
	    static createFrom(source: any = {}) {
	        return new ToolStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tools_dir = source["tools_dir"];
	        this.python = source["python"];
	        this.python_version = source["python_version"];
	        this.python_ok = source["python_ok"];
	        this.unity_py = source["unity_py"];
	        this.ff_tools = source["ff_tools"];
	        this.ff_tools_ok = source["ff_tools_ok"];
	        this.repak = source["repak"];
	        this.repak_ok = source["repak_ok"];
	        this.retoc = source["retoc"];
	        this.retoc_ok = source["retoc_ok"];
	        this.gdre_tools = source["gdre_tools"];
	        this.gdre_tools_ok = source["gdre_tools_ok"];
	        this.patcher_exe = source["patcher_exe"];
	        this.patcher_ok = source["patcher_ok"];
	        this.available = source["available"];
	    }
	}

}

export namespace exchange {
	export class ImportResult {
	    Accepted: number;
	    Skipped: number;
	    Rejected: number;
	    Errors: string[];

	    static createFrom(source: any = {}) {
	        return new ImportResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Accepted = source["Accepted"];
	        this.Skipped = source["Skipped"];
	        this.Rejected = source["Rejected"];
	        this.Errors = source["Errors"];
	    }
	}
}

export namespace fonts {
	export class GlyphCoverageReport {
	    supports_persian: boolean;
	    supports_arabic: boolean;
	    supports_persian_digits: boolean;
	    has_opentype_shaping: boolean;
	    total_tested_runes: number;
	    supported_runes_count: number;
	    missing_runes?: string[];
	    coverage_percent: number;

	    static createFrom(source: any = {}) {
	        return new GlyphCoverageReport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.supports_persian = source["supports_persian"];
	        this.supports_arabic = source["supports_arabic"];
	        this.supports_persian_digits = source["supports_persian_digits"];
	        this.has_opentype_shaping = source["has_opentype_shaping"];
	        this.total_tested_runes = source["total_tested_runes"];
	        this.supported_runes_count = source["supported_runes_count"];
	        this.missing_runes = source["missing_runes"];
	        this.coverage_percent = source["coverage_percent"];
	    }
	}
	export class FontInventoryItem {
	    path: string;
	    container?: string;
	    resource_id?: string;
	    engine: string;
	    detected_format: string;
	    asset_type: string;
	    family?: string;
	    style?: string;
	    atlas_texture?: string;
	    confidence: number;
	    can_export: boolean;
	    can_replace: boolean;
	    unsupported_reason?: string;
	    glyph_coverage?: GlyphCoverageReport;
	    file_size: number;

	    static createFrom(source: any = {}) {
	        return new FontInventoryItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.container = source["container"];
	        this.resource_id = source["resource_id"];
	        this.engine = source["engine"];
	        this.detected_format = source["detected_format"];
	        this.asset_type = source["asset_type"];
	        this.family = source["family"];
	        this.style = source["style"];
	        this.atlas_texture = source["atlas_texture"];
	        this.confidence = source["confidence"];
	        this.can_export = source["can_export"];
	        this.can_replace = source["can_replace"];
	        this.unsupported_reason = source["unsupported_reason"];
	        this.glyph_coverage = source["glyph_coverage"];
	        this.file_size = source["file_size"];
	    }
	}
	export class FontInventoryReport {
	    game_root: string;
	    engine: string;
	    total_fonts_found: number;
	    direct_font_files: number;
	    engine_resources: number;
	    bitmap_atlases: number;
	    persian_ready_count: number;
	    items: FontInventoryItem[];
	    diagnostic_summary: string;
	    recommended_action: string;

	    static createFrom(source: any = {}) {
	        return new FontInventoryReport(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game_root = source["game_root"];
	        this.engine = source["engine"];
	        this.total_fonts_found = source["total_fonts_found"];
	        this.direct_font_files = source["direct_font_files"];
	        this.engine_resources = source["engine_resources"];
	        this.bitmap_atlases = source["bitmap_atlases"];
	        this.persian_ready_count = source["persian_ready_count"];
	        this.items = this.convertValues(source["items"], FontInventoryItem);
	        this.diagnostic_summary = source["diagnostic_summary"];
	        this.recommended_action = source["recommended_action"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) return a;
		    if (a.slice && a.map) return (a as any[]).map(elem => this.convertValues(elem, classs));
		    if ("object" === typeof a) return new classs(a);
		    return a;
		}
	}
	export class FontReplacementResult {
	    target_rel_path: string;
	    staged_path: string;
	    source_sha256: string;
	    target_sha256: string;
	    font_family: string;

	    static createFrom(source: any = {}) {
	        return new FontReplacementResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target_rel_path = source["target_rel_path"];
	        this.staged_path = source["staged_path"];
	        this.source_sha256 = source["source_sha256"];
	        this.target_sha256 = source["target_sha256"];
	        this.font_family = source["font_family"];
	    }
	}
}

export namespace assetbrowser {
	export class AssetNode {
	    id: string;
	    name: string;
	    rel_path: string;
	    container?: string;
	    member_path?: string;
	    type: string;
	    format: string;
	    engine: string;
	    size_bytes: number;
	    text_count: number;
	    parsing_status: string;
	    can_extract: boolean;
	    can_inject: boolean;
	    read_only: boolean;
	    read_only_reason?: string;
	    children?: AssetNode[];
	    metadata?: Record<string, string>;

	    static createFrom(source: any = {}) {
	        return new AssetNode(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.rel_path = source["rel_path"];
	        this.container = source["container"];
	        this.member_path = source["member_path"];
	        this.type = source["type"];
	        this.format = source["format"];
	        this.engine = source["engine"];
	        this.size_bytes = source["size_bytes"];
	        this.text_count = source["text_count"];
	        this.parsing_status = source["parsing_status"];
	        this.can_extract = source["can_extract"];
	        this.can_inject = source["can_inject"];
	        this.read_only = source["read_only"];
	        this.read_only_reason = source["read_only_reason"];
	        this.children = this.convertValues(source["children"], AssetNode);
	        this.metadata = source["metadata"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) return a;
		    if (a.slice && a.map) return (a as any[]).map(elem => this.convertValues(elem, classs));
		    if ("object" === typeof a) return new classs(a);
		    return a;
		}
	}

	export class AssetBrowserView {
	    game_root: string;
	    engine: string;
	    total_files: number;
	    total_containers: number;
	    total_parsed_resources: number;
	    total_text_records: number;
	    root_nodes: AssetNode[];

	    static createFrom(source: any = {}) {
	        return new AssetBrowserView(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.game_root = source["game_root"];
	        this.engine = source["engine"];
	        this.total_files = source["total_files"];
	        this.total_containers = source["total_containers"];
	        this.total_parsed_resources = source["total_parsed_resources"];
	        this.total_text_records = source["total_text_records"];
	        this.root_nodes = this.convertValues(source["root_nodes"], AssetNode);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) return a;
		    if (a.slice && a.map) return (a as any[]).map(elem => this.convertValues(elem, classs));
		    if ("object" === typeof a) return new classs(a);
		    return a;
		}
	}
}

