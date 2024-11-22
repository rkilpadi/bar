package main

import (
    "context"
	"html/template"
	"io"
    "strconv"

    "github.com/labstack/echo/v4"
    "github.com/labstack/echo/v4/middleware"
    "github.com/redis/go-redis/v9"
)

type Templates struct {
    templates *template.Template
}

func (t *Templates) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
    return t.templates.ExecuteTemplate(w, name, data)
}

func newTemplate() *Templates {
    return &Templates {
        templates: template.Must(template.ParseGlob("*.html")),
    }
}

type Data struct {
    Avg float64
    Votes int
}

func getData(ctx context.Context, r *redis.Client) (Data, error) {
    var data Data

	avg, err := r.Get(ctx, "avg").Float64()
	if err != nil {
		return data, err
	}
	votes, err := r.Get(ctx, "votes").Int()
    if err != nil {
		return data, err
	}

    return Data { Avg: avg, Votes: votes }, nil
}

func main() {
    r := redis.NewClient(&redis.Options{
        Addr:	  "localhost:6379",
        Password: "",
        DB:		  0,
    })
    ctx := context.Background()

    err := r.FlushAll(ctx).Err()
    if err != nil {
        panic(err)
    }

    err = r.Set(ctx, "avg", 0, 0).Err()
    if err != nil {
        panic(err)
    }

    err = r.Set(ctx, "votes", 0, 0).Err()
    if err != nil {
        panic(err)
    }

    e := echo.New()
    e.Use(middleware.Logger())
    e.Renderer = newTemplate()

    e.File("/styles.css", "styles.css")

    e.GET("/", func(c echo.Context) error {
        data, err := getData(ctx, r)
        if err != nil {
            panic(err)
        }
        return c.Render(200, "index", data)
    })

    e.POST("/vote", func(c echo.Context) error {
        inc, err := strconv.ParseFloat(c.FormValue("increment"), 64)
        if err != nil {
            return err
        }

        avg, err := r.Get(ctx, "avg").Float64()
        if err != nil {
            return err
        }

        votes, err := r.Get(ctx, "votes").Int()
        if err != nil {
            return err
        }

        ip := c.RealIP()

        voted, err := r.HExists(ctx, "ips", ip).Result()
        if err != nil {
            return err
        }

        if voted {
            vote, err := r.HGet(ctx, "ips", ip).Float64()
            if err != nil {
                return err
            }
            avg = (avg * float64(votes) - float64(vote) + inc) / float64(votes)
        } else {
            err = r.Incr(ctx, "votes").Err()
            if err != nil {
                return err
            }
            avg = (avg * float64(votes) + inc) / float64(votes + 1)
            votes++
        }

        err = r.HSet(ctx, "ips", ip, inc).Err()
        if err != nil {
            return err
        }

        err = r.Set(ctx, "avg", avg, 0).Err()
        if err != nil {
            return err
        }

        return c.Render(200, "bar", Data { Avg: avg, Votes: votes })
    })

    e.Logger.Fatal(e.Start(":3000"))
}

