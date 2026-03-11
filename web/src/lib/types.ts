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

export type ScanStatus =
	| 'pending'
	| 'cloning'
	| 'detecting'
	| 'scanning'
	| 'mapping'
	| 'scoring'
	| 'completed'
	| 'failed';

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

export type Severity = 'critical' | 'high' | 'medium' | 'low' | 'info';

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

export interface ScanStats {
	total_findings: number;
	by_severity: Record<string, number>;
	by_owasp: Record<string, number>;
	by_tool: Record<string, number>;
	score?: number;
	grade?: string;
}

export interface FindingResult {
	findings: Finding[];
	total: number;
	page: number;
	limit: number;
}

export interface SSEEvent {
	type: string;
	data: Record<string, unknown>;
}
