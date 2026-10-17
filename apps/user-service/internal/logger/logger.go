package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Log is the global structured logger used across the application.
// It must be initialized via Init() before use.
var Log *zap.Logger

// Init configures the global Log to write JSON to stdout. App Service
// and Docker collect container stdout, so no log file is needed.
func Init() error {

	// Same keys as the gateway so both services parse the same way in
	// the log backend.
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "ts"
	encoderConfig.EncodeTime = zapcore.EpochTimeEncoder

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		zap.InfoLevel,
	)

	Log = zap.New(
		core,
		zap.AddCaller(),
	)

	return nil
}
