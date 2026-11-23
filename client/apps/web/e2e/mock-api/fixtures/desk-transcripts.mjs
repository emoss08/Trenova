// Everyone's Desk conversations as the administrator's oversight page lists them, each with its
// transcript in the AgentRunTranscript shape the server opens it in, and the organization's
// exports of them. Timestamps are relative to the board's anchor.
const HOUR = 3600;
const DAY = 86400;

const PEOPLE = [
  { id: "usr_dana", name: "Dana Reyes" },
  { id: "usr_marcus", name: "Marcus Lee" },
  { id: "usr_priya", name: "Priya Shah" },
  { id: "usr_owen", name: "Owen Carter" },
];

const AGENTS = [
  { id: "agd_billing", name: "Billing exceptions" },
  { id: "agd_dispatch", name: "Dispatch planner" },
  { id: "agd_general", name: "Assistant" },
];

const TITLES = [
  ["Why is invoice INV-20419 short-paid", "Desk"],
  ["Detention on Acme Retail's Tuesday load", "Desk"],
  ["Cover the Dallas to Memphis reefer", "Watchtower"],
  ["Rate check for Northwind lanes", "Panel"],
  ["Driver hours for the Atlanta run", "Desk"],
  ["Morning briefing follow-up", "Briefing"],
  ["Import the Contoso shipment sheet", "Import"],
  ["Fuel surcharge formula for Q3", "Formula"],
  ["", "Panel"],
  ["Late delivery on PRO 88213", "Desk"],
  ["Carrier insurance expiring this week", "Watchtower"],
  ["Customer asked for a POD copy", "Desk"],
];

function transcript(anchor, title, person, agent, index) {
  const at = anchor - index * 5 * HOUR;
  const question = title || "Can you summarize what happened on my last load?";
  return {
    omittedMessages: 0,
    omittedAt: 0,
    messages: [
      {
        role: "User",
        kind: "Message",
        content: question,
        reasoning: "",
        toolCalls: [],
        toolCallId: "",
        toolName: "",
        toolFailed: false,
        toolVerdict: "",
        toolSummary: "",
        agentDefinitionId: "",
        delegateCallId: "",
        omitted: false,
        createdAt: at - 300,
      },
      {
        role: "Assistant",
        kind: "Message",
        content: "",
        reasoning: `Looking up what ${person.name} asked about before answering.`,
        toolCalls: [{ id: `call_${index}`, name: "shipment_list", arguments: { query: question } }],
        toolCallId: "",
        toolName: "",
        toolFailed: false,
        toolVerdict: "",
        toolSummary: "",
        agentDefinitionId: "",
        delegateCallId: "",
        omitted: false,
        createdAt: at - 290,
      },
      {
        role: "Tool",
        kind: "Message",
        content: '{"rows":3}',
        reasoning: "",
        toolCalls: [],
        toolCallId: `call_${index}`,
        toolName: "shipment_list",
        toolFailed: false,
        toolVerdict: "ran",
        toolSummary: "Read 3 shipments",
        agentDefinitionId: "",
        delegateCallId: "",
        omitted: false,
        createdAt: at - 285,
      },
      {
        role: "Assistant",
        kind: "Message",
        content: `${agent.name} found three shipments that match. The latest delivered two hours late and the customer was told at 3:40 PM.`,
        reasoning: "",
        toolCalls: [],
        toolCallId: "",
        toolName: "",
        toolFailed: false,
        toolVerdict: "",
        toolSummary: "",
        agentDefinitionId: "",
        delegateCallId: "",
        omitted: false,
        createdAt: at - 270,
      },
    ],
  };
}

export function createDeskTranscriptState(anchor) {
  const conversations = TITLES.map(([title, origin], index) => {
    const person = PEOPLE[index % PEOPLE.length];
    const agent = AGENTS[index % AGENTS.length];
    const lastMessageAt = anchor - index * 5 * HOUR - 270;
    return {
      __typename: "DeskConversation",
      id: `athr_oversight_${index + 1}`,
      ownerId: person.id,
      agentId: agent.id,
      ownerName: person.name,
      agentName: agent.name,
      title,
      origin,
      archived: index === 7,
      messageCount: 4,
      startedAt: lastMessageAt - 600,
      lastMessageAt,
      transcript: transcript(anchor, title, person, agent, index),
    };
  });

  const exports = [
    {
      __typename: "DeskTranscriptExport",
      id: "dtex_1",
      requestedByUserId: "usr_admin",
      requestedByName: "System Administrator",
      kind: "Filter",
      coverage: {
        kind: "Filter",
        timeframe: { preset: "Last30Days", fromDate: null, toDate: null },
        personName: null,
        agentName: null,
        origins: [],
        selectedCount: 0,
        ownerNames: PEOPLE.map((person) => person.name),
        ownerCount: PEOPLE.length,
      },
      status: "Succeeded",
      conversationCount: 12,
      byteSize: 184_320,
      artifactExpiresAt: anchor + 6 * DAY,
      errorMessage: null,
      startedAt: anchor - DAY,
      completedAt: anchor - DAY + 40,
      downloadable: true,
      createdAt: anchor - DAY,
    },
    {
      __typename: "DeskTranscriptExport",
      id: "dtex_2",
      requestedByUserId: "usr_priya",
      requestedByName: "Priya Shah",
      kind: "Filter",
      coverage: {
        kind: "Filter",
        timeframe: { preset: "Custom", fromDate: "2026-03-01", toDate: "2026-03-31" },
        personName: "Dana Reyes",
        agentName: "Billing exceptions",
        origins: [],
        selectedCount: 0,
        ownerNames: ["Dana Reyes"],
        ownerCount: 1,
      },
      status: "Succeeded",
      conversationCount: 41,
      byteSize: 902_144,
      artifactExpiresAt: anchor + 2 * DAY,
      errorMessage: null,
      startedAt: anchor - 5 * DAY,
      completedAt: anchor - 5 * DAY + 90,
      downloadable: false,
      createdAt: anchor - 5 * DAY,
    },
    {
      __typename: "DeskTranscriptExport",
      id: "dtex_3",
      requestedByUserId: "usr_admin",
      requestedByName: "System Administrator",
      kind: "Selection",
      coverage: {
        kind: "Selection",
        timeframe: null,
        personName: null,
        agentName: null,
        origins: [],
        selectedCount: 3,
        ownerNames: ["Marcus Lee", "Owen Carter"],
        ownerCount: 2,
      },
      status: "Failed",
      conversationCount: 3,
      byteSize: 0,
      artifactExpiresAt: null,
      errorMessage:
        "The export could not be written. Try again; if it fails again, export a shorter timeframe.",
      startedAt: anchor - 9 * DAY,
      completedAt: anchor - 9 * DAY + 20,
      downloadable: false,
      createdAt: anchor - 9 * DAY,
    },
  ];

  return { conversations, exports, people: PEOPLE, agents: AGENTS };
}
