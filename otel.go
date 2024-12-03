package main

import (
	"context"
	"errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"time"
)

const useStdout = false

type OtelSetup struct {
	shutdown       func(context.Context) error
	loggerProvider *log.LoggerProvider
}

// setupOtelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func setupOtelSDK(serviceName, serviceVersion, instanceId string, prettyPrint bool, ctx context.Context, metricInterval time.Duration) (setup OtelSetup, err error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	setup.shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, setup.shutdown(ctx))
	}

	var attrs []attribute.KeyValue

	if serviceName != "" {
		attrs = append(attrs, semconv.ServiceName(serviceName))
	}
	if serviceVersion != "" {
		attrs = append(attrs, semconv.ServiceVersion(serviceVersion))
	}
	if instanceId != "" {
		attrs = append(attrs, semconv.ServiceInstanceID(instanceId))
	}

	res, err := resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL, attrs...),
	)
	if err != nil {
		return
	}

	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// Set up trace provider.
	tracerProvider, err := newTraceProvider(prettyPrint, ctx, res)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	// Set up meter provider.
	meterProvider, err := newMeterProvider(prettyPrint, ctx, res, metricInterval)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	otel.SetMeterProvider(meterProvider)

	// Set up logger provider.
	setup.loggerProvider, err = newLoggerProvider(prettyPrint, ctx, res)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, setup.loggerProvider.Shutdown)
	global.SetLoggerProvider(setup.loggerProvider)

	return
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func newTraceProvider(prettyPrint bool, ctx context.Context, res *resource.Resource) (*trace.TracerProvider, error) {
	var traceExporter trace.SpanExporter
	var err error

	if useStdout {
		var opts []stdouttrace.Option
		if prettyPrint {
			opts = append(opts, stdouttrace.WithPrettyPrint())
		}

		traceExporter, err = stdouttrace.New(opts...)
	} else {
		traceExporter, err = otlptracehttp.New(ctx)
	}
	if err != nil {
		return nil, err
	}

	traceProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter,
			// Default is 5s. Set to 1s for demonstrative purposes.
			trace.WithBatchTimeout(time.Second)),
		trace.WithResource(res),
	)
	return traceProvider, nil
}

func newMeterProvider(prettyPrint bool, ctx context.Context, res *resource.Resource, metricInterval time.Duration) (*metric.MeterProvider, error) {
	var metricExporter metric.Exporter
	var err error

	if useStdout {
		var opts []stdoutmetric.Option
		if prettyPrint {
			opts = append(opts, stdoutmetric.WithPrettyPrint())
		}

		metricExporter, err = stdoutmetric.New(opts...)
	} else {
		metricExporter, err = otlpmetrichttp.New(ctx)
	}
	if err != nil {
		return nil, err
	}

	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			// Default is 1m. Set to 3s for demonstrative purposes.
			metric.WithInterval(metricInterval))),
		metric.WithResource(res),
	)
	return meterProvider, nil
}

func newLoggerProvider(prettyPrint bool, ctx context.Context, res *resource.Resource) (*log.LoggerProvider, error) {
	var logExporter log.Exporter
	var err error

	if useStdout {
		var opts []stdoutlog.Option
		if prettyPrint {
			opts = append(opts, stdoutlog.WithPrettyPrint())
		}

		logExporter, err = stdoutlog.New(opts...)
	} else {
		var opts []otlploghttp.Option
		logExporter, err = otlploghttp.New(ctx, opts...)
	}

	if err != nil {
		return nil, err
	}

	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
		log.WithResource(res),
	)
	return loggerProvider, nil
}
