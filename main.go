package main

import (
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
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
	server := FutarServer{
		version:          v,
		serviceName:      "futar",
		environment:      env,
		healthzErrorRate: healthzErrorRate,
	}
	RegisterHandlers(e, &server)

	slog.Info("Application is starting")
	time.AfterFunc(time.Duration(startupDelay)*time.Millisecond, func() {
		server.markReady()
	})

	e.Logger.Fatal(e.Start(":" + port))
}
