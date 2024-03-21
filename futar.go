package main

import (
	"github.com/labstack/echo/v4"
	"log/slog"
	"net/http"
)

type FutarServer struct {
	version     string
	environment string
	serviceName string
	ready       bool
}

func (d *FutarServer) markReady() {
	slog.Info("Application is ready")
	d.ready = true
}

func (d *FutarServer) HelloWorld(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}

func (d *FutarServer) MetaHealth(ctx echo.Context) error {
	message := "Version: " + d.version
	name := "futar"
	status := Ok

	checks := []ServiceHealthCheck{
		{
			Message: &message,
			Name:    &name,
			Status:  &status,
		},
	}
	return ctx.JSONPretty(200, ServiceHealth{
		Checks:          &checks,
		EnvironmentName: &d.environment,
		ServiceName:     &d.serviceName,
	}, "  ")
}

func (d *FutarServer) MetaReady(ctx echo.Context) error {
	if d.ready {
		return ctx.String(200, "ready\n")
	}

	return ctx.String(503, "not ready\n")
}
