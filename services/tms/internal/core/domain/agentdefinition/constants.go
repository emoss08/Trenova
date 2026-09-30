package agentdefinition

// insightAnalystDailyRuns bounds how often the insight analyst runs on its own
// schedule. Insight is a read-only narrator, so a day is the most it can usefully
// produce.
const insightAnalystDailyRuns = 20

// toolSearchDocuments is the one tool the insight analyst always starts with:
// it needs to read the record before it can say anything about it.
const toolSearchDocuments = "search_documents"
