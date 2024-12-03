package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/labstack/echo/v4/middleware"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"log"
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
)

var startupDelay int64 = 10000
var healthzErrorRate int64 = 20

const prettyPrintOtel = true
const serviceName = "futar"

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

	instanceId := uuid.NewString()[:5]

	var err error
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	aiConnectionString, _ := os.LookupEnv("APPLICATIONINSIGHTS_CONNECTION_STRING")
	otelEnabled := strings.TrimSpace(aiConnectionString) != ""

	if otelEnabled {
		//metricDuration := 3 * time.Second
		metricDuration := 1 * time.Minute

		var otelSetup OtelSetup

		otelSetup, err = setupOtelSDK(serviceName, v, instanceId, prettyPrintOtel, ctx, metricDuration)
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			err = errors.Join(err, otelSetup.shutdown(context.Background()))
		}()

		// Connect slog to otel
		slogLogger := otelslog.NewLogger("futar", otelslog.WithLoggerProvider(otelSetup.loggerProvider))
		slog.SetDefault(slogLogger)
	}

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
	e.HideBanner = true
	if otelEnabled {
		e.Use(otelecho.Middleware(serviceName))
	} else {
		e.Use(middleware.Logger())
	}

	var mu = new(sync.Mutex)
	var cond = sync.NewCond(mu)
	server := FutarServer{
		version:          v,
		instanceId:       fmt.Sprintf("futar-%s", instanceId),
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
