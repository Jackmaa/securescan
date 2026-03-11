package handlers

import (
	"bufio"
	"fmt"

	"securescan/services"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type SSEHandler struct {
	ScanService *services.ScanService
}

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

	ch := h.ScanService.Subscribe(scanID)

	return c.SendStreamWriter(func(w *bufio.Writer) {
		for event := range ch {
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, event.Data)
			w.Flush()
		}
		h.ScanService.Unsubscribe(scanID, ch)
	})
}
