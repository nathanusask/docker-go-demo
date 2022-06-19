package containerize

import "context"

type Interface interface {
	RunFactor(ctx context.Context, baseImage string, code string, factorNameLowercase string, paramArgs []string) error
}
