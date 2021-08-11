package liqoctl

import (
	"context"

	flag "github.com/spf13/pflag"
)

type InstallCommandGenerator interface {
	ValidateParameters(*flag.FlagSet) error
	GenerateCommand(ctx context.Context) (string, error)
}
