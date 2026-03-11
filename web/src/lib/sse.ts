import type { SSEEvent } from './types';

// Creates an EventSource connection and returns an object with
// a typed event handler and a close method. Reconnects automatically
// on error (browser default) unless explicitly closed.
export function connectSSE(scanId: string, onEvent: (event: SSEEvent) => void) {
	const source = new EventSource(`/api/scans/${scanId}/progress`);

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
				const data = JSON.parse(e.data);
				onEvent({ type, data });
			} catch {
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
		close: () => source.close()
	};
}
