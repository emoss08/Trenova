# Agent profiles

An agent used to know an organization by its name and timezone, and a person by their name,
email and roles. It did not know whether the organization runs trucks or brokers loads, for
whom, on which lanes, with which integrations; or that the person asking about "my invoices"
works the billing queue for the Midwest accounts. Memory could not fill the gap: a memory is
a rule or fact someone thought to record, not a picture of the business.

A **profile** is that picture. There are two kinds, built the same way:

| Subject | Written from | Read by |
|---|---|---|
| `Organization` | 90 days of the organization's records, its settings and integrations | every turn of every agent with the `Organization` context provider, in the cached part of the prompt |
| `User` | 30 days of what one person's records show they work on | turns that person starts, for agents with the `User` context provider, in the runtime context |

## Facts, write-up, guidance

A profile is made in three layers, and only one of them is written by a model.

1. **Facts** are worked out by queries (`agentprofileports.FactsReader`,
   `agentprofilerepository/facts.go`): shipment volume and cancellations, share of moves
   covered by carriers, busiest customers and lanes (`pkg/lanequery`), most frequent pickups,
   fleet and driver mix, integrations, accounting connections, EDI partners, and the dispatch,
   billing and shipment settings. For a person: owned and entered shipments, the customers on
   them, the kinds of record they change (`audit_entries`), their Desk conversations and the
   tools agents ran for them, their favourite pages, and whether they are a driver.
   `agentprofileservice.OrganizationFacts` and `UserFacts` turn them into one line each.
2. **Guidance** is what people say about how to read the facts (`agent_profile_guidance`):
   "the Laredo lane is seasonal", "I only handle the Midwest accounts". A person types it in
   AI Control or on the Desk Memory page, or tells an agent, which writes it down with the core
   tool `guide_profile`. At most 50 active lines per profile.
3. **The write-up** (`agent_profiles.content`) is a structured completion
   (`FeatureAgentProfile`, `TaskAssistantChat`) from the facts, the guidance and the current
   version. Guidance outranks a fact it explains. The model may summarise and prioritise; it
   may not add a name, lane, setting or figure.

Nobody edits the write-up, a person included. Guidance is how it is steered, so every
version can be traced to the facts and lines it was written from.

## Stable on purpose

Facts are said in rounded words: a count under ten exactly, under a hundred to the nearest
five, above that to two figures (`intutils.RoundToStep`, `RoundSignificant`); a share to the
nearest five percent (`RoundedShare`). The **digest** (`agentprofileservice.Digest`) hashes
the facts, the guidance ids and text, and `PromptVersion`. A pass whose digest matches the
current version writes nothing and only moves `checked_at`, so a profile is rewritten when
the business moves, not every night; and the organization profile, which sits in the stable
part of the system prompt, does not break the provider's prompt cache from one day to the next.

A scheduled pass also skips a profile written or checked recently, without working out its
facts: the organization's within 20 hours, a person's within six days (`organizationInterval`,
`userInterval`). A person asking
(`Requested`) or guidance changing (`Guidance`) writes it at once, whatever the schedule;
`Requested` even writes a version whose digest has not changed. `Regenerate` refuses up front
where a pass would write nothing (plan, learning off, a person with no work on record), so the
person is told why rather than left waiting.

## Guards

- **Figures.** `readReply` runs `numberguard.CheckNumbers` over the write-up against every
  figure in the facts, the guidance and the organization's own name ("Route 66 Logistics").
  A write-up citing anything else fails the version.
- **A failed version never becomes current.** `Finish` records it with `status = Failed` and
  its reason; the current version stays what agents are told, and AI Control says the last
  attempt did not finish.
- **No provider is no failure.** When the organization has no provider for the task, the
  model activity answers `NoProvider` and nothing is recorded.
- **Gating.** No profile is written for an organization whose plan lacks
  `CapabilityAgentAutomation`, whose agent control has `LearningOff`, or that is in shadow
  mode. A person whose records show no work (`HasActivity`) gets no profile.
- **Taint.** `guide_profile` carries taint with a `TaintHold` that always applies: guidance
  written after a run read outside content waits for a person's approval, because every later
  prompt is written from it. It is held only on turns a person is in: unattended runs and
  delegated tasks do not hold it (`agentruntime/turn.go`).
- **Prompt framing.** Both profiles are fenced (`<organization_profile>`, `<person_profile>`),
  listed among the tags whose text is data, and introduced as background, never a source:
  agents still look figures up.
- **Retries.** `Prepare` chooses the version's id (`GenerationPlan.VersionID`), and `Publish`
  and `RecordFailure` return the row already kept under it, so a `Finish` activity retried
  after its transaction committed keeps one version and announces it once.
- **Retention.** Each profile keeps its newest 60 versions (`agentprofileports.KeptVersions`);
  `Prune` runs after every write and never removes the current version.

## Running it

`agentprofilejobs` on the system queue:

| Workflow | When |
|---|---|
| `AgentProfileSweepWorkflow` | daily at 09:17 UTC (`agent-profiles` schedule), fanning out one child per organization |
| `AgentProfileOrganizationWorkflow` | the organization's profile, then each person who started an agent turn in the last 14 days (`ActiveUserDays`, at most 500), four at a time. The organization's failure does not cost its people their profiles; an organization that writes none (learning off, no agents in the plan) lists nobody |
| `AgentProfileWorkflow` | one profile: prepare → model → finish → prune |
| `AgentProfileRequestWorkflow` | `agent-profile:<org>:<bu>:organization` or `…:user:<id>`, started or signalled by `agentprofilecontract.Scheduler`. A person asking writes at once; guidance waits for two quiet minutes (at most ten), so several lines cost one write. The request that starts the workflow arrives as its first signal too, and is folded in before the write rather than written twice |

Activities read the tenant from the request's `SubjectRef` (organization and business unit;
the person is `PersonID`, never `UserID`, so it is not mistaken for the actor).

## Reading and changing it

- GraphQL (`schema/agent_profile.graphqls`): `agentOrganizationProfile` and
  `agentOrganizationProfileHistory` need agent memory read;
  `addAgentOrganizationProfileGuidance` create; `retireAgentOrganizationProfileGuidance` and
  `regenerateAgentOrganizationProfile` update. `myAgentProfile`, `addMyAgentProfileGuidance`,
  `retireMyAgentProfileGuidance` and `regenerateMyAgentProfile` are the person's own and need
  assistant access. A line for another person's profile reads as not found.
- Guidance changes are audited under `ResourceAgentMemory`; every change is announced on the
  realtime resource `agent_profile` (a person's own only to them).
- AI Control → Memory shows the organization's profile, its facts, guidance and versions; the
  Desk Memory page shows the person's own.
- Usage lands under `AIUsageFeature.AgentProfile`.

## Changing it

- A new fact: add the read to `FactsReader` (tenant-scoped on the driving table and every
  join), the line to `OrganizationFacts` or `UserFacts`, rounded like the others, and a test.
  Every profile is rewritten on the next pass, because the digest changes.
- The writing instructions: change `organizationSystemPrompt` / `userSystemPrompt` and bump
  `PromptVersion`.
- The prompt sections: `buildOrganizationProfileSection` and `describeUserProfile` in
  `domain/agentdefinition/prompt.go`; bump `agentdefinition.PromptVersion` and regenerate the
  prompt goldens.
