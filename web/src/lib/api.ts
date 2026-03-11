import type { Project, Scan, ScanStats, FindingResult, Fix } from './types';

/**
 * Minimal API client for the SvelteKit frontend.
 *
 * Why this wrapper exists (instead of calling `fetch` everywhere):
 * - Centralizes base path and JSON headers.
 * - Provides consistent error handling (API returns `{ error }`).
 * - Keeps route components focused on UI state, not HTTP details.
 *
 * Design choice:
 * - We keep the client "thin" and type-driven: each exported function mirrors one backend endpoint.
 * - The types come from `src/lib/types.ts` and match the API's snake_case payloads.
 */

const BASE = '/api';

/**
 * request performs a JSON HTTP request and returns a typed JSON response.
 *
 * Why it throws:
 * - Call sites can `try/catch` and surface a single string message in the UI.
 * - It avoids forcing every page/component to duplicate status checks and parsing.
 */
async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(`${BASE}${path}`, {
		headers: { 'Content-Type': 'application/json' },
		...init
	});

	if (!res.ok) {
		const body = await res.json().catch(() => ({}));
		throw new Error(body.error || `Request failed: ${res.status}`);
	}

	return res.json();
}

/** createProject calls POST /projects to register a new scan target. */
export function createProject(name: string, sourceType: string, sourceUrl?: string) {
	return request<Project>('/projects', {
		method: 'POST',
		body: JSON.stringify({ name, source_type: sourceType, source_url: sourceUrl })
	});
}

/** getProject fetches a project record by id. */
export function getProject(id: string) {
	return request<Project>(`/projects/${id}`);
}

/** triggerScan starts a scan for the given project and returns the created Scan record. */
export function triggerScan(projectId: string) {
	return request<Scan>(`/projects/${projectId}/scan`, { method: 'POST' });
}

/** getScan fetches the current scan state (status, progress counters, score/grade when ready). */
export function getScan(id: string) {
	return request<Scan>(`/scans/${id}`);
}

/** getScanStats fetches aggregated dashboard metrics derived from findings. */
export function getScanStats(scanId: string) {
	return request<ScanStats>(`/scans/${scanId}/stats`);
}

/**
 * getFindings fetches the paginated findings list.
 *
 * `params` is used for server-side filtering/sorting (severity/owasp/tool/page/limit/etc.).
 */
export function getFindings(scanId: string, params?: Record<string, string>) {
	const query = params ? '?' + new URLSearchParams(params).toString() : '';
	return request<FindingResult>(`/scans/${scanId}/findings${query}`);
}

/** getFixes fetches all fix suggestions for a scan. */
export function getFixes(scanId: string) {
	return request<Fix[]>(`/scans/${scanId}/fixes`);
}

/** acceptFix marks a fix as accepted (review workflow). */
export function acceptFix(fixId: string) {
	return request<{ status: string }>(`/fixes/${fixId}/accept`, { method: 'POST' });
}

/** rejectFix marks a fix as rejected (review workflow). */
export function rejectFix(fixId: string) {
	return request<{ status: string }>(`/fixes/${fixId}/reject`, { method: 'POST' });
}

/**
 * bulkFixAction applies accept/reject to multiple fixes in one round-trip.
 *
 * This exists for UX: the UI can support “accept all low risk fixes” without N requests.
 */
export function bulkFixAction(fixIds: string[], action: 'accept' | 'reject') {
	return request<{ updated: number }>('/fixes/bulk', {
		method: 'POST',
		body: JSON.stringify({ fix_ids: fixIds, action })
	});
}
