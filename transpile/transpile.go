//go:build !typescript

package transpile

import (
	"context"
	"fmt"
	"io"
)

var ErrTypescriptNotEnabled = fmt.Errorf("typescript support is not enabled on this build")

func TranspileTypescript(ctx context.Context, reader io.Reader) (string, error) {
	return "", ErrTypescriptNotEnabled
}
