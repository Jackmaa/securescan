import type { Project, Scan, ScanStats, FindingResult, Fix } from './types';

const BASE = '/api';

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

export function createProject(name: string, sourceType: string, sourceUrl?: string) {
	return request<Project>('/projects', {
		method: 'POST',
		body: JSON.stringify({ name, source_type: sourceType, source_url: sourceUrl })
	});
}

export function getProject(id: string) {
	return request<Project>(`/projects/${id}`);
}

export function triggerScan(projectId: string) {
	return request<Scan>(`/projects/${projectId}/scan`, { method: 'POST' });
}

export function getScan(id: string) {
	return request<Scan>(`/scans/${id}`);
}

export function getScanStats(scanId: string) {
	return request<ScanStats>(`/scans/${scanId}/stats`);
}

export function getFindings(scanId: string, params?: Record<string, string>) {
	const query = params ? '?' + new URLSearchParams(params).toString() : '';
	return request<FindingResult>(`/scans/${scanId}/findings${query}`);
}

export function getFixes(scanId: string) {
	return request<Fix[]>(`/scans/${scanId}/fixes`);
}

export function acceptFix(fixId: string) {
	return request<{ status: string }>(`/fixes/${fixId}/accept`, { method: 'POST' });
}

export function rejectFix(fixId: string) {
	return request<{ status: string }>(`/fixes/${fixId}/reject`, { method: 'POST' });
}

export function bulkFixAction(fixIds: string[], action: 'accept' | 'reject') {
	return request<{ updated: number }>('/fixes/bulk', {
		method: 'POST',
		body: JSON.stringify({ fix_ids: fixIds, action })
	});
}
