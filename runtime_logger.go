package heresy

import (
	"github.com/dop251/goja_nodejs/console"
	"go.uber.org/zap"
)

const loggerModuleName = "runtime:logger"

type runtimeZapLogger struct {
	logger *zap.Logger
}

var _ console.Printer = (*runtimeZapLogger)(nil)

func newRuntimeLogger(name string, logger *zap.Logger) console.Printer {
	return &runtimeZapLogger{
		logger: logger.With(zap.String("script", name)),
	}
}

func (z *runtimeZapLogger) Log(s string) {
	z.logger.Info(s)
}

func (z *runtimeZapLogger) Warn(s string) {
	z.logger.Warn(s)
}

func (z *runtimeZapLogger) Error(s string) {
	z.logger.Error(s)
}
