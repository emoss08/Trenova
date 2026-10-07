const EXT0 = { id: "websearch", n: "Web search", vendor: "Exa", cat: "Web research", m: "Ex", h: 0, c: 0, summary: "Let agents search the web and read pages when an answer is not in Trenova.", caps: ["Searches the open web by meaning or keywords", "Reads a page's text so an agent can quote it", "Cites every page it used in the answer", "Sends only the agent's question, never your records"], tools: ["web_search", "web_read"], notice: "Searches run under your organization's own Exa account. Exa sees the search text, never the records it came from.", enabled: false, key: false, avail: "SelectedAgents", depth: "auto", limit: 500, today: 0, agents: ["dispatch", "report"] };
const MEM_KIND = { Instruction: "b", Fact: "t", Correction: "w", Procedure: "v" };
const MEM_SUBJ = { Customer: "customer", Location: "location", Worker: "driver", Carrier: "carrier" };
const MEMORIES0 = [
  { id: "m1", kind: "Instruction", content: "Always copy Acme Foods' AP inbox (ap@acmefoods.com) on their invoices.", subj: ["Customer", "Acme Foods"], src: "Sarah Alvarez", reads: 14, last: "2 h ago", rec: "Sep 28" },
  { id: "m2", kind: "Procedure", content: "To release a billing hold for a missing POD: attach the POD, check the signature date matches delivery, then re-run the billing check.", tool: "release_billing_hold", src: "Billing exceptions", learned: true, reads: 21, last: "12 min ago", rec: "Sep 30" },
  { id: "m3", kind: "Fact", content: "Dock 4 at the Joliet DC closes at 15:00 on Fridays, so book pickups before 14:00.", subj: ["Location", "Joliet DC"], src: "Dispatch desk", learned: true, reads: 6, last: "Yesterday", rec: "Oct 2" },
  { id: "m4", kind: "Correction", content: "Detention at Kroger Romeoville starts after 2 hours, not 3.", subj: ["Customer", "Kroger Romeoville"], src: "Mike Chen", reads: 3, last: "Mon", rec: "Oct 5" },
  { id: "m5", kind: "Instruction", content: "Marcus Reed prefers not to run hazmat loads. Offer them to other drivers first.", subj: ["Worker", "Marcus Reed"], src: "Dana Ortiz", reads: 4, last: "Oct 4", rec: "Sep 21", until: "Dec 31" },
  { id: "m6", kind: "Fact", content: "Ridgeline Freight usually accepts Ohio lanes at $2.30–2.45 a mile.", subj: ["Carrier", "Ridgeline Freight"], src: "Dispatch desk", learned: true, reads: 9, last: "3 h ago", rec: "Sep 26" },
];
const SUGGESTIONS0 = [
  { id: "s1", content: "For Acme Foods shipments, put the PO number in the BOL reference field.", src: "Ratings", ratings: 4, people: 3, when: "Oct 6", quotes: ["It keeps leaving the PO off the BOL", "Acme wants the PO in the reference field"] },
  { id: "s2", content: "Before tendering to a carrier, check that their insurance runs past the delivery date.", src: "Reflection", agent: "Dispatch desk", when: "Oct 7, 4:10 AM", reason: "A person corrected two tenders this week because the carrier's insurance had expired." },
];
const MEM_EXAMPLES = [["Instruction", "Always CC the customer's AP inbox on invoices for …"], ["Fact", "The receiving dock at … closes at …"], ["Procedure", "To clear a rate mismatch: …"]];
const RETR0 = { paused: false, budget: 10, spent: 0, last: null, src: [
  { k: "Memory", n: "Memories", ic: "brain", d: "What the organization told its agents and what they recorded", on: true, total: 6, idx: 0, fail: 0, skip: 0 },
  { k: "Document", n: "Documents", ic: "receipt", d: "Document text, when the record it belongs to may be read by a model", on: true, total: 1842, idx: 0, fail: 0, skip: 63 },
  { k: "InboundMessage", n: "Inbound email", ic: "inbox", d: "The sender's own words, never the quoted thread", on: true, total: 312, idx: 0, fail: 0, skip: 0 },
], fails: [] };
const FAIL_ITEMS = [{ item: "BOL-88213.pdf", src: "Documents", err: "Text too long: 212 pages, limit is 150", n: 3 }, { item: "Rate confirmation — SHP-48190", src: "Documents", err: "Provider returned 413: request too large", n: 2 }];
Object.assign(window, { EXT0, MEM_KIND, MEM_SUBJ, MEMORIES0, SUGGESTIONS0, MEM_EXAMPLES, RETR0, FAIL_ITEMS });
