package agentextensionservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
)

var categoryLabels = map[agentextension.Category]string{
	agentextension.CategoryWebResearch:       "Web research",
	agentextension.CategoryBrowserAutomation: "Browser automation",
	agentextension.CategoryDataAnalysis:      "Data analysis",
}

type capabilityDefinition struct {
	Label string
	Plan  platformplan.Capability
	Tools []agentextensionports.AgentExtensionTool
}

var capabilityDefinitions = map[agentextension.Capability]capabilityDefinition{
	agentextension.CapabilityDataAnalysis: {
		Label: "Data analysis",
		Plan:  platformplan.CapabilityAgentDataAnalysis,
		Tools: []agentextensionports.AgentExtensionTool{
			{
				Name:        agentextension.ToolAnalyzeData,
				Label:       "Analyze data",
				Description: "Runs Python over report results and attached spreadsheets, and returns the figures and charts.",
			},
		},
	},
	agentextension.CapabilityCodeMode: {
		Label: "Read-only scripts",
		Plan:  platformplan.CapabilityAgentCodeMode,
		Tools: []agentextensionports.AgentExtensionTool{
			{
				Name:        agentextension.ToolRunScript,
				Label:       "Run a read-only script",
				Description: "Answers a question that needs many reads with one short Python script whose every read is the agent's own, checked as the person.",
			},
		},
	},
	agentextension.CapabilityPortalAutomation: {
		Label: "Portal work",
		Plan:  platformplan.CapabilityAgentPortalAutomation,
		Tools: []agentextensionports.AgentExtensionTool{
			{
				Name:        agentextension.ToolListPortalSites,
				Label:       "List portal sites",
				Description: "Lists the portals set up for agents and what each is for.",
			},
			{
				Name:        agentextension.ToolReadPortal,
				Label:       "Look something up on a portal",
				Description: "Reads a portal the organization set up and says what it shows, without submitting anything.",
			},
			{
				Name:        agentextension.ToolActOnPortal,
				Label:       "Do something on a portal",
				Description: "Books, uploads, accepts or updates something on a portal after a person approves it.",
			},
		},
	},
	agentextension.CapabilityWebResearch: {
		Label: "Web research",
		Plan:  platformplan.CapabilityAgentWebSearch,
		Tools: []agentextensionports.AgentExtensionTool{
			{
				Name:        agentextension.ToolWebSearch,
				Label:       "Search the web",
				Description: "Finds public pages about a topic and returns short passages from each.",
			},
			{
				Name:        agentextension.ToolWebRead,
				Label:       "Read a web page",
				Description: "Reads the full text of a page a search returned.",
			},
			{
				Name:        agentextension.ToolWatchWeb,
				Label:       "Watch the web",
				Description: "Keeps searching on a schedule and brings back only what is new, until the watch ends.",
			},
			{
				Name:        agentextension.ToolListWebWatches,
				Label:       "List web watches",
				Description: "Lists the watches set for the person and what each has found.",
			},
			{
				Name:        agentextension.ToolStopWebWatch,
				Label:       "Stop a web watch",
				Description: "Stops a watch the person no longer needs, so it searches and bills no more.",
			},
		},
	},
}

type definition struct {
	Type         agentextension.Type
	Name         string
	Vendor       string
	Summary      string
	Description  string
	Category     agentextension.Category
	BrandDomain  string
	DocsURL      string
	WebsiteURL   string
	PricingURL   string
	Capabilities []string
	DataNotice   string
	Featured     bool
	SortOrder    int
	ReleasedAt   int64
}

func (d *definition) title() string {
	return d.Vendor + " " + d.Name
}

var definitions = []definition{
	{
		Type:    agentextension.TypeExa,
		Name:    "Web search",
		Vendor:  "Exa",
		Summary: "Let agents search the web and read pages when an answer is not in Trenova.",
		Description: "Agents can look up current regulations, rules and public information " +
			"— hours of service, ELD requirements, hazmat placarding, IFTA filing — and " +
			"cite the page and its date instead of answering from memory. Government " +
			"sites are ranked first and marked as official.",
		Category:    agentextension.CategoryWebResearch,
		BrandDomain: "exa.ai",
		DocsURL:     "https://exa.ai/docs/reference/search",
		WebsiteURL:  "https://exa.ai",
		PricingURL:  "https://exa.ai/pricing",
		Capabilities: []string{
			"Searches the web and returns the most relevant passages from each page",
			"Ranks government sites (.gov, .gc.ca, .gob.mx) first and marks them official",
			"Reads a full page the search returned, in parts for long documents",
			"Answers cite the page and its publication date",
			"Keeps watching the web for what a person asked to hear about, and brings back only what is new",
		},
		DataNotice: "Search queries and the addresses of pages agents read are sent to Exa. " +
			"Queries that contain Trenova record IDs, email addresses or phone numbers are " +
			"refused before they leave Trenova. A web watch sends its search and what it " +
			"watches for to Exa, which runs it on a schedule until the watch ends. After an " +
			"agent reads web content, any change it wants to make for the rest of that turn " +
			"waits for a person's approval.",
		Featured:   true,
		SortOrder:  10,
		ReleasedAt: 1_790_121_600,
	},
	{
		Type:    agentextension.TypeParallel,
		Name:    "Web search",
		Vendor:  "Parallel",
		Summary: "Web search for agents from Parallel, the lowest-cost search an organization can bring.",
		Description: "The same web search agents already know — regulations, agency guidance, " +
			"industry news — run on the organization's Parallel account. Parallel reads " +
			"pages built with JavaScript and PDFs, and a fast search costs a fifth of a " +
			"standard one. Turn it on in place of Exa; agents keep the same tools.",
		Category:    agentextension.CategoryWebResearch,
		BrandDomain: "parallel.ai",
		DocsURL:     "https://docs.parallel.ai/search/search-quickstart",
		WebsiteURL:  "https://parallel.ai",
		PricingURL:  "https://parallel.ai/pricing",
		Capabilities: []string{
			"Searches the web and returns the passages that answer the question",
			"Ranks government sites (.gov, .gc.ca, .gob.mx) first and marks them official",
			"Reads a full page the search returned, including JavaScript pages and PDFs",
			"Answers cite the page and its publication date",
		},
		DataNotice: "Search queries and the addresses of pages agents read are sent to " +
			"Parallel. Queries that contain Trenova record IDs, email addresses or phone " +
			"numbers are refused before they leave Trenova. After an agent reads web " +
			"content, any change it wants to make for the rest of that turn waits for a " +
			"person's approval.",
		SortOrder:  20,
		ReleasedAt: 1_792_108_800,
	},
	{
		Type:    agentextension.TypeBrowserUse,
		Name:    "Portal work",
		Vendor:  "Browser Use",
		Summary: "Let agents read and act on the web portals your work runs on: dock appointments, shipper portals, tracking pages.",
		Description: "Agents look things up on the portals you set up — an appointment slot, a " +
			"load's status on a shipper's portal, a carrier's certificate on an insurer's site " +
			"— and, once a person approves, book, upload or accept there. Each portal is a site " +
			"you add with its addresses and sign-in; the browser never leaves those addresses, " +
			"and every task is recorded.",
		Category:    agentextension.CategoryBrowserAutomation,
		BrandDomain: "browser-use.com",
		DocsURL:     "https://docs.browser-use.com/cloud/api-v2-overview",
		WebsiteURL:  "https://browser-use.com",
		PricingURL:  "https://browser-use.com/pricing",
		Capabilities: []string{
			"Looks things up on the portals you set up and answers with what the portal shows",
			"Books, uploads or accepts on a portal only after a person approves the exact instruction",
			"Keeps the browser on each site's addresses and withholds what it found if it strays",
			"Signs in with stored credentials the models never see",
			"Records every task, with each page it visited, in AI control",
		},
		DataNotice: "The instruction, the details it names and the site's sign-in are sent to " +
			"Browser Use, which types them into that site. Sign-ins are stored encrypted and " +
			"never shown to a model. After an agent reads a portal, any change it wants to make " +
			"for the rest of that turn waits for a person's approval, and acting on a portal " +
			"always does.",
		SortOrder:  30,
		ReleasedAt: 1_792_108_800,
	},
	{
		Type:    agentextension.TypeE2B,
		Name:    "Data analysis",
		Vendor:  "E2B",
		Summary: "Let agents work out figures and charts from your reports and spreadsheets with Python.",
		Description: "Agents answer questions a report alone does not — margin by lane over a " +
			"quarter, which customers drift past their payment terms, where detention clusters " +
			"by facility and hour — by running Python over the report results and spreadsheets " +
			"they were given, and show the figures and charts beside the reply. The code runs " +
			"in a sandbox that cannot reach the internet.",
		Category:    agentextension.CategoryDataAnalysis,
		BrandDomain: "e2b.dev",
		DocsURL:     "https://e2b.dev/docs",
		WebsiteURL:  "https://e2b.dev",
		PricingURL:  "https://e2b.dev/pricing",
		Capabilities: []string{
			"Runs Python with pandas, numpy and matplotlib over report runs and attached files",
			"Shows the charts it draws beside the reply and in AI control",
			"Runs every analysis in a fresh sandbox with no internet access, stopped when it ends",
			"Reads only report runs the person ran and files they attached",
			"Records every analysis, its code and its output, in AI control",
		},
		DataNotice: "The rows of the report runs and the files an analysis reads, and the " +
			"code an agent writes, are sent to E2B for as long as the analysis runs. The " +
			"sandbox cannot reach the internet, so nothing it reads can be sent anywhere " +
			"else, and it is stopped when the analysis ends.",
		SortOrder:  40,
		ReleasedAt: 1_792_195_200,
	},
}

func definitionFor(typ agentextension.Type) (definition, bool) {
	for idx := range definitions {
		if definitions[idx].Type == typ {
			return definitions[idx], true
		}
	}

	return definition{}, false
}

func categoryOptions() []agentextensionports.AgentExtensionCategoryOption {
	categories := agentextension.AllCategories()
	options := make([]agentextensionports.AgentExtensionCategoryOption, 0, len(categories))
	for _, category := range categories {
		options = append(options, agentextensionports.AgentExtensionCategoryOption{
			Value: category,
			Label: categoryLabels[category],
		})
	}

	return options
}
