package handlers

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
	"web-server/redis"
)

type StreamHandler struct {
	rdb *redis.Client
}

func NewStreamHandler(rdb *redis.Client) *StreamHandler {
	return &StreamHandler{rdb: rdb}
}

// GET /matches/:match_id/stream
// Server-Sent Events endpoint. Subscribes to Redis Pub/Sub and pushes
// the updated MatchState to all connected clients in real time.
func (h *StreamHandler) Stream(c *fiber.Ctx) error {
	matchID := c.Params("match_id")

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Access-Control-Allow-Origin", "*")
	c.Set("X-Accel-Buffering", "no") // Disable Nginx buffering

	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		pubsub := h.rdb.Subscribe(ctx, matchID)
		defer pubsub.Close()

		ch := pubsub.Channel()

		// Send current state immediately so client has initial data.
		if state, err := h.rdb.GetMatchState(ctx, matchID); err == nil && state != nil {
			data, _ := json.Marshal(state)
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				return
			}
			_ = w.Flush()
		}

		heartbeat := time.NewTicker(30 * time.Second)
		defer heartbeat.Stop()

		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				if _, err := fmt.Fprintf(w, "data: %s\n\n", msg.Payload); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					log.Printf("stream: client disconnected from match %s", matchID)
					return
				}

			case <-heartbeat.C:
				if _, err := fmt.Fprintf(w, ": heartbeat\n\n"); err != nil {
					return
				}
				if err := w.Flush(); err != nil {
					return
				}
			}
		}
	}))

	return nil
}
