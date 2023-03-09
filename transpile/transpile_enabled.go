//go:build typescript

package transpile

import (
	"context"
	"io"

	"github.com/clarkmcc/go-typescript"
)

func TranspileTypescript(ctx context.Context, reader io.Reader) (string, error) {
	return typescript.TranspileCtx(ctx, reader)
}
