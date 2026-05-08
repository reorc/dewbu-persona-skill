package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	database string
	format   string
	limit    int
	offset   int
	fields   string
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
}
