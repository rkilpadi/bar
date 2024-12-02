package main

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"
)

const redisAddr = "localhost:6379"
const serverPort = ":3000"

type Data struct {
    Avg float64 
    NumVotes int
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

    if err := r.Set(ctx, "avg", 0, 0).Err(); err != nil {
        return nil, nil, err
    }

    if err := r.Set(ctx, "numVotes", 0, 0).Err(); err != nil {
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

    ctx, r, err := initializeRedis(); if err != nil {
        e.Logger.Fatal(err)
    }

    e.File("/styles.css", "styles.css")

    e.File("/", "index.html")

    e.GET("/bar", func(c echo.Context) error {
        avg, err := r.Get(ctx, "avg").Float64(); if err != nil {
            return err
        }

        numVotes, err := r.Get(ctx, "numVotes").Int(); if err != nil {
            return err
        }

        data := Data{Avg: avg, NumVotes: numVotes}
        barHTML := formatBar(data)

        return c.HTML(200, barHTML)
    })

    e.POST("/vote", func(c echo.Context) error {
        vote, err := strconv.ParseFloat(c.FormValue("confidence"), 64); if err != nil {
            return err
        }
        if vote < 0 || vote > 100 {
            return echo.NewHTTPError(400, fmt.Sprintf("Invalid confidence: %f", vote))
        }

        avg, err := r.Get(ctx, "avg").Float64(); if err != nil {
            return err
        }

        numVotes, err := r.Get(ctx, "numVotes").Int(); if err != nil {
            return err
        }

        ip := c.RealIP()
        voted, err := r.HExists(ctx, "ips", ip).Result(); if err != nil {
            return err
        }

        if voted {
            oldVote, err := r.HGet(ctx, "ips", ip).Float64(); if err != nil {
                return err
            }
            avg = (avg * float64(numVotes) - oldVote + vote) / float64(numVotes)
        } else {
            err = r.Incr(ctx, "numVotes").Err(); if err != nil {
                return err
            }
            avg = (avg * float64(numVotes) + vote) / float64(numVotes + 1)
            numVotes++
        }

        err = r.HSet(ctx, "ips", ip, vote).Err(); if err != nil {
            return err
        }

        err = r.Set(ctx, "avg", avg, 0).Err(); if err != nil {
            return err
        }

        html := formatBar(Data{Avg: avg, NumVotes: numVotes})
        err = r.Publish(ctx, "bar", html).Err(); if err != nil {
            return err
        }

        return c.HTML(200, html)
    })

    e.GET("/barstream", func(c echo.Context) error {
        fmt.Printf("Connection from %s\n", c.Request().RemoteAddr)

        c.Response().Header().Set("Content-Type", "text/event-stream")
        c.Response().Header().Set("Cache-Control", "no-cache")
        c.Response().Header().Set("Connection", "keep-alive")

        pubsub := r.Subscribe(ctx, "bar")
        defer pubsub.Close()

        ticker := time.NewTicker(time.Second)
        defer ticker.Stop()
        for {
            select {
            case <-c.Request().Context().Done():
                fmt.Printf("Disconnected %s\n", c.Request().RemoteAddr)
                return nil
            case <-ticker.C:
                msg, err := pubsub.ReceiveMessage(ctx); if err != nil {
                    return err
                }
                fmt.Fprintf(c.Response(), "data: %s\n\n", msg.Payload)
                c.Response().Flush()
            }
        }
    })

    e.Logger.Fatal(e.Start(serverPort))
}

