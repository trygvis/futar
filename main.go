package main

import (
	"github.com/labstack/gommon/log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

var startupDelay int64 = 1000

type DemoServer struct {
	ready       bool
	environment string
	serviceName string
}

func (d *DemoServer) markReady() {
	log.Info("Application is ready")
	d.ready = true
}

func (d *DemoServer) HelloWorld(c echo.Context) error {
	return c.String(http.StatusOK, "Hello, World!")
}

func (d *DemoServer) MetaHealth(ctx echo.Context) error {
	message := "OK!"
	name := "demo"
	status := Ok

	checks := []ServiceHealthCheck{
		{
			Message: &message,
			Name:    &name,
			Status:  &status,
		},
	}
	return ctx.JSON(200, ServiceHealth{
		Checks:          &checks,
		EnvironmentName: &d.environment,
		ServiceName:     &d.serviceName,
	})
}

func (d *DemoServer) MetaReady(ctx echo.Context) error {
	if d.ready {
		return ctx.String(200, "ready\n")
	}

	return ctx.String(503, "not ready\n")
}

func main() {
	var err error
	port, _ := os.LookupEnv("PORT")
	if port == "" {
		port = "8080"
	}

	startupDelayS, _ := os.LookupEnv("STARTUP_DELAY")
	if startupDelayS != "" {
		startupDelay, err = strconv.ParseInt(startupDelayS, 10, 64)
		if err != nil {
			return
		}
	}

	e := echo.New()
	server := DemoServer{}
	RegisterHandlers(e, &server)

	log.Info("Application is starting")
	time.AfterFunc(time.Duration(startupDelay)*time.Millisecond, func() {
		server.markReady()
	})

	e.Logger.Fatal(e.Start(":" + port))
}
