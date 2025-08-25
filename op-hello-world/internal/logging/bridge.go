package logging

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.uber.org/zap/zapcore"
)

///////////////////////////////
// Custom code start
// OTelCore implements zapcore.Core to bridge zap logging to OpenTelemetry's log SDK

// OTelCore is a zapcore.Core implementation that forwards logs to an OpenTelemetry LoggerProvider
type OTelCore struct {
	logger   log.Logger
	minLevel zapcore.Level
	fields   []zapcore.Field
	ctx      context.Context
}

// NewOTelCore creates a new OTelCore that forwards logs to the given LoggerProvider
func NewOTelCore(provider *sdklog.LoggerProvider, minLevel zapcore.Level) *OTelCore {
	logger := provider.Logger("zap-bridge")
	return &OTelCore{
		logger:   logger,
		minLevel: minLevel,
		ctx:      context.Background(),
	}
}

// Enabled returns true if the given level is at or above the minimum level
func (c *OTelCore) Enabled(level zapcore.Level) bool {
	return level >= c.minLevel
}

// With adds structured context to the Core
func (c *OTelCore) With(fields []zapcore.Field) zapcore.Core {
	return &OTelCore{
		logger:   c.logger,
		minLevel: c.minLevel,
		fields:   append(c.fields, fields...),
		ctx:      c.ctx,
	}
}

// Check determines whether the supplied Entry should be logged
func (c *OTelCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(entry.Level) {
		return ce.AddCore(entry, c)
	}
	return ce
}

// Write serializes the Entry and any Fields supplied at the log site and writes them to the destination
func (c *OTelCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// Convert zap level to OTel severity
	severity := zapLevelToOTelSeverity(entry.Level)

	// Create log record
	record := log.Record{}
	record.SetTimestamp(entry.Time)
	record.SetSeverity(severity)
	record.SetSeverityText(entry.Level.String())
	record.SetBody(log.StringValue(entry.Message))

	// Add all fields as attributes
	allFields := append(c.fields, fields...)
	for _, field := range allFields {
		addFieldToRecord(&record, field)
	}

	// Add caller information if available
	if entry.Caller.Defined {
		record.AddAttributes(
			log.String("caller.file", entry.Caller.File),
			log.String("caller.function", entry.Caller.Function),
			log.Int("caller.line", entry.Caller.Line),
		)
	}

	// Add logger name if available
	if entry.LoggerName != "" {
		record.AddAttributes(log.String("logger.name", entry.LoggerName))
	}

	// Add stack trace if available
	if entry.Stack != "" {
		record.AddAttributes(log.String("stack", entry.Stack))
	}

	// Emit the log record
	c.logger.Emit(c.ctx, record)

	return nil
}

// Sync flushes buffered logs (noop for OTel)
func (c *OTelCore) Sync() error {
	// OpenTelemetry SDK handles flushing internally
	return nil
}

// zapLevelToOTelSeverity converts zap log levels to OpenTelemetry severity levels
func zapLevelToOTelSeverity(level zapcore.Level) log.Severity {
	switch level {
	case zapcore.DebugLevel:
		return log.SeverityDebug
	case zapcore.InfoLevel:
		return log.SeverityInfo
	case zapcore.WarnLevel:
		return log.SeverityWarn
	case zapcore.ErrorLevel:
		return log.SeverityError
	case zapcore.DPanicLevel:
		return log.SeverityError
	case zapcore.PanicLevel:
		return log.SeverityFatal
	case zapcore.FatalLevel:
		return log.SeverityFatal
	default:
		return log.SeverityInfo
	}
}

// addFieldToRecord adds a zap field to an OpenTelemetry log record as an attribute
func addFieldToRecord(record *log.Record, field zapcore.Field) {
	switch field.Type {
	case zapcore.BoolType:
		record.AddAttributes(log.Bool(field.Key, field.Integer == 1))
	case zapcore.Int8Type, zapcore.Int16Type, zapcore.Int32Type, zapcore.Int64Type,
		zapcore.Uint8Type, zapcore.Uint16Type, zapcore.Uint32Type, zapcore.Uint64Type:
		record.AddAttributes(log.Int64(field.Key, field.Integer))
	case zapcore.Float32Type, zapcore.Float64Type:
		record.AddAttributes(log.Float64(field.Key, field.Interface.(float64)))
	case zapcore.StringType:
		record.AddAttributes(log.String(field.Key, field.String))
	case zapcore.DurationType:
		record.AddAttributes(log.String(field.Key, time.Duration(field.Integer).String()))
	case zapcore.TimeType:
		if field.Interface != nil {
			if t, ok := field.Interface.(time.Time); ok {
				record.AddAttributes(log.String(field.Key, t.Format(time.RFC3339)))
			}
		}
	case zapcore.ErrorType:
		if field.Interface != nil {
			if err, ok := field.Interface.(error); ok {
				record.AddAttributes(log.String(field.Key, err.Error()))
			}
		}
	case zapcore.ReflectType, zapcore.StringerType:
		if field.Interface != nil {
			record.AddAttributes(log.String(field.Key, field.String))
		}
	default:
		// For unknown types, try to convert to string
		record.AddAttributes(log.String(field.Key, field.String))
	}
}

// Custom code end
///////////////////////////////
