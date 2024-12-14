package main

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

const redisAddr = "localhost:6379"
const serverPort = ":3000"

func initializeRedis() (context.Context, *redis.Client, error) {
    ctx := context.Background()
    r := redis.NewClient(&redis.Options{
        Addr:	  redisAddr,
        Password: "",
        DB:       0,
    })

    if err := r.FlushAll(ctx).Err(); err != nil {
        return nil, nil, err
    }

    err := r.MSet(ctx, "confidence", 0, "numVotes", 0).Err(); if err != nil {
        return nil, nil, err
    }

    return ctx, r, nil
}

func formatBar(confidence float64, numVotes int) string {
    rawHtml := fmt.Sprintf(`
        <div id="progress-bar">
            <div id="progress" style="background-color: hsl(%f, 70%%, 50%%); width: %f%%;">
                %d%%
            </div>
        </div>
        <p>Total votes: %d</p>`,
        confidence, confidence, int(math.Round(confidence)), numVotes)

    return strings.ReplaceAll(rawHtml, "\n", "")
}

func formatQuestions(questions []string, page int64) string {
    var htmlBuilder strings.Builder
    var triggerAttributes string

    for i, question := range questions {
        if i == len(questions) - 1 {
            triggerAttributes = fmt.Sprintf("hx-get=\"/question?page=%d\" hx-trigger=\"revealed\" hx-swap=\"afterend\"", page + 1)
        }
        htmlBuilder.WriteString(fmt.Sprintf("<p %s>%s</p>", triggerAttributes, question))
    }

    return htmlBuilder.String()
}

func main() {
    e := echo.New()
    e.HTTPErrorHandler = func(err error, c echo.Context) {
        code := 500
        if he, ok := err.(*echo.HTTPError); ok {
            code = he.Code
        }
        c.Logger().Error(err)
        if code >= 500 {
            c.JSON(code, "Something went wrong")
        } else {
            c.JSON(code, err)
        }
    }

    rctx, r, err := initializeRedis(); if err != nil {
        e.Logger.Fatal(err)
    }

    e.File("/styles.css", "styles.css")

    e.File("/", "index.html")

    e.GET("/bar", func(c echo.Context) error {
        confidence, err := r.Get(rctx, "confidence").Float64(); if err != nil {
            return echo.NewHTTPError(500, err)
        }

        numVotes, err := r.Get(rctx, "numVotes").Int(); if err != nil {
            return echo.NewHTTPError(500, err)
        }

        return c.HTML(200, formatBar(confidence, numVotes))
    })

    e.POST("/vote", func(c echo.Context) error {
        vote, err := strconv.ParseFloat(c.FormValue("confidence"), 64); if err != nil {
            return echo.NewHTTPError(400, fmt.Sprintf("Error parsing confidence: %v", err))
        }
        if vote < 0 || vote > 100 {
            return echo.NewHTTPError(400, fmt.Sprintf("Confidence expected to be between 0-100, found: %f", vote))
        }

        scriptBytes, err := os.ReadFile("vote.lua"); if err != nil {
            echo.NewHTTPError(500, fmt.Sprintf("Failed to read script: %v", err))
        }
        result, err := redis.NewScript(string(scriptBytes)).Run(rctx, r, []string{}, c.RealIP(), vote).StringSlice()
        if err != nil {
            return echo.NewHTTPError(500, fmt.Sprintf("Error running script: %v", err))
        }
        if len(result) != 2 {
            return echo.NewHTTPError(500, "Expected script to return result of length 2")
        }

        newAvg, err := strconv.ParseFloat(result[0], 64); if err != nil {
            return echo.NewHTTPError(500, err)
        }
        totalVotes, err := strconv.ParseInt(result[1], 10, 64); if err != nil {
            return echo.NewHTTPError(500, err)
        }

        html := formatBar(newAvg, int(totalVotes))
        payload := fmt.Sprintf("event: vote\ndata: %s\n\n", html)
        err = r.Publish(rctx, "sse", payload).Err(); if err != nil {
            return echo.NewHTTPError(500, err)
        }

        return c.HTML(200, html)
    })

    e.GET("/question", func(c echo.Context) error {
        const pageLength = int64(30)

        page, err := strconv.ParseInt(c.QueryParam("page"), 10, 64)
        if err != nil {
            return echo.NewHTTPError(400, fmt.Sprintf("Error parsing page: %v", err))
        }
        if page < 1 {
            return echo.NewHTTPError(400, fmt.Sprintf("Page expected to be a positive integer, found: %d", page))
        }

        questions, err := r.ZRevRange(rctx, "questions", pageLength * (page - 1), page * pageLength - 1).Result()
        if err != nil {
            return echo.NewHTTPError(500, err)
        }

        return c.HTML(200, formatQuestions(questions, page))
    })

    e.POST("/question", func(c echo.Context) error {
        timestamp := time.Now().Local().Format("15:04:05")
        questionText := fmt.Sprintf("%s: %s", timestamp, c.FormValue("question"))
        
        err := r.ZAdd(rctx, "questions", 
            redis.Z{Member: questionText, Score: float64(time.Now().Unix())},
        ).Err(); if err != nil {
            return echo.NewHTTPError(500, err)
        }

        payload := fmt.Sprintf("event: question\ndata: <p>%s</p>\n\n", questionText)
        err = r.Publish(rctx, "sse", payload).Err(); if err != nil {
            return echo.NewHTTPError(500, err)
        }
        return c.NoContent(202)
    })

    e.GET("/sse", func(c echo.Context) error {
        fmt.Printf("Connection from %s\n", c.Request().RemoteAddr)

        c.Response().Header().Set("Content-Type", "text/event-stream")
        c.Response().Header().Set("Cache-Control", "no-cache")
        c.Response().Header().Set("Connection", "keep-alive")

        pubsub := r.Subscribe(rctx, "sse")
        defer pubsub.Close()

        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-c.Request().Context().Done():
                fmt.Printf("Disconnected %s\n", c.Request().RemoteAddr)
                return nil
            case <-ticker.C:
                msg, err := pubsub.ReceiveMessage(rctx); if err != nil {
                    return err
                }
                fmt.Fprint(c.Response(), msg.Payload)
                c.Response().Flush()
            }
        }
    })

    e.Logger.Fatal(e.Start(serverPort))
}

