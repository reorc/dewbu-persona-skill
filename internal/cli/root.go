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
	Use:   "dewbu",
	Short: "Dewbu Persona query CLI",
	Long:  "Query user profiles, evidence, and tag statistics from the Dewbu Persona database.",
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
		fmt.Printf("dewbu %s\n", version)
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
	rootCmd.PersistentFlags().StringVar(&database, "db", "dewbu_persona_v2", "database name")
	rootCmd.PersistentFlags().StringVar(&format, "format", "json", "output format: json, table, csv")
	rootCmd.PersistentFlags().IntVar(&limit, "limit", 20, "max rows to return")
	rootCmd.PersistentFlags().IntVar(&offset, "offset", 0, "offset for pagination")
	rootCmd.PersistentFlags().StringVar(&fields, "fields", "", "comma-separated fields to return")
	rootCmd.PersistentFlags().StringVar(&backend, "backend", os.Getenv("DEWBU_BACKEND"), "query backend (default: http)")
	rootCmd.PersistentFlags().StringVar(&apiURL, "api-url", os.Getenv("DEWBU_API_BASE_URL"), "HTTP backend base URL")
	rootCmd.PersistentFlags().StringVar(&apiURL, "svc-base-url", os.Getenv("DEWBU_API_BASE_URL"), "HTTP backend service base URL")
	rootCmd.PersistentFlags().StringVar(&apiKey, "api-key", os.Getenv("DEWBU_API_KEY"), "HTTP backend API key")
}
