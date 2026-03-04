package cli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/smcronin/epo-cli/internal/auth"
	"github.com/smcronin/epo-cli/internal/config"
	epoerrors "github.com/smcronin/epo-cli/internal/errors"
	"github.com/spf13/cobra"
)

func newAuthCmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate and manage OPS credentials",
	}

	authCmd.AddCommand(newAuthConfigureCmd())
	authCmd.AddCommand(newAuthTokenCmd())
	authCmd.AddCommand(newAuthCheckCmd())

	return authCmd
}

func newAuthConfigureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Persist client credentials to local config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			clientID, clientSecret, _, _ := config.ResolveCredentials(flagClientID, flagClientSecret, cfg)
			if strings.TrimSpace(clientID) == "" || strings.TrimSpace(clientSecret) == "" {
				return &epoerrors.CLIError{
					Code:    400,
					Type:    "VALIDATION_ERROR",
					Message: "missing credentials for configure",
					Hint:    "Pass --client-id and --client-secret (or set env vars)",
				}
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
				"configured":       true,
				"configFile":       path,
				"clientIDMasked":   config.Mask(clientID),
				"clientSecretMask": config.Mask(clientSecret),
			})
		},
	}
	return cmd
}

func newAuthTokenCmd() *cobra.Command {
	var raw bool

	cmd := &cobra.Command{
		Use:   "token",
		Short: "Request a fresh OPS access token",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientID, clientSecret, idSource, secretSource, err := resolveRuntimeCredentials()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flagTimeout)*time.Second)
			defer cancel()

			httpClient := &http.Client{Timeout: time.Duration(flagTimeout) * time.Second}
			token, err := auth.RequestToken(ctx, httpClient, clientID, clientSecret)
			if err != nil {
				return err
			}

			if raw {
				fmt.Fprintln(os.Stdout, token.AccessToken)
				return nil
			}

			return outputSuccess(cmd, map[string]any{
				"tokenType":   token.TokenType,
				"expiresIn":   token.ExpiresIn,
				"scope":       token.Scope,
				"status":      token.Status,
				"issuedAt":    token.IssuedAt,
				"accessToken": token.AccessToken,
				"credentialSources": map[string]string{
					"clientID":     idSource,
					"clientSecret": secretSource,
				},
			})
		},
	}

	cmd.Flags().BoolVar(&raw, "raw", false, "Print only the access token")
	return cmd
}

func newAuthCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Validate configured credentials by requesting a token",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientID, clientSecret, idSource, secretSource, err := resolveRuntimeCredentials()
			if err != nil {
				return err
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(flagTimeout)*time.Second)
			defer cancel()

			httpClient := &http.Client{Timeout: time.Duration(flagTimeout) * time.Second}
			token, err := auth.RequestToken(ctx, httpClient, clientID, clientSecret)
			if err != nil {
				return err
			}

			return outputSuccess(cmd, map[string]any{
				"ok":        true,
				"tokenType": token.TokenType,
				"expiresIn": token.ExpiresIn,
				"scope":     token.Scope,
				"status":    token.Status,
				"credentialSources": map[string]string{
					"clientID":     idSource,
					"clientSecret": secretSource,
				},
			})
		},
	}
	return cmd
}

func resolveRuntimeCredentials() (clientID, clientSecret, idSource, secretSource string, err error) {
	cfg, err := config.Load()
	if err != nil {
		return "", "", "", "", err
	}

	clientID, clientSecret, idSource, secretSource = config.ResolveCredentials(flagClientID, flagClientSecret, cfg)
	if strings.TrimSpace(clientID) == "" || strings.TrimSpace(clientSecret) == "" {
		return "", "", "", "", &epoerrors.CLIError{
			Code:    401,
			Type:    "AUTH_FAILURE",
			Message: "missing EPO OPS credentials",
			Hint:    "Run `epo auth configure --client-id ... --client-secret ...` or set env vars",
		}
	}
	return clientID, clientSecret, idSource, secretSource, nil
}
