package handlers

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type Handler struct {
    Redis *redis.Client
    RequestContext context.Context
    Renderer *Template
}

func NewHandler(r *redis.Client, ctx context.Context, renderer *Template) *Handler {
    return &Handler{
        Redis: r,
        RequestContext: ctx,
        Renderer: renderer,
    }
}

