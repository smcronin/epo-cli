package cli

import (
	"strings"

	"github.com/smcronin/epo-cli/internal/config"
	epoerrors "github.com/smcronin/epo-cli/internal/errors"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage global CLI configuration",
		Long: `Manage global epo configuration.

Credentials are stored in a user-level config file, so commands work from any
directory without relying on a local .env file.`,
	}

	configCmd.AddCommand(newConfigSetCredsCmd())
	configCmd.AddCommand(newConfigShowCmd())
	return configCmd
}

func newConfigSetCredsCmd() *cobra.Command {
	var fromEnv bool
	var fromDotEnv string

	cmd := &cobra.Command{
		Use:     "set-creds [client-id] [client-secret]",
		Aliases: []string{"set-credentials"},
		Short:   "Persist EPO credentials in global config",
		Long: `Persist EPO credentials in global config.

Provide credentials as two arguments, load them from the current environment
(--from-env), or import them from a dotenv file (--from-dotenv).`,
		Args: cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientID, clientSecret, source, err := resolveConfigSetCredsInput(args, fromEnv, fromDotEnv)
			if err != nil {
				return err
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			cfg.ClientID = clientID
			cfg.ClientSecret = clientSecret
			if err := config.Save(cfg); err != nil {
				return err
			}

			path, err := config.ConfigFilePath()
			if err != nil {
				return err
			}

			return outputSuccess(cmd, map[string]any{
				"saved":            true,
				"source":           source,
				"configFile":       path,
				"clientIDMasked":   config.Mask(clientID),
				"clientSecretMask": config.Mask(clientSecret),
			})
		},
	}

	cmd.Flags().BoolVar(&fromEnv, "from-env", false, "Read credentials from current environment variables")
	cmd.Flags().StringVar(&fromDotEnv, "from-dotenv", "", "Read credentials from a dotenv file path")
	return cmd
}

func resolveConfigSetCredsInput(args []string, fromEnv bool, fromDotEnv string) (clientID, clientSecret, source string, err error) {
	sources := 0
	useArgs := false
	useFlags := false
	useEnv := false

	if len(args) > 0 {
		if len(args) != 2 {
			return "", "", "", &epoerrors.CLIError{
				Code:    400,
				Type:    "VALIDATION_ERROR",
				Message: "set-creds requires both client-id and client-secret when using arguments",
				Hint:    "Use: epo config set-creds <client-id> <client-secret>",
			}
		}
		useArgs = true
		sources++
	}

	flagID := strings.TrimSpace(flagClientID)
	flagSecret := strings.TrimSpace(flagClientSecret)
	if flagID != "" || flagSecret != "" {
		if flagID == "" || flagSecret == "" {
			return "", "", "", &epoerrors.CLIError{
				Code:    400,
				Type:    "VALIDATION_ERROR",
				Message: "both --client-id and --client-secret are required together",
			}
		}
		useFlags = true
		sources++
	}

	if fromEnv {
		useEnv = true
		sources++
	}

	fromDotEnv = strings.TrimSpace(fromDotEnv)
	if fromDotEnv != "" {
		sources++
	}

	if sources == 0 {
		return "", "", "", &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: "missing credentials source",
			Hint:    "Pass args, --client-id/--client-secret, --from-env, or --from-dotenv",
		}
	}
	if sources > 1 {
		return "", "", "", &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: "use exactly one credentials source",
			Hint:    "Choose one of args, --client-id/--client-secret, --from-env, or --from-dotenv",
		}
	}

	if useArgs {
		return strings.TrimSpace(args[0]), strings.TrimSpace(args[1]), "args", nil
	}
	if useFlags {
		return flagID, flagSecret, "flags", nil
	}
	if useEnv {
		clientID, clientSecret, idSource, secretSource := config.ResolveCredentials("", "", config.Config{})
		if clientID == "" || clientSecret == "" {
			return "", "", "", &epoerrors.CLIError{
				Code:    400,
				Type:    "VALIDATION_ERROR",
				Message: "environment credentials are incomplete",
				Hint:    "Set EPO_CLIENT_ID and EPO_CLIENT_SECRET (or supported aliases)",
			}
		}
		return clientID, clientSecret, idSource + "," + secretSource, nil
	}

	dotenvCfg, err := config.LoadCredentialsFromDotEnv(fromDotEnv)
	if err != nil {
		return "", "", "", err
	}
	if strings.TrimSpace(dotenvCfg.ClientID) == "" || strings.TrimSpace(dotenvCfg.ClientSecret) == "" {
		return "", "", "", &epoerrors.CLIError{
			Code:    400,
			Type:    "VALIDATION_ERROR",
			Message: "dotenv file is missing EPO credentials",
			Hint:    "Define EPO_CLIENT_ID and EPO_CLIENT_SECRET (or supported aliases)",
		}
	}
	return strings.TrimSpace(dotenvCfg.ClientID), strings.TrimSpace(dotenvCfg.ClientSecret), "dotenv", nil
}

func newConfigShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show global config path and stored credential status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			path, err := config.ConfigFilePath()
			if err != nil {
				return err
			}

			clientID := strings.TrimSpace(cfg.ClientID)
			clientSecret := strings.TrimSpace(cfg.ClientSecret)

			return outputSuccess(cmd, map[string]any{
				"configFile":         path,
				"configured":         clientID != "" && clientSecret != "",
				"clientIDConfigured": clientID != "",
				"clientIDMasked":     config.Mask(clientID),
				"secretConfigured":   clientSecret != "",
				"secretMasked":       config.Mask(clientSecret),
			})
		},
	}
	return cmd
}
