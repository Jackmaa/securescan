package handlers

import (
	"bufio" // Fiber streaming writes use a buffered writer for efficiency.
	"fmt"   // SSE frames are written as formatted text lines.

	"securescan/services" // ScanService provides subscribe/unsubscribe + event broadcasting.

	"github.com/gofiber/fiber/v3" // HTTP framework providing streaming response primitives.
	"github.com/google/uuid"      // Scan IDs are UUIDs; parsed from URL params.
)

// SSEHandler serves Server-Sent Events endpoints.
//
// SSE is a good fit here because clients only need server→client updates (progress
// notifications). It’s simpler than WebSockets (no bidirectional state) and works
// well with HTTP infrastructure (proxies, auth, etc.).
type SSEHandler struct {
	ScanService *services.ScanService
}

// NewSSEHandler constructs an SSE handler with its dependency injected.
func NewSSEHandler(ss *services.ScanService) *SSEHandler {
	return &SSEHandler{ScanService: ss}
}

// Progress streams scan progress events via Server-Sent Events.
// SSE is chosen over WebSocket because communication is strictly server→client.
func (h *SSEHandler) Progress(c fiber.Ctx) error {
	scanID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid scan ID"})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	// Subscribe returns a per-connection channel. The ScanService fan-outs events to
	// all subscribers of a scan so multiple tabs can follow the same scan.
	ch := h.ScanService.Subscribe(scanID)

	return c.SendStreamWriter(func(w *bufio.Writer) {
		for event := range ch {
			// SSE is a line-oriented protocol:
			// - "event:" names the event type
			// - "data:" carries payload (JSON string in our case)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, event.Data)
			w.Flush()
		}
		// Unsubscribe after the client disconnects or the channel closes.
		// Doing this explicitly avoids leaking channels in the subscribers map.
		h.ScanService.Unsubscribe(scanID, ch)
	})
}
