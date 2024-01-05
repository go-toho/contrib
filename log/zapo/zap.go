package zapo

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/go-toho/toho/logger"
)

func New(config logger.Config) (*zap.Logger, *zap.SugaredLogger, error) {
	cores := []zapcore.Core{}

	level, err := zapcore.ParseLevel(config.Level)
	if err != nil {
		return nil, nil, err
	}

	encoder := getEncoder(config.Format)
	writer := zapcore.Lock(os.Stdout)
	core := zapcore.NewCore(encoder, writer, level)
	cores = append(cores, core)

	combinedCore := zapcore.NewTee(cores...)

	var opts []zap.Option

	if config.Caller {
		opts = append(opts, zap.AddCaller())
	}

	logger := zap.New(combinedCore, opts...)

	return logger, logger.Sugar(), nil
}

func getEncoder(logFormat string) zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "time" // This will change the key from 'ts' to 'time'
	encoderConfig.EncodeTime = zapcore.RFC3339NanoTimeEncoder

	if logger.JSONFormat == logFormat {
		return zapcore.NewJSONEncoder(encoderConfig)
	}

	return zapcore.NewConsoleEncoder(encoderConfig)
}
