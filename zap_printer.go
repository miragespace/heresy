package heresy

import (
	"github.com/dop251/goja_nodejs/console"
	"go.uber.org/zap"
)

const moduleName = "runtime:logger"

type zapPrinter struct {
	logger *zap.Logger
}

var _ console.Printer = (*zapPrinter)(nil)

func newZapPrinter(name string, logger *zap.Logger) console.Printer {
	return &zapPrinter{
		logger: logger.With(zap.String("script", name)),
	}
}

func (z *zapPrinter) Log(s string) {
	z.logger.Info(s)
}

func (z *zapPrinter) Warn(s string) {
	z.logger.Warn(s)
}

func (z *zapPrinter) Error(s string) {
	z.logger.Error(s)
}
