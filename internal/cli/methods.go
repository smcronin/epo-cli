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
				{Name: "--input-format", Default: "auto"},
				{Name: "--constituents", Default: "biblio"},
				{Name: "--flat", Description: "Flatten biblio payload to row fields"},
				{Name: "--summary", Description: "Return compact biblio summary"},
				{Name: "--stdin", Description: "Read multiple references from stdin"},
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
				{Name: "--input-format", Default: "auto"},
				{Name: "--stdin", Description: "Read multiple references from stdin"},
			},
			OutputShapeHint: "Published-data envelope with abstract payload",
			Examples: []string{
				"epo pub abstract EP1000000.A1 -f json -q",
			},
		},
		{
			Command:     "epo pub fulltext",
			Service:     "published-data",
			Implemented: true,
			Summary:     "Retrieve fulltext availability inquiry",
			Args: []methodArg{
				{Name: "reference", Required: true, Example: "EP1000000"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--ref-type", Default: "publication"},
				{Name: "--input-format", Default: "auto"},
				{Name: "--stdin", Description: "Read multiple references from stdin"},
			},
			OutputShapeHint: "Published-data envelope with fulltext inquiry payload",
			Examples: []string{
				"epo pub fulltext EP1000000 -f json -q",
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
				{Name: "--input-format", Default: "auto"},
				{Name: "--kind", Description: "Optional kind code when omitted from reference"},
				{Name: "--stdin", Description: "Read multiple references from stdin"},
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
				{Name: "--input-format", Default: "auto"},
				{Name: "--kind", Description: "Optional kind code when omitted from reference"},
				{Name: "--stdin", Description: "Read multiple references from stdin"},
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
				{Name: "--input-format", Default: "auto"},
				{Name: "--constituents", Description: "abstract, biblio, biblio,full-cycle, images"},
				{Name: "--stdin", Description: "Read multiple references from stdin"},
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
				{Name: "--query", Required: true, Description: "CQL expression"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--cql", Description: "Alias for --query"},
				{Name: "--q", Description: "Deprecated alias for --query"},
				{Name: "--range", Description: "Result range header (example 1-25)"},
				{Name: "--constituents", Description: "biblio,abstract,full-cycle"},
				{Name: "--post", Description: "Use POST body instead of GET query"},
				{Name: "--all", Description: "Auto-paginate all results"},
				{Name: "--sort", Description: "Sort results by publication date (pub-date-asc|pub-date-desc)"},
				{Name: "--stdin", Description: "Read one query per stdin line"},
				{Name: "--flat", Description: "Flatten search results to top-level fields"},
				{Name: "--enrich", Description: "Ensure biblio-enriched fields (title/pubDate) in flat output"},
				{Name: "--summary", Description: "Agent summary output {query,total,topResults}"},
				{Name: "--flat-pick", Description: "Enable --flat --enrich and set pick fields quickly"},
				{Name: "--table", Description: "Shortcut for --format table --flat with default fields"},
				{Name: "--pick", Description: "Project selected fields in output"},
			},
			OutputShapeHint: "Published-data envelope with search result set and pagination",
			Examples: []string{
				"epo pub search --query \"applicant=IBM\" --range 1-25 -f json -q",
				"epo pub search --query \"applicant=IBM and pd within \\\"20250101 20260304\\\"\" --all --sort pub-date-desc --flat -f json -q",
				"epo pub search --query \"applicant=IBM\" --summary --flat-pick \"reference,title,pubDate\" -f json -q",
				"epo pub search --query \"applicant=IBM\" --all --table",
				"echo \"applicant=IBM\" | epo pub search --stdin --all --table",
			},
		},
		{
			Command:     "epo pub images inquiry",
			Service:     "published-data",
			Implemented: true,
			Summary:     "List available image links for a publication reference",
			Args: []methodArg{
				{Name: "reference", Required: true, Example: "EP1000000.A1"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--ref-type", Default: "publication"},
				{Name: "--input-format", Default: "auto"},
				{Name: "--stdin", Description: "Read multiple references from stdin"},
			},
			OutputShapeHint: "Published-data envelope with document-instance links",
			Examples: []string{
				"epo pub images inquiry EP1000000.A1 -f json -q",
			},
		},
		{
			Command:     "epo pub images fetch",
			Service:     "published-data",
			Implemented: true,
			Summary:     "Fetch image/document content from an inquiry link path",
			Args: []methodArg{
				{Name: "link-path", Required: true, Example: "EP/1000000/A1/thumbnail"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--accept", Default: "application/pdf"},
				{Name: "--range", Description: "Single page selector for fullimage/tiff"},
				{Name: "--from", Description: "Source system (From)"},
				{Name: "--link", Description: "Accept raw inquiry @link path directly"},
				{Name: "--out", Description: "Write binary output to file"},
				{Name: "--include-body", Description: "Include base64 response body"},
				{Name: "--stdin", Description: "Read multiple link paths from stdin"},
			},
			OutputShapeHint: "Content metadata (type/bytes/hash) and optional body/file path",
			Examples: []string{
				"epo pub images fetch \"published-data/images/EP/1000000/A1/fullimage\" --link --range 1 --accept application/pdf -f json -q",
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
				{Name: "--flat", Description: "Flatten member rows"},
				{Name: "--table", Description: "Shortcut for --format table --flat"},
				{Name: "--stdin", Description: "Read multiple references from stdin"},
			},
			OutputShapeHint: "Family envelope with member documents and optional biblio/legal payload",
			Examples: []string{
				"epo family get EP.1000000.A1 --ref-type publication --input-format docdb -f json -q",
			},
		},
		{
			Command:     "epo family summary",
			Service:     "family",
			Implemented: true,
			Summary:     "Return condensed family summary with country counts",
			Args: []methodArg{
				{Name: "reference", Required: true, Example: "EP.1000000.A1"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--ref-type", Default: "publication"},
				{Name: "--input-format", Default: "docdb"},
				{Name: "--stdin", Description: "Read multiple references from stdin"},
			},
			OutputShapeHint: "Summary with member count and per-country breakdown",
			Examples: []string{
				"epo family summary EP.1000000.A1 -f json -q",
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
				{Name: "--from-format", Default: "auto"},
				{Name: "--to-format", Default: "epodoc"},
				{Name: "--guess-format", Description: "Auto-detect input format when from-format=auto"},
				{Name: "--auto-detect", Description: "Alias for --guess-format"},
				{Name: "--normalize", Description: "Return flattened conversion fields"},
				{Name: "--stdin", Description: "Read multiple references from stdin"},
			},
			OutputShapeHint: "Number-service envelope with converted reference formats",
			Examples: []string{
				"epo number convert EP.1000000.A1 --ref-type publication --from-format docdb --to-format epodoc -f json -q",
			},
		},
		{
			Command:     "epo number normalize",
			Service:     "number-service",
			Implemented: true,
			Summary:     "Auto-detect number format and normalize to docdb",
			Args: []methodArg{
				{Name: "reference", Required: true, Example: "EP1000000A1"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--ref-type", Default: "publication"},
				{Name: "--stdin", Description: "Read multiple references from stdin"},
			},
			OutputShapeHint: "Normalized docdb fields with detected input format",
			Examples: []string{
				"epo number normalize EP1000000A1 -f json -q",
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
				{Name: "--summary", Description: "Compact prosecution summary"},
				{Name: "--stdin", Description: "Read multiple references from stdin"},
			},
			OutputShapeHint: "Register envelope with dossier payload; combined constituents typically save quota calls versus separate endpoint calls",
			Examples: []string{
				"epo register get EP99203729 -f json -q",
				"epo register get EP99203729 --constituents biblio,events -f json -q",
			},
		},
		{
			Command:     "epo register events",
			Service:     "register",
			Implemented: true,
			Summary:     "Fetch register events endpoint (expects application epodoc reference)",
			Args: []methodArg{
				{Name: "reference", Required: true, Example: "EP99203729"},
			},
			OutputShapeHint: "Register envelope with event list",
			OptionalFlags: []methodFlag{
				{Name: "--stdin", Description: "Read multiple references from stdin"},
			},
			Examples: []string{
				"epo register events EP99203729 -f json -q",
			},
		},
		{
			Command:     "epo register procedural-steps",
			Service:     "register",
			Implemented: true,
			Summary:     "Fetch register procedural steps endpoint (expects application epodoc reference)",
			Args: []methodArg{
				{Name: "reference", Required: true, Example: "EP99203729"},
			},
			OutputShapeHint: "Register envelope with procedural steps",
			OptionalFlags: []methodFlag{
				{Name: "--stdin", Description: "Read multiple references from stdin"},
			},
			Examples: []string{
				"epo register procedural-steps EP99203729 -f json -q",
			},
		},
		{
			Command:     "epo register upp",
			Service:     "register",
			Implemented: true,
			Summary:     "Fetch unitary patent protection endpoint (publication or application-style ref)",
			Args: []methodArg{
				{Name: "reference", Required: true, Example: "EP99203729"},
			},
			OutputShapeHint: "Register envelope with UPP payload",
			OptionalFlags: []methodFlag{
				{Name: "--stdin", Description: "Read multiple references from stdin"},
			},
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
				{Name: "--all", Description: "Auto-paginate all results"},
				{Name: "--stdin", Description: "Read one query per stdin line"},
				{Name: "--pick", Description: "Project selected fields in output"},
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
				{Name: "--flat", Description: "Return simplified legal event rows"},
				{Name: "--summary", Description: "Return compact legal summary"},
				{Name: "--stdin", Description: "Read multiple references from stdin"},
			},
			OutputShapeHint: "Legal envelope with INPADOC events",
			Examples: []string{
				"epo legal get EP.1000000.A1 --ref-type publication --input-format docdb -f json -q",
			},
		},
		{
			Command:     "epo status",
			Service:     "status",
			Implemented: true,
			Summary:     "Combine legal, register, and procedural timeline views",
			Args: []methodArg{
				{Name: "reference", Required: true, Example: "EP.1000000.A1"},
			},
			OptionalFlags: []methodFlag{
				{Name: "--input-format", Default: "auto"},
				{Name: "--register-ref", Description: "Explicit application epodoc ref for register merge"},
			},
			OutputShapeHint: "Combined timeline payload with legal events and register/procedural summaries",
			Examples: []string{
				"epo status EP.1000000.A1 --register-ref EP99203729 -f json -q",
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
				{Name: "--normalize", Description: "Parse XML into structured rows"},
				{Name: "--parsed", Description: "Alias for --normalize"},
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
				{Name: "--normalize", Description: "Parse XML into structured rows"},
				{Name: "--parsed", Description: "Alias for --normalize"},
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
				{Name: "--normalize", Description: "Parse XML into mapping rows"},
				{Name: "--parsed", Description: "Alias for --normalize"},
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
				{Name: "--human-dates", Description: "Add human-readable dates beside timestamps"},
			},
			OutputShapeHint: "Usage envelope with message counts and response size",
			Examples: []string{
				"epo usage stats --date 01/03/2024 -f json -q",
				"epo usage stats --from 01/03/2024 --to 07/03/2024 -f json -q",
			},
		},
		{
			Command:         "epo usage today",
			Service:         "usage",
			Implemented:     true,
			Summary:         "Shortcut for current-day usage stats",
			OutputShapeHint: "Usage envelope for today",
			Examples: []string{
				"epo usage today -f json -q",
			},
		},
		{
			Command:         "epo usage week",
			Service:         "usage",
			Implemented:     true,
			Summary:         "Shortcut for trailing 7-day usage stats",
			OutputShapeHint: "Usage envelope for last 7 days",
			Examples: []string{
				"epo usage week -f json -q",
			},
		},
		{
			Command:         "epo usage quota",
			Service:         "usage",
			Implemented:     true,
			Summary:         "Show current quota and throttle counters only",
			OutputShapeHint: "Quota/throttle metadata without full usage payload",
			Examples: []string{
				"epo usage quota -f json -q",
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
				"MSYS_NO_PATHCONV=1 epo raw get \"/published-data/search\" --query q=applicant=IBM -f json -q",
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
				"MSYS_NO_PATHCONV=1 epo raw post \"/published-data/search\" --content-type text/plain --body \"q=applicant%3DIBM\" -f json -q",
			},
		},
	}
}
