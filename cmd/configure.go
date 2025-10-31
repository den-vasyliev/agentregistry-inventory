package cmd

import (
	"github.com/agentregistry-dev/agentregistry/cmd/configure"
)

func init() {
	rootCmd.AddCommand(configure.NewConfigureCmd())
}
