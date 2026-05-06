package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func confirmDestructiveAction(cmd *cobra.Command, yes bool, action, id string) error {
	if yes {
		return nil
	}
	answer, err := readLine(cmd.InOrStdin(), cmd.OutOrStdout(), fmt.Sprintf("%s %s? Type yes to confirm: ", action, id))
	if err != nil {
		return err
	}
	answer = strings.TrimSpace(answer)
	if strings.EqualFold(answer, "yes") || strings.EqualFold(answer, "y") {
		return nil
	}
	return fmt.Errorf("%s %s canceled", action, id)
}
