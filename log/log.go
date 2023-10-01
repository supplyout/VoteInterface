package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

var (
	Logger *zap.SugaredLogger
)

func InitLogger() {
	writeSyncer := getLogWriter()
	encoder := getEncoder()
	core := zapcore.NewCore(encoder, zapcore.NewMultiWriteSyncer(writeSyncer...), zapcore.DebugLevel)
	Logger = zap.New(core, zap.AddCaller()).Sugar()
	Logger.Info("已启用log")
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getLogWriter() []zapcore.WriteSyncer {
	writes := []zapcore.WriteSyncer{zapcore.AddSync(os.Stdout)}
	return writes
}

func ErrPanic(err error, logger *zap.SugaredLogger) {
	if err != nil {
		logger.Panic(err)
	}
	return
}
