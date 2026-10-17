package agentextensionservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
)

var categoryLabels = map[agentextension.Category]string{
	agentextension.CategoryWebResearch: "Web research",
}

type capabilityDefinition struct {
	Label string
	Plan  platformplan.Capability
	Tools []agentextensionports.AgentExtensionTool
}

var capabilityDefinitions = map[agentextension.Capability]capabilityDefinition{
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
		},
		DataNotice: "Search queries and the addresses of pages agents read are sent to Exa. " +
			"Queries that contain Trenova record IDs, email addresses or phone numbers are " +
			"refused before they leave Trenova. After an agent reads web content, any change " +
			"it wants to make for the rest of that turn waits for a person's approval.",
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
