package zapofx

import (
	"log/slog"

	slogzap "github.com/samber/slog-zap/v2"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"

	"github.com/go-toho/contrib/log/zapo"
	"github.com/go-toho/toho/contrib/log/slogo"
	"github.com/go-toho/toho/logger"
	"github.com/go-toho/toho/pkg/fxtags"
)

var Module = fx.Module("zap",
	provideLogger,
	provideSlogHandler,
	provideSetupLoggerWrapper,
)

var FxPrinterLogger = fx.Provide(newLoggerPrinter)

var FxEventLogger = fx.WithLogger(newFxEventLogger)

var (
	provideLogger = fx.Provide(zapo.New)

	provideSlogHandler = fx.Provide(
		fx.Annotate(
			func(config logger.Config, logger *zap.Logger) slog.Handler {
				return slogzap.Option{Level: slog.LevelDebug, Logger: logger}.NewZapHandler()
			},
			fx.ResultTags(fxtags.Group(slogo.GroupSlogHandler)),
		),
	)

	provideSetupLoggerWrapper = fx.Provide(
		fx.Annotate(
			newSetupLoggerWrapper,
			fx.ParamTags(fxtags.Named(logger.NamedFxSetupConfig)),
		),
	)
)

type loggerPrinter struct {
	l *zap.SugaredLogger
}

func newLoggerPrinter(logger *zap.SugaredLogger) fx.Printer {
	return loggerPrinter{l: logger}
}

func (p loggerPrinter) Printf(msg string, args ...interface{}) {
	log := p.l.Infow
	for i := 0; i < len(args); i = i + 2 {
		if k, ok := args[i].(string); ok && k == "error" {
			log = p.l.Errorw
			break
		}
	}
	log(msg, args...)
}

type setupLoggerWrapper struct {
	*zap.Logger
}

func newSetupLoggerWrapper(config *logger.Config) (*setupLoggerWrapper, error) {
	logger, _, err := zapo.New(*config)
	if err != nil {
		return nil, err
	}
	return &setupLoggerWrapper{Logger: logger}, nil
}

func newSlogFxEventLogger(logger *zap.Logger) fxevent.Logger {
	zapLogger := &fxevent.ZapLogger{Logger: logger}
	zapLogger.UseLogLevel(zap.DebugLevel)
	return zapLogger
}

func newFxEventLogger(logger *setupLoggerWrapper) fxevent.Logger {
	return newSlogFxEventLogger(logger.Logger)
}
