// lading.ts types the `lading.*` operations of edi.wasm the browser calls
// (docs/engineering/lading.md, "Module operations"). Each is synchronous on a loaded runtime;
// every position and range in and out is UTF-16 code units, which is what CodeMirror counts. A
// refused request throws ModuleRequestError. Nothing here lexes, parses, checks or runs Lading:
// the module answers and the studio draws the answer.

import type { CatalogMessage } from "@trenova/shared/i18n/message";
import type { EdiRuntime } from "./runtime";

export type LadingRange = { from: number; to: number };

export type LadingFile = { path: string; source: string };
export type LadingSample = { path: string; content: string };
export type LadingProjectInput = { files: LadingFile[]; samples: LadingSample[] };

export type LadingCustomField = { name: string; type: string; options?: string[] };

/** What a project is checked against: the organisation and its shipment custom fields. */
export type LadingContext = {
  org: { name: string; scac: string; zone: string };
  customFields?: LadingCustomField[];
};

export type LadingChange = { range: LadingRange; insert: string };
export type LadingFix = { label: CatalogMessage; changes: LadingChange[] };
export type LadingSeverity = "error" | "warning" | "info";

export type LadingDiagnostic = {
  range: LadingRange;
  severity: LadingSeverity;
  code: string;
  message: CatalogMessage;
  fixes: LadingFix[];
  related: { range: LadingRange; message: CatalogMessage }[];
};

export type LadingSymbol = {
  name: string;
  kind: string;
  range: LadingRange;
  selection: LadingRange;
  children?: LadingSymbol[];
};

export type LadingCheckedFile = {
  path: string;
  /** `partner`, `guide`, `map`, `rules`, `flow`, `library`, or `` when its header is missing. */
  kind: string;
  name: string | null;
  diagnostics: LadingDiagnostic[];
  outline: LadingSymbol[];
  folds: { range: LadingRange }[];
};

export type LadingSetting = {
  name: string;
  type: string;
  default: string | null;
  range: [string, string] | null;
  doc: string | null;
  value: unknown;
};

export type LadingParty = { qualifier: string; id: string };

export type LadingInterchange = {
  them: LadingParty | null;
  us: LadingParty | null;
  group: { theirs: string; ours: string } | null;
  version: string | null;
  usage: string | null;
  separators: string[];
};

export type LadingPartner = {
  name: string;
  label: string;
  zone: string;
  currency: string;
  settings: LadingSetting[];
  interchange: LadingInterchange | null;
};

export type LadingMap = {
  name: string;
  path: string;
  direction: "inbound" | "outbound";
  from: string;
  to: string;
  document: string;
  set: string | null;
  source: "x12" | "csv" | "json" | "document";
};

export type LadingClause = {
  kind: "receive" | "respond" | "send" | "expect";
  set: string;
  binder?: string;
  map?: string | null;
  rules?: string | null;
  acknowledge?: string;
  events?: string[];
  until?: string | null;
  batched?: boolean;
};

export type LadingFlow = { name: string; path: string; clauses: LadingClause[] };

export type LadingGuide = {
  name: string;
  path: string | null;
  set: string;
  version: string;
  base: string | null;
};

/** What `lading.check` answers: the manifest of a project. */
export type LadingCheck = {
  files: LadingCheckedFile[];
  partner: LadingPartner | null;
  maps: LadingMap[];
  flows: LadingFlow[];
  rules: { name: string; path: string; map: string }[];
  libraries: string[];
  guides: LadingGuide[];
  errors: number;
  warnings: number;
};

export type LadingToken = { range: LadingRange; kind: string };

export type LadingDoc = string | CatalogMessage;

export type LadingCompletionItem = {
  label: string;
  kind: string;
  detail: string;
  doc?: LadingDoc;
  insert: string;
  snippet: boolean;
  sort: string;
};

export type LadingRunValue = { path: string; value: string; sources?: string[]; steps?: string[] };

export type LadingHover = {
  range: LadingRange;
  contents: {
    signature: string;
    type: string;
    doc: LadingDoc | null;
    value: LadingRunValue | null;
  };
};

export type LadingLocation = { path: string; range: LadingRange };
export type LadingEdit = { path: string; range: LadingRange; insert: string };

export type LadingRunInput = {
  x12?: string;
  csv?: string;
  json?: unknown;
  document?: unknown;
};

export type LadingLookupAnswer = {
  kind: string;
  code: string;
  found?: { id: string; label: string; zone?: string };
};

export type LadingSource = { path: string; segment?: string; position?: number; element?: number };
export type LadingLineage = { path: string; sources: LadingSource[]; steps: string[]; value: string };

export type LadingOutcome = {
  kind: "reject" | "decline" | "hold" | "warn";
  message: string;
  rule?: string;
  path?: string;
  element?: LadingSource;
  code?: string;
};

export type LadingFinding = {
  kind: string;
  severity: LadingSeverity;
  level: string;
  code?: string;
  segmentId?: string;
  element?: number;
  message: CatalogMessage;
};

export type LadingTransactionResult = {
  index: number;
  control?: string;
  status: "accepted" | "rejected";
  findings: LadingFinding[];
  document: unknown;
  outcomes: LadingOutcome[];
  fixes: { rule: string; path: string; before: unknown; after: unknown }[];
  lineage: LadingLineage[];
  unread: string[];
};

export type LadingWriteResult = {
  text: string;
  transaction: string;
  document: unknown;
  findings: { severity: LadingSeverity; code: string; message: CatalogMessage }[];
  validation: LadingFinding[];
  outcomes: LadingOutcome[];
  lineage: LadingLineage[];
};

export type LadingNeed = { kind: string; code: string };
export type LadingUnmapped = { kind: string; code: string; path: string };

export type LadingRunResult = {
  transactions: LadingTransactionResult[];
  output?: LadingWriteResult;
  needs: LadingNeed[];
  unmapped: LadingUnmapped[];
  error?: CatalogMessage;
  steps: number;
};

export type LadingRunRequest = {
  map: string;
  input: LadingRunInput;
  settings?: Record<string, unknown>;
  lookups?: LadingLookupAnswer[];
  now?: string;
  rules?: boolean;
};

export type LadingTestResult = {
  path: string;
  name: string;
  kind: "test" | "property";
  range: LadingRange;
  passed: boolean;
  cases: number;
  failures: { range: LadingRange; message: CatalogMessage }[];
  counterexample: { name: string; value: string }[] | null;
};

export type LadingReadiness = {
  errors: number;
  failing: number;
  tests: number;
  uncoveredFlowClauses: number;
  ready: boolean;
};

export type LadingTestReport = {
  tests: LadingTestResult[];
  coverage: { path: string; covered: LadingRange[]; uncovered: LadingRange[] }[];
  unread: {
    map: string;
    element: string;
    qualifier?: string;
    samples: number;
    total: number;
    example: string;
  }[];
  readiness: LadingReadiness;
};

export type LadingStarterRequest = {
  name: string;
  label?: string;
  role: "carrier" | "broker";
  them?: [string, string];
  us?: [string, string];
  zone?: string;
  group?: [string, string];
  version?: string;
  separators?: string[];
  usage?: "test" | "production";
  acknowledge?: string;
  ackWithin?: number;
  sets?: string[];
};

export type LadingCatalogField = {
  name: string;
  type: string;
  required: boolean;
  doc: CatalogMessage;
  maxLength: number | null;
};

export type LadingCatalogRecord = {
  name: string;
  doc: CatalogMessage;
  set: string | null;
  fields: LadingCatalogField[];
};

export type LadingCatalog = {
  documents: LadingCatalogRecord[];
  records: LadingCatalogRecord[];
  enums: { name: string; doc: CatalogMessage; variants: { name: string; doc: CatalogMessage }[] }[];
  lookups: { name: string; entity: string; doc: CatalogMessage; zoned: boolean }[];
  events: { name: string; document: string; doc: CatalogMessage }[];
  states: { name: string; doc: CatalogMessage }[];
  functions: { name: string; signature: string; doc: CatalogMessage }[];
};

type ProjectRequest = { project: LadingProjectInput; context: LadingContext };

function head(request: ProjectRequest): ProjectRequest {
  return { project: request.project, context: request.context };
}

export function ladingCheck(edi: EdiRuntime, request: ProjectRequest): LadingCheck {
  return edi.call("lading.check", head(request)) as LadingCheck;
}

export function ladingTokens(edi: EdiRuntime, source: string): LadingToken[] {
  return (edi.call("lading.tokens", { source }) as { tokens: LadingToken[] }).tokens;
}

/** The file formatted, or the reason it is left as written. */
export function ladingFormat(
  edi: EdiRuntime,
  source: string,
): { text: string } | { error: CatalogMessage } {
  return edi.call("lading.format", { source }) as { text: string } | { error: CatalogMessage };
}

type AtRequest = ProjectRequest & { path: string; offset: number };

export function ladingComplete(
  edi: EdiRuntime,
  request: AtRequest,
): { range: LadingRange; items: LadingCompletionItem[] } {
  return edi.call("lading.complete", {
    ...head(request),
    path: request.path,
    offset: request.offset,
  }) as { range: LadingRange; items: LadingCompletionItem[] };
}

export function ladingHover(
  edi: EdiRuntime,
  request: AtRequest & { values?: LadingRunValue[] },
): LadingHover | null {
  return edi.call("lading.hover", {
    ...head(request),
    path: request.path,
    offset: request.offset,
    values: request.values ?? [],
  }) as LadingHover | null;
}

export type LadingSignatureHelp = {
  signatures: { label: string; doc: LadingDoc | null; parameters: { label: [number, number] }[] }[];
  active: number | null;
  activeParameter: number | null;
};

export function ladingSignature(edi: EdiRuntime, request: AtRequest): LadingSignatureHelp {
  return edi.call("lading.signature", {
    ...head(request),
    path: request.path,
    offset: request.offset,
  }) as LadingSignatureHelp;
}

export function ladingDefinition(edi: EdiRuntime, request: AtRequest): LadingLocation[] {
  return (
    edi.call("lading.definition", {
      ...head(request),
      path: request.path,
      offset: request.offset,
    }) as { targets: LadingLocation[] }
  ).targets;
}

export function ladingReferences(
  edi: EdiRuntime,
  request: AtRequest & { includeDeclaration?: boolean },
): (LadingLocation & { kind: string })[] {
  return (
    edi.call("lading.references", {
      ...head(request),
      path: request.path,
      offset: request.offset,
      includeDeclaration: request.includeDeclaration ?? true,
    }) as { references: (LadingLocation & { kind: string })[] }
  ).references;
}

export type LadingRenameAnswer =
  | { edits: LadingEdit[] }
  | { range: LadingRange; placeholder: string }
  | { error: CatalogMessage };

/** Without `newName` it prepares (the range and placeholder); with it, the edits. */
export function ladingRename(
  edi: EdiRuntime,
  request: AtRequest & { newName?: string },
): LadingRenameAnswer {
  return edi.call("lading.rename", {
    ...head(request),
    path: request.path,
    offset: request.offset,
    newName: request.newName,
  }) as LadingRenameAnswer;
}

export type LadingInlay = { offset: number; label: string; kind: string };

export function ladingInlays(
  edi: EdiRuntime,
  request: ProjectRequest & { path: string; range: LadingRange; values?: LadingRunValue[] },
): LadingInlay[] {
  return (
    edi.call("lading.inlays", {
      ...head(request),
      path: request.path,
      range: request.range,
      values: request.values ?? [],
    }) as { hints: LadingInlay[] }
  ).hints;
}

export type LadingAction = { id: string; title: CatalogMessage; edits: LadingEdit[] };

export function ladingActions(
  edi: EdiRuntime,
  request: ProjectRequest & { path: string; range: LadingRange },
): LadingAction[] {
  return (
    edi.call("lading.actions", {
      ...head(request),
      path: request.path,
      range: request.range,
    }) as { actions: LadingAction[] }
  ).actions;
}

export function ladingRun(
  edi: EdiRuntime,
  request: ProjectRequest & LadingRunRequest,
): LadingRunResult {
  const { project, context, ...run } = request;
  return edi.call("lading.run", { project, context, rules: true, ...run }) as LadingRunResult;
}

export function ladingTest(edi: EdiRuntime, request: ProjectRequest): LadingTestReport {
  return edi.call("lading.test", head(request)) as LadingTestReport;
}

export function ladingStarter(edi: EdiRuntime, request: LadingStarterRequest): LadingProjectInput {
  return edi.call("lading.starter", request) as LadingProjectInput;
}

const catalogs = new WeakMap<EdiRuntime, LadingCatalog>();

/** The documents, records, enums, events, lookups and functions, memoised per runtime. */
export function ladingCatalog(edi: EdiRuntime): LadingCatalog {
  const cached = catalogs.get(edi);
  if (cached) return cached;
  const catalog = edi.call("lading.catalog", {}) as LadingCatalog;
  catalogs.set(edi, catalog);
  return catalog;
}

/** A doc as text: the author's words as written, a message translated. */
export function ladingDocText(
  doc: LadingDoc | null | undefined,
  translate: (message: CatalogMessage) => string,
): string {
  if (doc === null || doc === undefined) return "";
  return typeof doc === "string" ? doc : translate(doc);
}
