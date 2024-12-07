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

type Data struct {
    Avg float64   `redis:"avg"`
    NumVotes int64  `redis:"numVotes"`
}

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

    err := r.HSet(ctx, "session:test", "avg", "0", "numVotes", "0").Err(); if err != nil {
        return nil, nil, err
    }

    return ctx, r, nil
}

func formatBar(data Data) string {
    rawHtml := fmt.Sprintf(`
        <div id="progress-bar">
            <div id="progress" style="background-color: hsl(%f, 70%%, 50%%); width: %f%%;">
                %d%%
            </div>
        </div>
        <p>Total votes: %d</p>`,
        data.Avg, data.Avg, int(math.Round(data.Avg)), data.NumVotes)

    return strings.ReplaceAll(rawHtml, "\n", "")
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
        var data Data

        err := r.HGetAll(rctx, "session:test").Scan(&data); if err != nil {
            return echo.NewHTTPError(500, err)
        }

        return c.HTML(200, formatBar(data))
    })

    e.POST("/vote", func(c echo.Context) error {
        vote, err := strconv.ParseFloat(c.FormValue("confidence"), 64); if err != nil {
            return echo.NewHTTPError(400, fmt.Sprintf("Error parsing confidence: %v", err))
        }
        if vote < 0 || vote > 100 {
            return echo.NewHTTPError(400, fmt.Sprintf("Invalid confidence: %f", vote))
        }

        scriptBytes, err := os.ReadFile("vote.lua"); if err != nil {
            echo.NewHTTPError(500, fmt.Sprintf("Failed to read script: %v", err))
        }
        result, err := redis.NewScript(string(scriptBytes)).Run(rctx, r, 
            []string{"session:test"}, 
            c.RealIP(), vote,
        ).StringSlice()
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

        html := formatBar(Data{newAvg, totalVotes})
        err = r.Publish(rctx, "bar", html).Err(); if err != nil {
            return err
        }

        return c.HTML(200, html)
    })

    e.GET("/barstream", func(c echo.Context) error {
        fmt.Printf("Connection from %s\n", c.Request().RemoteAddr)

        c.Response().Header().Set("Content-Type", "text/event-stream")
        c.Response().Header().Set("Cache-Control", "no-cache")
        c.Response().Header().Set("Connection", "keep-alive")

        pubsub := r.Subscribe(rctx, "bar")
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
                fmt.Fprintf(c.Response(), "data: %s\n\n", msg.Payload)
                c.Response().Flush()
            }
        }
    })

    e.Logger.Fatal(e.Start(serverPort))
}

