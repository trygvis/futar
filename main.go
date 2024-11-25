package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	_ "go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
)

var startupDelay int64 = 1000
var healthzErrorRate int64 = 20

var version = ""
var date = ""
var commit = ""

func main() {
	var v string
	//goland:noinspection GoBoolExpressions
	if len(version) > 0 {
		v = version
	} else {
		v = "local"

		if len(date) > 0 {
			v += ", date=" + date
		}

		if len(commit) > 0 {
			v += ", git=" + commit
		}
	}

	var err error
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	//metricDuration := 3 * time.Second
	metricDuration := 1 * time.Minute

	otelShutdown, err := setupOtelSDK(ctx, metricDuration)
	if err != nil {
		return
	}
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	logEnv()

	port, _ := os.LookupEnv("PORT")
	if port == "" {
		port = "8080"
	}

	env, _ := os.LookupEnv("ENV")
	if env == "" {
		env = "local"
	}

	startupDelayS, _ := os.LookupEnv("STARTUP_DELAY")
	if startupDelayS != "" {
		startupDelay, err = strconv.ParseInt(startupDelayS, 10, 64)
		if err != nil {
			return
		}
	}

	healthzErrorRateS, _ := os.LookupEnv("HEALTHZ_ERROR_RATE")
	if healthzErrorRateS != "" {
		healthzErrorRate, err = strconv.ParseInt(healthzErrorRateS, 10, 64)
		if err != nil {
			return
		}
	}

	e := echo.New()

	id := uuid.NewString()[:5]
	var mu = new(sync.Mutex)
	var cond = sync.NewCond(mu)
	server := FutarServer{
		version:          v,
		serviceName:      fmt.Sprintf("futar-%s", id),
		environment:      env,
		healthzErrorRate: healthzErrorRate,
		ready:            false,
		readyCond:        cond,
		readyMutex:       mu,
	}
	RegisterHandlers(e, &server)

	slog.Info("Application is starting")
	slog.Info("Config", "startupDelay", startupDelay)
	time.AfterFunc(time.Duration(startupDelay)*time.Millisecond, func() {
		server.markReady()
	})

	srvErr := make(chan error, 1)
	go func() {
		srvErr <- e.Start(":" + port)
	}()

	select {
	case err = <-srvErr:
		return
	case <-ctx.Done():
		slog.Info("Stopping!")
		stop()
	}

	err = e.Shutdown(context.Background())
	return
}

func logEnv() {
	env := os.Environ()
	slices.Sort(env)
	var b strings.Builder
	b.WriteString("Environment variables:\b")
	for _, val := range env {
		b.WriteString("\t")
		b.WriteString(val)
		b.WriteString("\n")
	}
	slog.Info(b.String())
}
