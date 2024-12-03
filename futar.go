package main

import (
	"fmt"
	"github.com/labstack/echo/v4"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

type FutarServer struct {
	version          string
	environment      string
	instanceId       string
	healthzErrorRate int64
	startTime        time.Time
	ready            bool
	readyCond        *sync.Cond
	readyMutex       *sync.Mutex
}

func (d *FutarServer) getClientInfo(ctx echo.Context) string {
	uptime := fmt.Sprintf("Uptime: %s", d.uptime())
	if !d.ready {
		uptime = ""
	}
	requestIp := fmt.Sprintf("Request from ip: %s", ctx.Request().RemoteAddr)

	var headers []string
	for key, value := range ctx.Request().Header {
		headers = append(headers, fmt.Sprintf(" - %s: %s", key, value[0]))
	}
	slices.Sort(headers)
	return fmt.Sprintf("%s\n%s\nHeaders: \n%s", uptime, requestIp, strings.Join(headers[:], "\n"))
}

func (d *FutarServer) uptime() string {
	return time.Since(d.startTime).String()
}

func (d *FutarServer) markReady() {
	slog.Info("Application is ready", "instanceId", d.instanceId)

	d.startTime = time.Now()

	d.readyMutex.Lock()
	defer d.readyMutex.Unlock()
	d.ready = true
	d.readyCond.Broadcast()
}

func (d *FutarServer) HelloWorld(c echo.Context) error {
	ci := d.getClientInfo(c)
	slog.Info(ci)
	return c.String(http.StatusOK, ci+"\n")
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
		ServiceName:     &d.instanceId,
	}, "  ")
}

func (d *FutarServer) MetaHealthz(ctx echo.Context) error {
	success := rand.Int64N(100) + 1
	status := http.StatusOK
	statusMessage := "OK"
	if success <= d.healthzErrorRate {
		statusMessage = "random error"
		status = http.StatusInternalServerError
	}
	if !d.ready {
		statusMessage = "Not ready"
		status = http.StatusServiceUnavailable
	}

	ci := d.getClientInfo(ctx)
	log.Printf("status: %d %s, %s", status, statusMessage, ci)

	return ctx.String(status, ci)
}

func (d *FutarServer) MetaAppServiceWarmup(ctx echo.Context) error {
	success := rand.Int64N(100) + 1
	status := http.StatusOK
	statusMessage := "OK"
	if success <= d.healthzErrorRate {
		statusMessage = "random error"
		status = http.StatusInternalServerError
	}

	if !d.ready {
		d.readyMutex.Lock()
		defer d.readyMutex.Unlock()
		for !d.ready {
			ci := d.getClientInfo(ctx)
			statusMessage = "waiting (sync)"
			slog.Info("status: %s, %s", statusMessage, ci)
			d.readyCond.Wait()
			statusMessage = "ready (sync)"
		}
	}

	ci := d.getClientInfo(ctx)
	slog.Info("status: %d %s, %s", status, statusMessage, ci)

	return ctx.String(status, ci+"\n")
}

func (d *FutarServer) MetaReady(ctx echo.Context) error {
	if d.ready {
		return ctx.String(200, "ready\n")
	}

	return ctx.String(503, "not ready\n")
}
