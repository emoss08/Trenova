import type { AssistantArtifact } from "@/types/assistant";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { z } from "zod";
import type { CardCheckState } from "../../record-card/record-card";

/*
 * The draft edi_draft_connection published, as the recorder stored it. Every
 * reader here refuses to guess: a value that is not what the draft promises
 * reads as absent, never as a crash in the pane.
 */

const text = z.string().catch("");
const count = z.number().catch(0);
const flag = z.boolean().catch(false);

function list<T extends z.ZodType>(item: T) {
  return z
    .array(z.unknown())
    .catch([])
    .transform((values) =>
      values.flatMap((value) => {
        const parsed = item.safeParse(value);
        return parsed.success ? [parsed.data as z.output<T>] : [];
      }),
    );
}

const changeSchema = z.enum(["Add", "Change", "Same", "None"]).catch("None");
const valueStatusSchema = z
  .enum(["Accepted", "Defaulted", "Kept", "Held", "Unverified", "Invalid", "Missing"])
  .catch("Unverified");
const sourceKindSchema = z
  .enum(["Document", "Sample", "Person", "Configuration", "Default", "Agent"])
  .catch("Agent");

const sourceSchema = z
  .object({
    kind: sourceKindSchema,
    page: count,
    quote: text,
    section: text,
    note: text,
  })
  .catch({ kind: "Agent", page: 0, quote: "", section: "", note: "" });

const candidateSchema = z.object({
  value: text,
  confidence: count,
  source: sourceSchema,
  problem: text,
});

const valueSchema = z.object({
  field: text,
  label: text,
  value: text,
  current: text,
  change: changeSchema,
  status: valueStatusSchema,
  required: flag,
  confidence: count,
  source: sourceSchema,
  candidates: list(candidateSchema),
  reason: text,
});

const partnerCandidateSchema = z.object({
  partnerId: text,
  code: text,
  name: text,
  reasons: list(z.string()),
});

const partnerSchema = z
  .object({
    match: z.enum(["New", "Existing", "Ambiguous"]).catch("New"),
    partnerId: text,
    version: count,
    status: text,
    reasons: list(z.string()),
    candidates: list(partnerCandidateSchema),
    customerId: text,
    change: changeSchema,
    fields: list(valueSchema),
  })
  .catch({
    match: "New",
    partnerId: "",
    version: 0,
    status: "",
    reasons: [],
    candidates: [],
    customerId: "",
    change: "None",
    fields: [],
  });

const setDirectionSchema = z.object({ transactionSet: text, direction: text });

const connectionSchema = z
  .object({ method: text, inbound: flag, outbound: flag, sets: list(setDirectionSchema) })
  .catch({ method: "", inbound: false, outbound: false, sets: [] });

const credentialSchema = z.object({
  keys: list(z.string()),
  label: text,
  kind: z.enum(["Secret", "Setting"]).catch("Secret"),
  required: flag,
  missing: flag,
});

const transportSchema = z
  .object({
    profileId: text,
    version: count,
    name: text,
    method: text,
    status: text,
    change: changeSchema,
    fields: list(valueSchema),
    credentials: list(credentialSchema),
  })
  .nullable()
  .catch(null);

const requirementSchema = z.object({
  segmentId: text,
  elementPosition: count,
  usage: text,
  codes: list(z.string()),
  problem: text,
  source: sourceSchema,
});

const controlNumberSchema = z.object({
  kind: text,
  start: count,
  current: count,
  change: changeSchema,
  source: sourceSchema,
  problem: text,
});

const setSchema = z.object({
  transactionSet: text,
  direction: text,
  change: changeSchema,
  fields: list(valueSchema),
  requirements: count,
  unmet: list(requirementSchema),
  unverified: list(requirementSchema),
  controlNumbers: list(controlNumberSchema),
});

const projectFileSchema = z.object({ path: text, kind: text, change: changeSchema });

const projectDiagnosticSchema = z.object({
  path: text,
  severity: text,
  code: text,
  message: text,
});

const projectSchema = z
  .object({
    projectId: text,
    role: text,
    change: changeSchema,
    files: list(projectFileSchema),
    errors: count,
    warnings: count,
    diagnostics: list(projectDiagnosticSchema),
  })
  .nullable()
  .catch(null);

const mappingSchema = z.object({
  entityType: text,
  sourceId: text,
  sourceLabel: text,
  targetId: text,
  targetLabel: text,
  currentTargetId: text,
  change: changeSchema,
  status: valueStatusSchema,
  confidence: count,
  source: sourceSchema,
  reason: text,
});

const sampleCheckSchema = z.object({
  field: text,
  label: text,
  expected: text,
  found: text,
  matches: flag,
});

const unmappedSchema = z.object({ entityType: text, sourceId: text, sourceLabel: text });

const sampleSchema = z.object({
  page: count,
  transactionSet: text,
  direction: text,
  parsed: flag,
  error: text,
  overview: z
    .object({
      x12Version: text,
      summary: z
        .object({ segmentCount: count, errorCount: count, warningCount: count })
        .nullable()
        .catch(null),
    })
    .nullable()
    .catch(null),
  checks: list(sampleCheckSchema),
  unmapped: list(unmappedSchema),
});

const gapSchema = z.object({
  kind: z.enum(["Unsupported", "Ambiguous"]).catch("Ambiguous"),
  summary: text,
  source: sourceSchema,
  problem: text,
});

const failureSchema = z.object({ record: text, field: text, message: text });

const keptSchema = z.object({ record: text, label: text });

const draftSchema = z.object({
  documentId: text,
  fileName: text,
  pageCount: count,
  documentDigest: text,
  partner: partnerSchema,
  connection: connectionSchema,
  interchange: list(valueSchema),
  transport: transportSchema,
  sets: list(setSchema),
  project: projectSchema,
  mappings: list(mappingSchema),
  mappingCount: count,
  truncated: flag,
  kept: list(keptSchema),
  samples: list(sampleSchema),
  gaps: list(gapSchema),
  decisions: list(valueSchema),
  failures: list(failureSchema),
  blockers: list(z.string()),
  changes: count,
  ready: flag,
});

export type DraftChange = z.output<typeof changeSchema>;
export type DraftValueStatus = z.output<typeof valueStatusSchema>;
export type DraftSource = z.output<typeof sourceSchema>;
export type DraftCandidate = z.output<typeof candidateSchema>;
export type DraftValue = z.output<typeof valueSchema>;
export type DraftCredential = z.output<typeof credentialSchema>;
export type DraftRequirement = z.output<typeof requirementSchema>;
export type DraftSet = z.output<typeof setSchema>;
export type DraftProject = NonNullable<z.output<typeof projectSchema>>;
export type DraftControlNumber = z.output<typeof controlNumberSchema>;
export type DraftMapping = z.output<typeof mappingSchema>;
export type DraftSample = z.output<typeof sampleSchema>;
export type DraftGap = z.output<typeof gapSchema>;
export type DraftFailure = z.output<typeof failureSchema>;
export type ConnectionDraft = z.output<typeof draftSchema>;

/** A connection draft artifact's payload, read defensively. */
export function connectionDraftFrom(artifact: Pick<AssistantArtifact, "payload">): ConnectionDraft {
  const parsed = draftSchema.safeParse(artifact.payload ?? {});
  return parsed.success ? parsed.data : draftSchema.parse({});
}

/** What the person still has to do, and what the guide asks that the draft does not meet. */
export type DraftOutline = {
  decisions: number;
  /** Errors the drafted Lading project still has, to fix in the studio. */
  projectErrors: number;
  secretsToEnter: string[];
  unmet: number;
  gaps: number;
  failures: number;
};

export function draftOutline(draft: ConnectionDraft): DraftOutline {
  return {
    decisions: draft.decisions.length,
    projectErrors: draft.project?.errors ?? 0,
    secretsToEnter: (draft.transport?.credentials ?? [])
      .filter((credential) => credential.required && credential.missing)
      .map((credential) => credential.label),
    unmet: draft.sets.reduce((total, set) => total + set.unmet.length, 0),
    gaps: draft.gaps.length,
    failures: draft.failures.length,
  };
}

const PARTNER_NAME_FIELD = "partner.name";

/** The partner's name as the draft has it, else the guide's file name. */
export function partnerNameOf(draft: ConnectionDraft): string {
  const name = draft.partner.fields.find((field) => field.field === PARTNER_NAME_FIELD);
  return name?.value || name?.current || draft.fileName;
}

/** Where a value came from, in a few words: the page, the sample, the person, the setup. */
export function sourceLabel(source: DraftSource, t: TranslateFn): string {
  switch (source.kind) {
    case "Document":
      return source.page > 0 ? t("Page {0}", source.page) : t("The guide");
    case "Sample":
      return source.page > 0 ? t("Sample, page {0}", source.page) : t("Sample");
    case "Person":
      return t("Your answer");
    case "Configuration":
      return t("Current setup");
    case "Default":
      return t("Default");
    case "Agent":
      return t("Agent");
  }
}

/** Whether a value holds, is worth knowing, needs a look, or cannot be saved. */
export function valueState(status: DraftValueStatus): CardCheckState {
  switch (status) {
    case "Accepted":
      return "ok";
    case "Defaulted":
    case "Kept":
      return "info";
    case "Held":
    case "Unverified":
      return "warn";
    case "Invalid":
    case "Missing":
      return "bad";
  }
}

/** The person's choice for a held value, put to the agent as a request to draft again. */
export function resolutionQuestion(
  draft: ConnectionDraft,
  decision: DraftValue,
  candidate: DraftCandidate,
  t: TranslateFn,
): string {
  const file = draft.fileName || t("the guide");
  const label = decision.label || decision.field;
  return candidate.source.page > 0
    ? t(
        'For the EDI connection drafted from {0}, use "{1}" for {2} (page {3}) and draft it again.',
        file,
        candidate.value,
        label,
        candidate.source.page,
      )
    : t(
        'For the EDI connection drafted from {0}, use "{1}" for {2} and draft it again.',
        file,
        candidate.value,
        label,
      );
}

/** The request to propose saving a ready draft, naming the digest the person reviewed. */
export function proposeQuestion(draft: ConnectionDraft, t: TranslateFn): string {
  return t(
    "Propose saving the EDI connection drafted from {0}, with document digest {1}.",
    draft.fileName || t("the guide"),
    draft.documentDigest,
  );
}
