/**
 * Shared API types used by the SvelteKit frontend.
 *
 * Why these types exist:
 * - The API speaks snake_case JSON (matching the Go structs / DB naming).
 * - Having a single source of truth for shapes reduces "undefined" bugs in UI code.
 * - Keeping them in `src/lib` makes them importable from both components and routes.
 *
 * Note on naming:
 * - These interfaces intentionally use snake_case to match the wire format exactly,
 *   so we don't need mapping layers (and we avoid subtle mismatches).
 */

/** Project is the root resource a scan runs against (git/zip source staged locally). */
export interface Project {
	id: string;
	name: string;
	source_type: 'git' | 'zip';
	source_url?: string;
	local_path: string;
	languages: string[];
	frameworks: string[];
	created_at: string;
}

/**
 * Scan represents one execution of the scanning pipeline for a project.
 *
 * Optional fields (score/grade/error_msg/timestamps) are omitted until the scan reaches
 * stages where those values exist. This mirrors the API and keeps UI conditionals simple.
 */
export interface Scan {
	id: string;
	project_id: string;
	status: ScanStatus;
	score?: number;
	grade?: string;
	tool_count: number;
	tools_done: number;
	error_msg?: string;
	started_at?: string;
	completed_at?: string;
	created_at: string;
}

/**
 * ScanStatus is the lifecycle state machine exposed by the backend.
 *
 * The frontend uses this both for:
 * - rendering user-friendly progress states
 * - deciding navigation (e.g., when completed, go to dashboard)
 */
export type ScanStatus =
	| 'pending'
	| 'cloning'
	| 'detecting'
	| 'scanning'
	| 'mapping'
	| 'scoring'
	| 'completed'
	| 'failed';

/**
 * Finding is a normalized security issue produced by any scanner.
 *
 * Many fields are optional because different tools report different levels of detail
 * (some provide file/line/snippet, some only provide package-level vulnerability info).
 */
export interface Finding {
	id: string;
	scan_id: string;
	tool_name: string;
	rule_id?: string;
	file_path?: string;
	line_start?: number;
	line_end?: number;
	col_start?: number;
	col_end?: number;
	message: string;
	severity: Severity;
	owasp_category?: string;
	owasp_label?: string;
	cwe_id?: string;
	raw_output?: unknown;
	code_snippet?: string;
	created_at: string;
}

/** Severity is normalized across tools so the UI can sort and color consistently. */
export type Severity = 'critical' | 'high' | 'medium' | 'low' | 'info';

/**
 * Fix represents a remediation suggestion associated with a finding.
 *
 * Some fixes include code snippets (original/fixed) while others may be descriptive only.
 * Status supports a review workflow (pending → accepted/rejected, and eventually applied).
 */
export interface Fix {
	id: string;
	finding_id: string;
	scan_id: string;
	fix_type: 'template' | 'ai';
	description: string;
	explanation?: string;
	original_code?: string;
	fixed_code?: string;
	file_path: string;
	line_start?: number;
	line_end?: number;
	status: 'pending' | 'accepted' | 'rejected' | 'applied';
	created_at: string;
}

/**
 * ScanStats is a compact dashboard payload derived from a scan's findings.
 *
 * Keeping stats separate from full finding lists allows the UI to render the overview
 * quickly without loading potentially large datasets.
 */
export interface ScanStats {
	total_findings: number;
	by_severity: Record<string, number>;
	by_owasp: Record<string, number>;
	by_tool: Record<string, number>;
	score?: number;
	grade?: string;
}

/** FindingResult is the server-side paginated finding list response. */
export interface FindingResult {
	findings: Finding[];
	total: number;
	page: number;
	limit: number;
}

/**
 * SSEEvent is the frontend representation of an SSE message.
 *
 * `data` is intentionally untyped because each event type carries a different payload.
 * Route code should narrow/validate based on `type`.
 */
export interface SSEEvent {
	type: string;
	data: Record<string, unknown>;
}
