package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/reorc/dewbu-persona-skill/internal/db"
	"github.com/spf13/cobra"
)

var (
	database string
	format   string
	limit    int
	offset   int
	fields   string
	backend  string
	apiURL   string
	apiKey   string
)

var (
	version   = "dev"
	buildTime = ""
	gitCommit = ""
)

var rootCmd = &cobra.Command{
	Use:   "voc",
	Short: "VOC persona analytics CLI",
	Long:  "Query user profiles, evidence, personas, and tag statistics from a configured VOC analytics deployment.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		fileConfig, err := db.LoadDefaultConfigFile()
		if err != nil {
			return err
		}
		db.Configure(fileConfig)
		db.Configure(db.Config{
			Backend: backend,
			APIURL:  apiURL,
			APIKey:  apiKey,
			Timeout: 30 * time.Second,
		})
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("voc %s\n", version)
		if buildTime != "" {
			fmt.Printf("  built: %s\n", buildTime)
		}
		if gitCommit != "" {
			fmt.Printf("  commit: %s\n", gitCommit)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func SetVersionInfo(v, bt, gc string) {
	version = v
	buildTime = bt
	gitCommit = gc
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&format, "format", "json", "output format: json, table, csv")
	rootCmd.PersistentFlags().IntVar(&limit, "limit", 20, "max rows to return")
	rootCmd.PersistentFlags().IntVar(&offset, "offset", 0, "offset for pagination")
	rootCmd.PersistentFlags().StringVar(&fields, "fields", "", "comma-separated fields to return")
	apiURLDefault := firstEnv("VOC_API_BASE_URL", "DEWBU_API_BASE_URL")
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", apiURLDefault, "API base URL")
	rootCmd.PersistentFlags().StringVar(&apiURL, "svc-base-url", apiURLDefault, "API service base URL")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", firstEnv("VOC_API_KEY", "DEWBU_API_KEY"), "API key")

	// Deprecated, ignored — the deployment (svc_base_url + api_key) determines
	// the brand/database. Kept hidden so older invocations don't break.
	rootCmd.PersistentFlags().StringVar(&database, "db", "", "deprecated, ignored")
	rootCmd.PersistentFlags().StringVar(&backend, "backend", firstEnv("VOC_BACKEND", "DEWBU_BACKEND"), "deprecated, ignored — HTTP is the only backend")
	_ = rootCmd.PersistentFlags().MarkHidden("db")
	_ = rootCmd.PersistentFlags().MarkHidden("backend")
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return ""
}
