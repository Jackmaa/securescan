import type { SSEEvent } from './types';

/**
 * SSE (Server-Sent Events) client helper.
 *
 * Why SSE (and why this helper exists):
 * - The backend pushes progress events while a scan runs.
 * - Communication is strictly server → browser, which maps cleanly to SSE.
 * - Browsers provide automatic reconnection for EventSource, so this is simpler than websockets.
 *
 * This module centralizes:
 * - which event names we listen for
 * - JSON parsing behavior (including fallbacks for non-JSON payloads)
 * - lifecycle cleanup (explicit close)
 */

/**
 * connectSSE opens an EventSource to the scan progress stream and dispatches typed events.
 *
 * Contract:
 * - `onEvent` is called for every event frame we subscribe to.
 * - Each event carries a `{ type, data }` shape, where data is parsed JSON when possible.
 *
 * Error handling:
 * - If the server sends a non-JSON payload, we still surface it as `{ raw: string }`
 *   rather than dropping the event.
 */
export function connectSSE(scanId: string, onEvent: (event: SSEEvent) => void) {
	// EventSource uses a simple GET request and keeps the connection open.
	const source = new EventSource(`/api/scans/${scanId}/progress`);

	// These must match the backend event names (`services.SSEEvent.Type`).
	const eventTypes = [
		'status',
		'tool_start',
		'tool_complete',
		'tool_error',
		'mapping',
		'scoring',
		'complete'
	];

	for (const type of eventTypes) {
		source.addEventListener(type, (e: MessageEvent) => {
			try {
				// The backend sends JSON strings in the `data:` field.
				const data = JSON.parse(e.data);
				onEvent({ type, data });
			} catch {
				// Best-effort fallback when data isn't valid JSON.
				onEvent({ type, data: { raw: e.data } });
			}
		});
	}

	source.onerror = () => {
		// EventSource reconnects automatically; if the scan is done
		// the server closes the stream and we clean up here
		if (source.readyState === EventSource.CLOSED) {
			source.close();
		}
	};

	return {
		// We expose close so routes can clean up on navigation/unmount.
		close: () => source.close()
	};
}
