package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/reorc/dewbu-persona-skill/internal/db"
	"github.com/spf13/cobra"
)

var (
	prBrand       string
	prName        string
	prDescription string
	prFilter      string
)

var personaCmd = &cobra.Command{
	Use:   "persona",
	Short: "Manage saved personas (list/get/create/update/delete/build)",
	Long: "Manage saved personas via the API.\n\n" +
		"Permissions follow your API key:\n" +
		"  - read-only (user) key: list, get\n" +
		"  - admin key:            create, update, delete, build\n",
}

var personaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List personas for a brand (read-only)",
	RunE:  runPersonaList,
}

var personaGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get a single persona by ID (read-only)",
	Args:  cobra.ExactArgs(1),
	RunE:  runPersonaGet,
}

var personaCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a persona (requires admin key)",
	RunE:  runPersonaCreate,
}

var personaUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update a persona's name/description/filters (requires admin key)",
	Args:  cobra.ExactArgs(1),
	RunE:  runPersonaUpdate,
}

var personaDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a persona by ID (requires admin key)",
	Args:  cobra.ExactArgs(1),
	RunE:  runPersonaDelete,
}

var personaBuildCmd = &cobra.Command{
	Use:   "build <id>",
	Short: "Recompute a persona's profile/stats (requires admin key)",
	Args:  cobra.ExactArgs(1),
	RunE:  runPersonaBuild,
}

func init() {
	rootCmd.AddCommand(personaCmd)
	personaCmd.AddCommand(personaListCmd, personaGetCmd, personaCreateCmd, personaUpdateCmd, personaDeleteCmd, personaBuildCmd)

	for _, c := range []*cobra.Command{personaListCmd, personaCreateCmd, personaBuildCmd} {
		c.Flags().StringVar(&prBrand, "brand", "", "brand id (dewbu|dn); defaults from --db")
	}
	personaCreateCmd.Flags().StringVar(&prName, "name", "", "persona name (required)")
	personaCreateCmd.Flags().StringVar(&prDescription, "description", "", "persona description")
	personaCreateCmd.Flags().StringVar(&prFilter, "filter", "", "persona filter config as JSON object (required)")

	personaUpdateCmd.Flags().StringVar(&prName, "name", "", "new persona name")
	personaUpdateCmd.Flags().StringVar(&prDescription, "description", "", "new persona description")
	personaUpdateCmd.Flags().StringVar(&prFilter, "filter", "", "new persona filter config as JSON object")
}

// brandID resolves the brand for a persona request: explicit --brand wins,
// otherwise it is derived from the --db database name.
func brandID() string {
	if strings.TrimSpace(prBrand) != "" {
		return strings.TrimSpace(prBrand)
	}
	switch database {
	case "dn_persona":
		return "dn"
	default:
		return "dewbu"
	}
}

// printJSON pretty-prints a decoded JSON value to stdout.
func printJSON(v interface{}) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

// describeError turns an APIError into an actionable, role-aware message.
func describeError(err error) error {
	if apiErr, ok := err.(*db.APIError); ok {
		switch apiErr.Status {
		case http.StatusUnauthorized:
			return fmt.Errorf("unauthorized (401): check your API key — run `dewbu config show`")
		case http.StatusForbidden:
			return fmt.Errorf("forbidden (403): %s — this operation needs an admin API key", apiErr.Message)
		case http.StatusNotFound:
			return fmt.Errorf("not found (404): %s", apiErr.Message)
		}
		return fmt.Errorf("%s (%d)", apiErr.Message, apiErr.Status)
	}
	return err
}

func runPersonaList(cmd *cobra.Command, args []string) error {
	var out map[string]interface{}
	path := "/v1/personas?brand=" + url.QueryEscape(brandID())
	if err := db.Request(http.MethodGet, path, nil, &out); err != nil {
		return describeError(err)
	}
	return printJSON(out)
}

func runPersonaGet(cmd *cobra.Command, args []string) error {
	var out map[string]interface{}
	path := "/v1/personas/" + url.PathEscape(args[0])
	if err := db.Request(http.MethodGet, path, nil, &out); err != nil {
		return describeError(err)
	}
	return printJSON(out)
}

func runPersonaCreate(cmd *cobra.Command, args []string) error {
	if strings.TrimSpace(prName) == "" {
		return fmt.Errorf("--name is required")
	}
	filters, err := parseFilterFlag(prFilter)
	if err != nil {
		return err
	}
	if filters == nil {
		return fmt.Errorf("--filter is required (a JSON object, e.g. '{\"stars\":[1,2]}')")
	}

	body := map[string]interface{}{
		"brand":   brandID(),
		"name":    prName,
		"filters": filters,
	}
	if cmd.Flags().Changed("description") {
		body["description"] = prDescription
	}

	var out map[string]interface{}
	if err := db.Request(http.MethodPost, "/v1/personas", body, &out); err != nil {
		return describeError(err)
	}
	return printJSON(out)
}

func runPersonaUpdate(cmd *cobra.Command, args []string) error {
	body := map[string]interface{}{}
	if cmd.Flags().Changed("name") {
		body["name"] = prName
	}
	if cmd.Flags().Changed("description") {
		body["description"] = prDescription
	}
	if cmd.Flags().Changed("filter") {
		filters, err := parseFilterFlag(prFilter)
		if err != nil {
			return err
		}
		body["filters"] = filters
	}
	if len(body) == 0 {
		return fmt.Errorf("nothing to update: pass at least one of --name, --description, --filter")
	}

	var out map[string]interface{}
	path := "/v1/personas/" + url.PathEscape(args[0])
	if err := db.Request(http.MethodPatch, path, body, &out); err != nil {
		return describeError(err)
	}
	return printJSON(out)
}

func runPersonaDelete(cmd *cobra.Command, args []string) error {
	var out map[string]interface{}
	path := "/v1/personas?id=" + url.QueryEscape(args[0])
	if err := db.Request(http.MethodDelete, path, nil, &out); err != nil {
		return describeError(err)
	}
	return printJSON(out)
}

func runPersonaBuild(cmd *cobra.Command, args []string) error {
	body := map[string]interface{}{"brand": brandID()}
	var out map[string]interface{}
	path := "/v1/personas/" + url.PathEscape(args[0])
	if err := db.Request(http.MethodPost, path, body, &out); err != nil {
		return describeError(err)
	}
	return printJSON(out)
}

// parseFilterFlag parses the --filter JSON into a generic object, validating
// that it is a JSON object. Returns nil when the flag is empty.
func parseFilterFlag(raw string) (map[string]interface{}, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var filters map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &filters); err != nil {
		return nil, fmt.Errorf("invalid --filter JSON: %w", err)
	}
	return filters, nil
}
