package handlers

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

func barData(confidence float64, voteCount int) map[string]interface{} {
    return map[string]interface{}{
        "Confidence": confidence, 
        "DisplayConfidence": int(confidence),
        "VoteCount": int(voteCount),
    }
}

func (h *Handler) GetBar(c echo.Context) error {
    confidence, err := h.Redis.Get(h.RequestContext, "confidence").Float64(); if err != nil {
        return echo.NewHTTPError(500, err)
    }

    voteCount, err := h.Redis.Get(h.RequestContext, "voteCount").Int(); if err != nil {
        return echo.NewHTTPError(500, err)
    }

    return c.Render(200, "bar.html", barData(confidence, voteCount))
}

func (h *Handler) Vote(c echo.Context) error {
    vote, err := strconv.ParseFloat(c.FormValue("confidence"), 64); if err != nil {
        return echo.NewHTTPError(400, fmt.Sprintf("Error parsing confidence: %v", err))
    }
    if vote < 0 || vote > 100 {
        return echo.NewHTTPError(400, fmt.Sprintf("Confidence expected to be between 0-100, found: %f", vote))
    }

    scriptBytes, err := os.ReadFile("db/vote.lua"); if err != nil {
        echo.NewHTTPError(500, fmt.Sprintf("Failed to read script: %v", err))
    }
    result, err := redis.NewScript(string(scriptBytes)).Run(h.RequestContext, h.Redis, []string{}, c.RealIP(), vote).StringSlice()
    if err != nil {
        return echo.NewHTTPError(500, fmt.Sprintf("Error running script: %v", err))
    }
    if len(result) != 2 {
        return echo.NewHTTPError(500, "Expected script to return result of length 2")
    }

    confidence, err := strconv.ParseFloat(result[0], 64); if err != nil {
        return echo.NewHTTPError(500, err)
    }
    voteCount, err := strconv.ParseInt(result[1], 10, 64); if err != nil {
        return echo.NewHTTPError(500, err)
    }

    var html strings.Builder
    err = h.Renderer.Render(&html, "bar.html", barData(confidence, int(voteCount)), c); if err != nil {
        return echo.NewHTTPError(500, fmt.Sprintf("Error rendering template: %v", err))
    }

    payload := fmt.Sprintf("event: vote\ndata: %s\n\n", strings.ReplaceAll(html.String(), "\n", ""))
    err = h.Redis.Publish(h.RequestContext, "sse", payload).Err(); if err != nil {
        return echo.NewHTTPError(500, err)
    }

    return c.HTML(200, html.String())
}

