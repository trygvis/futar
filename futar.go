package main

import (
	"github.com/labstack/echo/v4"
	"log"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strings"
)

func getClientInfo(ctx echo.Context) string {
	ip := ctx.Request().RemoteAddr
	var b strings.Builder
	b.WriteString("Request from ip: ")
	b.WriteString(ip)
	b.WriteString("\nHeaders: \n")
	for key, value := range ctx.Request().Header {
		b.WriteString(" - ")
		b.WriteString(key)
		b.WriteString(": ")
		b.WriteString(value[0])
		b.WriteString("\n")
	}
	return b.String()
}

type FutarServer struct {
	version          string
	environment      string
	serviceName      string
	healthzErrorRate int64
	ready            bool
}

func (d *FutarServer) markReady() {
	slog.Info("Application is ready")
	d.ready = true
}

func (d *FutarServer) HelloWorld(c echo.Context) error {
	ci := getClientInfo(c)
	println(ci)
	return c.String(http.StatusOK, ci)
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

func (d *FutarServer) MetaHealthz(ctx echo.Context) error {
	ci := getClientInfo(ctx)

	success := rand.Int64N(100) + 1
	status := http.StatusOK
	if success <= d.healthzErrorRate {
		status = http.StatusInternalServerError
	}

	log.Printf("status: %d, %s", status, ci)

	return ctx.String(status, ci)
}

func (d *FutarServer) MetaReady(ctx echo.Context) error {
	if d.ready {
		return ctx.String(200, "ready\n")
	}

	return ctx.String(503, "not ready\n")
}
