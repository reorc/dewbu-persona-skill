package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/reorc/dewbu-persona-skill/internal/db"
	"github.com/spf13/cobra"
)

var (
	configBackend    string
	configSvcBaseURL string
	configAPIKey     string
	configTimeout    int
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage local VOC CLI configuration",
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the local configuration path",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(db.DefaultConfigPath())
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show effective local configuration",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := db.CurrentConfig()
		fmt.Printf("path: %s\n", db.DefaultConfigPath())
		fmt.Printf("backend: %s\n", valueOrDefault(cfg.Backend, "http"))
		fmt.Printf("svc_base_url: %s\n", valueOrDefault(cfg.APIURL, "(not set)"))
		fmt.Printf("api_key: %s\n", maskKey(cfg.APIKey))
		if cfg.Timeout > 0 {
			fmt.Printf("timeout_seconds: %d\n", int(cfg.Timeout/time.Second))
		}
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Write local VOC CLI configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := db.DefaultConfigPath()
		cfg, err := db.LoadConfigFile(path)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}
			cfg = db.Config{}
		}

		if configBackend != "" {
			cfg.Backend = configBackend
		}
		if configSvcBaseURL != "" {
			cfg.APIURL = configSvcBaseURL
		}
		if configAPIKey != "" {
			cfg.APIKey = configAPIKey
		}
		if configTimeout > 0 {
			cfg.Timeout = time.Duration(configTimeout) * time.Second
		}
		if cfg.Backend == "" && cfg.APIURL != "" && cfg.APIKey != "" {
			cfg.Backend = "http"
		}

		if err := db.SaveConfigFile(path, cfg); err != nil {
			return err
		}
		fmt.Printf("Saved %s\n", path)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configPathCmd, configShowCmd, configSetCmd)
	configSetCmd.Flags().StringVar(&configBackend, "backend", "", "query backend")
	configSetCmd.Flags().StringVar(&configSvcBaseURL, "svc-base-url", "", "HTTP backend service base URL, for example https://example.com/api/cli")
	configSetCmd.Flags().StringVar(&configAPIKey, "api-key", "", "HTTP backend API key")
	configSetCmd.Flags().IntVar(&configTimeout, "timeout-seconds", 0, "HTTP backend timeout in seconds")
}

func valueOrDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func maskKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "(not set)"
	}
	if len(value) <= 12 {
		return "(set)"
	}
	return value[:8] + "..." + value[len(value)-4:]
}
