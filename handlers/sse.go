package handlers

import (
    "fmt"
    "time"
    
    "github.com/labstack/echo/v4"
)

func (h *Handler) SseHandler(c echo.Context) error {
    fmt.Printf("Connection from %s\n", c.Request().RemoteAddr)

    c.Response().Header().Set("Content-Type", "text/event-stream")
    c.Response().Header().Set("Cache-Control", "no-cache")
    c.Response().Header().Set("Connection", "keep-alive")

    pubsub := h.Redis.Subscribe(h.RequestContext, "sse")
    defer pubsub.Close()

    ticker := time.NewTicker(time.Second)
    defer ticker.Stop()
    for {
        select {
        case <-c.Request().Context().Done():
            fmt.Printf("Disconnected %s\n", c.Request().RemoteAddr)
            return nil
        case <-ticker.C:
            msg, err := pubsub.ReceiveMessage(h.RequestContext); if err != nil {
                return err
            }
            fmt.Fprint(c.Response(), msg.Payload)
            c.Response().Flush()
        }
    }
}

