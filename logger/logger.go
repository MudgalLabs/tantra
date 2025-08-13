package logger

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type ctxKey struct{}

var once sync.Once
var logger *zap.SugaredLogger
var LEVEL, FILENAME string

func Init(level, filename string) {
	LEVEL = level
	FILENAME = filename
}

func istTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	ist, err := time.LoadLocation("Asia/Kolkata")
	if err != nil {
		enc.AppendString(t.UTC().Format("2006-01-02 15:04:05.000 UTC"))
		return
	}
	enc.AppendString(t.In(ist).Format("2006-01-02 15:04:05.000 MST"))
}

// Get initializes a zap.SugaredLogger instance if it has not been initialized
// already and returns the same instance for subsequent calls.
func Get() *zap.SugaredLogger {
	once.Do(func() {
		stdout := zapcore.AddSync(os.Stdout)

		// TODO: Have 2 log levels - development and production seperated.
		zapLevel := zap.InfoLevel
		levelEnv := LEVEL
		if levelEnv != "" {
			levelFromEnv, err := zapcore.ParseLevel(levelEnv)
			if err != nil {
				log.Println(
					fmt.Errorf("invalid level, defaulting to INFO: %w", err),
				)
			}

			zapLevel = levelFromEnv
		}

		logLevel := zap.NewAtomicLevelAt(zapLevel)

		developmentCfg := zap.NewDevelopmentEncoderConfig()
		developmentCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		developmentCfg.EncodeTime = istTimeEncoder
		consoleEncoder := zapcore.NewConsoleEncoder(developmentCfg)

		productionCfg := zap.NewProductionEncoderConfig()
		productionCfg.TimeKey = "timestamp"
		productionCfg.EncodeTime = istTimeEncoder

		fileEncoder := zapcore.NewJSONEncoder(productionCfg)

		var gitRevision string

		buildInfo, ok := debug.ReadBuildInfo()
		if ok {
			for _, v := range buildInfo.Settings {
				if v.Key == "vcs.revision" {
					gitRevision = v.Value
					break
				}
			}
		}

		file := zapcore.AddSync(&lumberjack.Logger{
			Filename:   FILENAME,
			MaxSize:    256, // megabytes
			MaxBackups: 4,   // Maximum number of old log files to keep
			MaxAge:     7,   // days
		})

		// log to multiple destinations (console and file)
		// extra fields are added to the JSON output alone
		core := zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, stdout, logLevel),
			zapcore.NewCore(fileEncoder, file, logLevel).
				With(
					[]zapcore.Field{
						zap.String("git_revision", gitRevision),
					},
				),
		)

		logger = zap.New(core, zap.AddCaller()).Sugar()
	})

	return logger
}

// FromCtx returns the Logger associated with the ctx. If no logger
// is associated, the default logger is returned, unless it is nil
// in which case a disabled logger is returned.
func FromCtx(ctx context.Context) *zap.SugaredLogger {
	if l, ok := ctx.Value(ctxKey{}).(*zap.SugaredLogger); ok {
		return l
	} else if l := logger; l != nil {
		return l
	}

	return zap.NewNop().Sugar()
}

// WithCtx returns a copy of ctx with the Logger attached.
func WithCtx(ctx context.Context, l *zap.SugaredLogger) context.Context {
	if lp, ok := ctx.Value(ctxKey{}).(*zap.SugaredLogger); ok {
		if lp == l {
			// Do not store same logger.
			return ctx
		}
	}

	return context.WithValue(ctx, ctxKey{}, l)
}
