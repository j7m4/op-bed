/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package logging

import (
	"context"
	"fmt"
	"os"
	"time"

	_ "go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	"go.uber.org/zap/zapcore"
)

// InitLogger initializes OpenTelemetry logging
func InitLogger(ctx context.Context, serviceName string) (zapcore.Core, func(context.Context) error, error) {
	// Get OTLP endpoint from environment or use default
	endpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	if endpoint == "" {
		endpoint = "localhost:4317"
	}

	// Create OTLP log exporter
	exporter, err := otlploggrpc.New(
		ctx,
		otlploggrpc.WithEndpoint(endpoint),
		otlploggrpc.WithInsecure(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("creating OTLP log exporter: %w", err)
	}

	// Create resource with service information
	resource := sdkresource.NewWithAttributes(
		"", // Empty schema URL to avoid conflicts
		attribute.String("service.name", serviceName),
		attribute.String("service.version", "1.0.0"),
		attribute.String("environment", getEnvironment()),
	)

	// Create log provider
	logProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		//sdklog.WithProcessor(sdklog.NewSimpleProcessor(exporter)),
		sdklog.WithResource(resource),
	)

	logCore := NewOTelCore(logProvider, zapcore.DebugLevel)

	// Return shutdown function
	return logCore, logProvider.Shutdown, nil
}

// GetLogger returns a logger for the given component
func GetLogger(component string) log.Logger {
	return global.GetLoggerProvider().Logger(
		"github.com/example/op-hello-world",
		log.WithInstrumentationVersion("1.0.0"),
		log.WithInstrumentationAttributes(
			attribute.String("component", component),
		),
	)
}

// getEnvironment returns the current environment
func getEnvironment() string {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = "development"
	}
	return env
}

// LogError logs an error with the given logger
func LogError(logger log.Logger, err error, message string) {
	var record log.Record
	record.SetSeverity(log.SeverityError)
	record.SetBody(log.StringValue(message))
	record.AddAttributes(log.KeyValue{Key: "error", Value: log.StringValue(err.Error())})
	record.SetObservedTimestamp(time.Now())
	logger.Emit(context.Background(), record)
}

// LogInfo logs an info message with the given logger
func LogInfo(logger log.Logger, message string) {
	var record log.Record
	record.SetSeverity(log.SeverityInfo)
	record.SetBody(log.StringValue(message))
	record.SetObservedTimestamp(time.Now())
	logger.Emit(context.Background(), record)
}
