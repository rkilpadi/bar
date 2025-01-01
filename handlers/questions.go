package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

func questionsData(questions []string, page int64) map[string]interface{} {
    return map[string]interface{}{
        "Questions": questions,
        "LastIndex": len(questions) - 1,
        "NextPage": page + 1,
    }
}

func (h *Handler) GetQuestions(c echo.Context) error {
    const pageLength = int64(20)

    page, err := strconv.ParseInt(c.QueryParam("page"), 10, 64); if err != nil {
        return echo.NewHTTPError(400, fmt.Sprintf("Error parsing page: %v", err))
    }
    if page < 1 {
        return echo.NewHTTPError(400, fmt.Sprintf("Page expected to be a positive integer, found: %d", page))
    }

    questions, err := h.Redis.ZRevRange(h.RequestContext, "questions", pageLength * (page - 1), page * pageLength - 1).Result()
    if err != nil {
        return echo.NewHTTPError(500, err)
    }

    return c.Render(200, "questions.html", questionsData(questions, page))
}

func (h *Handler) AskQuestion(c echo.Context) error {
    question := c.FormValue("question")
    if len(question) == 0 {
        return echo.NewHTTPError(400, "Question input not received")
    }

    timestamp := time.Now().Local().Format("15:04:05")
    questionText := fmt.Sprintf("%s: %s", timestamp, question)

    err := h.Redis.ZAdd(h.RequestContext, "questions", 
        redis.Z{Member: questionText, Score: float64(time.Now().Unix())},
    ).Err(); if err != nil {
        return echo.NewHTTPError(500, err)
    }

    payload := fmt.Sprintf("event: question\ndata: <p>%s</p>\n\n", questionText)
    err = h.Redis.Publish(h.RequestContext, "sse", payload).Err(); if err != nil {
        return echo.NewHTTPError(500, err)
    }
    return c.NoContent(202)
}

