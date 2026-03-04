package cli

import "github.com/spf13/cobra"

type methodFlag struct {
	Name        string `json:"name"`
	Required    bool   `json:"required,omitempty"`
	Description string `json:"description,omitempty"`
	Default     string `json:"default,omitempty"`
}

type methodArg struct {
	Name        string `json:"name"`
	Required    bool   `json:"required,omitempty"`
	Description string `json:"description,omitempty"`
	Example     string `json:"example,omitempty"`
}

type methodCatalogEntry struct {
	Command         string       `json:"command"`
	Service         string       `json:"service"`
	Implemented     bool         `json:"implemented"`
	Summary         string       `json:"summary"`
	RequiredFlags   []methodFlag `json:"requiredFlags,omitempty"`
	OptionalFlags   []methodFlag `json:"optionalFlags,omitempty"`
	Args            []methodArg  `json:"args,omitempty"`
	OutputShapeHint string       `json:"outputShapeHint,omitempty"`
	Examples        []string     `json:"examples,omitempty"`
}

func newMethodsCmd() *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:   "methods",
		Short: "List command catalog for agent discovery",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = asJSON // Compatibility flag: the command always emits structured output.

			payload := responsePayload{
				Service: "catalog",
				Results: buildMethodCatalog(),
			}
			return outputSuccess(cmd, payload)
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "Emit JSON catalog (default behavior)")
	return cmd
}

func buildMethodCatalog() []methodCatalogEntry {
	return []methodCatalogEntry{
		{
			Command:     "epo auth configure",
			Service:     "auth",
			Implemented: true,
			Summary:     "Persist credentials to config file",
			OptionalFlags: []methodFlag{
				{Name: "--client-id", Description: "OPS client ID (consumer key)"},
				{Name: "--client-secret", Description: "OPS client secret"},
			},
			OutputShapeHint: "Configuration status with config file path and masked credentials",
			Examples: []string{
				"epo auth configure --client-id $EPO_CLIENT_ID --client-secret $EPO_CLIENT_SECRET -f json -q",
			},
		},
		{
			Command:     "epo auth token",
			Service:     "auth",
			Implemented: true,
			Summary:     "Request a fresh OAuth access token",
			OptionalFlags: []methodFlag{
				{Name: "--raw", Description: "Print access token only"},
				{Name: "--client-id", Description: "Override client ID"},
				{Name: "--client-secret", Description: "Override client secret"},
			},
			OutputShapeHint: "Token metadata plus access token and credential source",
			Examples: []string{
				"epo auth token -f json -q",
				"epo auth token --raw",
			},
		},
		{
			Command:         "epo auth check",
			Service:         "auth",
			Implemented:     true,
			Summary:         "Validate credentials by requesting a token",
			OutputShapeHint: "Validation status and token metadata",
			Examples: []string{
				"epo auth check -f json -q",
			},
		},
		{
			Command:     "epo config set-creds",
			Service:     "config",
			Implemented: true,
			Summary:     "Persist client credentials in global config",
			Args: []methodArg{
				{Name: "client-id", Required: false, Example: "your-client-id"},
				{Name: "client-secret", Required: false, Example: "your-client-secret"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--from-env", Description: "Load credentials from environment"},
				{Name: "--from-dotenv", Description: "Load credentials from dotenv file path"},
				{Name: "--client-id", Description: "Global flag source for client ID"},
				{Name: "--client-secret", Description: "Global flag source for client secret"},
			},
			OutputShapeHint: "Save status, source, config path, and masked credentials",
			Examples: []string{
				"epo config set-creds --from-env -f json -q",
				"epo config set-creds --from-dotenv .env -f json -q",
				"epo config set-creds my-id my-secret -f json -q",
			},
		},
		{
			Command:         "epo config show",
			Service:         "config",
			Implemented:     true,
			Summary:         "Show global config path and credential status",
			OutputShapeHint: "Config path plus masked credential status flags",
			Examples: []string{
				"epo config show -f json -q",
			},
		},
		{
			Command:     "epo update",
			Service:     "release",
			Implemented: true,
			Summary:     "Self-update binary from GitHub Releases",
			OptionalFlags: []methodFlag{
				{Name: "--check", Description: "Check latest version without installing"},
				{Name: "--version", Description: "Install a specific release tag"},
				{Name: "--force", Description: "Reinstall even if current version matches"},
				{Name: "--dry-run", Description: "Download and verify without replace"},
			},
			OutputShapeHint: "Version/asset/installation status with migration metadata",
			Examples: []string{
				"epo update --check -f json -q",
				"epo update --version v0.1.2 -f json -q",
			},
		},
		{
			Command:     "epo pub biblio",
			Service:     "published-data",
			Implemented: true,
			Summary:     "Retrieve bibliographic data",
			Args: []methodArg{
				{Name: "reference", Required: true, Description: "Publication/application/priority reference", Example: "EP1000000.A1"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--ref-type", Default: "publication"},
				{Name: "--input-format", Default: "epodoc"},
				{Name: "--constituents", Default: "biblio"},
			},
			OutputShapeHint: "Published-data envelope with throttle, quota, and JSON results",
			Examples: []string{
				"epo pub biblio EP1000000.A1 -f json -q",
			},
		},
		{
			Command:     "epo pub abstract",
			Service:     "published-data",
			Implemented: true,
			Summary:     "Retrieve abstract data",
			Args: []methodArg{
				{Name: "reference", Required: true, Example: "EP1000000.A1"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--ref-type", Default: "publication"},
				{Name: "--input-format", Default: "epodoc"},
			},
			OutputShapeHint: "Published-data envelope with abstract payload",
			Examples: []string{
				"epo pub abstract EP1000000.A1 -f json -q",
			},
		},
		{
			Command:     "epo pub claims",
			Service:     "published-data",
			Implemented: true,
			Summary:     "Retrieve claims fulltext",
			Args: []methodArg{
				{Name: "reference", Required: true, Example: "EP1000000.A1"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--ref-type", Default: "publication"},
				{Name: "--input-format", Default: "epodoc"},
			},
			OutputShapeHint: "Published-data envelope with claims payload",
			Examples: []string{
				"epo pub claims EP1000000.A1 -f json -q",
			},
		},
		{
			Command:     "epo pub description",
			Service:     "published-data",
			Implemented: true,
			Summary:     "Retrieve description fulltext",
			Args: []methodArg{
				{Name: "reference", Required: true, Example: "EP1000000.A1"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--ref-type", Default: "publication"},
				{Name: "--input-format", Default: "epodoc"},
			},
			OutputShapeHint: "Published-data envelope with description payload",
			Examples: []string{
				"epo pub description EP1000000.A1 -f json -q",
			},
		},
		{
			Command:     "epo pub equivalents",
			Service:     "published-data",
			Implemented: true,
			Summary:     "Retrieve simple-family equivalent publications",
			Args: []methodArg{
				{Name: "reference", Required: true, Example: "EP1000000.A1"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--ref-type", Default: "publication"},
				{Name: "--input-format", Default: "epodoc"},
				{Name: "--constituents", Description: "abstract, biblio, biblio,full-cycle, images"},
			},
			OutputShapeHint: "Published-data envelope with equivalents payload",
			Examples: []string{
				"epo pub equivalents EP1000000.A1 --constituents biblio -f json -q",
			},
		},
		{
			Command:     "epo pub search",
			Service:     "published-data",
			Implemented: true,
			Summary:     "Run CQL search over published-data",
			RequiredFlags: []methodFlag{
				{Name: "--q", Required: true, Description: "CQL expression"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--range", Description: "Result range header (example 1-25)"},
				{Name: "--constituents", Description: "biblio,abstract,full-cycle"},
				{Name: "--post", Description: "Use POST body instead of GET query"},
			},
			OutputShapeHint: "Published-data envelope with search result set and pagination",
			Examples: []string{
				"epo pub search --q \"applicant=IBM\" --range 1-25 -f json -q",
			},
		},
		{
			Command:         "epo methods --json",
			Service:         "catalog",
			Implemented:     true,
			Summary:         "Discover command surface and argument contracts",
			OutputShapeHint: "Array of command descriptors with flags, args, and examples",
			Examples: []string{
				"epo methods --json -f json -q",
			},
		},
		{
			Command:     "epo family get",
			Service:     "family",
			Implemented: true,
			Summary:     "Retrieve INPADOC family members and optional constituents",
			Args: []methodArg{
				{Name: "reference", Required: true, Example: "EP.1000000.A1"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--ref-type", Default: "publication"},
				{Name: "--input-format", Default: "docdb"},
				{Name: "--constituents", Description: "biblio, legal, or biblio,legal"},
			},
			OutputShapeHint: "Family envelope with member documents and optional biblio/legal payload",
			Examples: []string{
				"epo family get EP.1000000.A1 --ref-type publication --input-format docdb -f json -q",
			},
		},
		{
			Command:     "epo number convert",
			Service:     "number-service",
			Implemented: true,
			Summary:     "Convert number formats via number-service",
			Args: []methodArg{
				{Name: "reference", Required: true, Example: "EP.1000000.A1"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--ref-type", Default: "application"},
				{Name: "--from-format", Default: "docdb"},
				{Name: "--to-format", Default: "epodoc"},
			},
			OutputShapeHint: "Number-service envelope with converted reference formats",
			Examples: []string{
				"epo number convert EP.1000000.A1 --ref-type publication --from-format docdb --to-format epodoc -f json -q",
			},
		},
		{
			Command:     "epo register get",
			Service:     "register",
			Implemented: true,
			Summary:     "Fetch EP register dossier data",
			Args: []methodArg{
				{Name: "reference", Required: true, Example: "EP99203729"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--constituents", Description: "biblio,events,procedural-steps"},
			},
			OutputShapeHint: "Register envelope with dossier payload",
			Examples: []string{
				"epo register get EP99203729 -f json -q",
				"epo register get EP99203729 --constituents biblio,events -f json -q",
			},
		},
		{
			Command:     "epo register events",
			Service:     "register",
			Implemented: true,
			Summary:     "Fetch register events endpoint",
			Args: []methodArg{
				{Name: "reference", Required: true, Example: "EP99203729"},
			},
			OutputShapeHint: "Register envelope with event list",
			Examples: []string{
				"epo register events EP99203729 -f json -q",
			},
		},
		{
			Command:     "epo register procedural-steps",
			Service:     "register",
			Implemented: true,
			Summary:     "Fetch register procedural steps endpoint",
			Args: []methodArg{
				{Name: "reference", Required: true, Example: "EP99203729"},
			},
			OutputShapeHint: "Register envelope with procedural steps",
			Examples: []string{
				"epo register procedural-steps EP99203729 -f json -q",
			},
		},
		{
			Command:     "epo register upp",
			Service:     "register",
			Implemented: true,
			Summary:     "Fetch unitary patent protection endpoint",
			Args: []methodArg{
				{Name: "reference", Required: true, Example: "EP99203729"},
			},
			OutputShapeHint: "Register envelope with UPP payload",
			Examples: []string{
				"epo register upp EP99203729 -f json -q",
			},
		},
		{
			Command:     "epo register search",
			Service:     "register",
			Implemented: true,
			Summary:     "Run CQL search against register",
			RequiredFlags: []methodFlag{
				{Name: "--q", Required: true, Description: "CQL expression"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--range", Description: "Result range header (example 1-25)"},
				{Name: "--post", Description: "Use POST body instead of GET query"},
			},
			OutputShapeHint: "Register envelope with search results",
			Examples: []string{
				"epo register search --q \"pa=IBM\" --range 1-25 -f json -q",
			},
		},
		{
			Command:     "epo legal get",
			Service:     "legal",
			Implemented: true,
			Summary:     "Fetch legal status events",
			Args: []methodArg{
				{Name: "reference", Required: true, Example: "EP.1000000.A1"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--ref-type", Default: "publication"},
				{Name: "--input-format", Default: "docdb"},
			},
			OutputShapeHint: "Legal envelope with INPADOC events",
			Examples: []string{
				"epo legal get EP.1000000.A1 --ref-type publication --input-format docdb -f json -q",
			},
		},
		{
			Command:     "epo cpc get",
			Service:     "classification/cpc",
			Implemented: true,
			Summary:     "Retrieve CPC symbol details",
			Args: []methodArg{
				{Name: "symbol", Required: true, Example: "H04W"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--depth", Description: "Depth integer or all"},
				{Name: "--navigation", Description: "Include prev/next nodes"},
				{Name: "--ancestors", Description: "Include ancestor nodes"},
				{Name: "--accept", Default: "application/cpc+xml"},
			},
			OutputShapeHint: "CPC envelope (XML body in raw field by default)",
			Examples: []string{
				"epo cpc get H04W --depth 1 -f json -q",
			},
		},
		{
			Command:     "epo cpc search",
			Service:     "classification/cpc",
			Implemented: true,
			Summary:     "Search CPC symbols by keyword",
			RequiredFlags: []methodFlag{
				{Name: "--q", Required: true, Description: "Search keyword"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--range", Description: "Range window, example 1-20"},
				{Name: "--accept", Default: "application/cpc+xml"},
			},
			OutputShapeHint: "CPC envelope (XML body in raw field by default)",
			Examples: []string{
				"epo cpc search --q chemistry --range 1-20 -f json -q",
			},
		},
		{
			Command:     "epo cpc media",
			Service:     "classification/cpc",
			Implemented: true,
			Summary:     "Fetch CPC media asset",
			Args: []methodArg{
				{Name: "media-id", Required: true, Example: "1000.gif"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--accept", Default: "image/gif"},
				{Name: "--out", Description: "Write binary payload to file"},
				{Name: "--include-body", Description: "Include base64 body in JSON output"},
			},
			OutputShapeHint: "CPC envelope with media metadata and optional base64",
			Examples: []string{
				"epo cpc media 1000.gif --out cpc.gif -f json -q",
			},
		},
		{
			Command:     "epo cpc map",
			Service:     "classification/cpc",
			Implemented: true,
			Summary:     "Map CPC/ECLA/IPC symbols",
			Args: []methodArg{
				{Name: "symbol", Required: true, Example: "A61K9/00"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--from", Default: "cpc"},
				{Name: "--to", Default: "ecla"},
				{Name: "--additional", Description: "Request additional mapping context"},
				{Name: "--accept", Default: "application/cpc+xml"},
			},
			OutputShapeHint: "CPC mapping envelope (XML body in raw field by default)",
			Examples: []string{
				"epo cpc map A61K9/00 --from cpc --to ecla -f json -q",
			},
		},
		{
			Command:     "epo usage stats",
			Service:     "usage",
			Implemented: true,
			Summary:     "Fetch OPS usage stats",
			OptionalFlags: []methodFlag{
				{Name: "--date", Description: "Single date dd/mm/yyyy"},
				{Name: "--from", Description: "Range start dd/mm/yyyy"},
				{Name: "--to", Description: "Range end dd/mm/yyyy"},
			},
			OutputShapeHint: "Usage envelope with message counts and response size",
			Examples: []string{
				"epo usage stats --date 01/03/2024 -f json -q",
				"epo usage stats --from 01/03/2024 --to 07/03/2024 -f json -q",
			},
		},
		{
			Command:     "epo raw get",
			Service:     "raw",
			Implemented: true,
			Summary:     "Run a raw GET request",
			Args: []methodArg{
				{Name: "path", Required: true, Example: "/published-data/search"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--base-url", Default: "https://ops.epo.org/rest-services"},
				{Name: "--accept", Default: "application/json"},
				{Name: "--query", Description: "Repeatable key=value pair"},
			},
			OutputShapeHint: "Raw envelope with direct service response",
			Examples: []string{
				"epo raw get /published-data/search --query q=applicant=IBM -f json -q",
			},
		},
		{
			Command:     "epo raw post",
			Service:     "raw",
			Implemented: true,
			Summary:     "Run a raw POST request",
			Args: []methodArg{
				{Name: "path", Required: true, Example: "/published-data/search"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--base-url", Default: "https://ops.epo.org/rest-services"},
				{Name: "--accept", Default: "application/json"},
				{Name: "--content-type", Default: "application/json"},
				{Name: "--query", Description: "Repeatable key=value pair"},
				{Name: "--body", Description: "Inline request body"},
				{Name: "--body-file", Description: "Body file path"},
			},
			OutputShapeHint: "Raw envelope with direct service response",
			Examples: []string{
				"epo raw post /published-data/search --content-type text/plain --body \"q=applicant%3DIBM\" -f json -q",
			},
		},
	}
}
