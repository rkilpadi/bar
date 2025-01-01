package main

import (
	"context"

	"github.com/labstack/echo/v4"
	"kilpadi.com/presentation-bar/db"
	"kilpadi.com/presentation-bar/handlers"
)

const serverPort string = ":3000"

func main() {
    e := echo.New()
    renderer := handlers.NewRenderer()
    e.Renderer = renderer
    e.HTTPErrorHandler = handlers.CustomHTTPErrorHandler

    r, err := db.Initialize(); if err != nil {
        e.Logger.Fatal(err)
    }
    
    h := handlers.NewHandler(r, context.Background(), renderer)

    e.File("/styles.css", "styles.css")

    e.File("/", "views/index.html")

    e.GET("/bar", h.GetBar)
    e.POST("/vote", h.Vote)

    e.GET("/question", h.GetQuestions)
    e.POST("/question", h.AskQuestion)

    e.GET("/sse", h.SseHandler)

    e.Logger.Fatal(e.Start(serverPort))
}

