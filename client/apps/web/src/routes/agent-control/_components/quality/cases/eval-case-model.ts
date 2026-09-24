import type {
  AgentEvalCaseStatus,
  AgentRunTrigger,
  CreateAgentEvalCaseInput,
  UpdateAgentEvalCaseInput,
} from "@trenova/graphql/generated/graphql";
import type { AgentEvalCaseDetail } from "@/lib/graphql/agent-eval-cases";
import { z } from "zod";

export const toleranceKinds = [
  "exact",
  "ci",
  "numeric",
  "dateWindow",
  "oneOf",
  "present",
  "ignore",
  "setEq",
] as const;

export type ToleranceKind = (typeof toleranceKinds)[number];

export const toolMatchModes = ["AnyOrder", "Ordered", "Subset"] as const;

export type ToolMatchMode = (typeof toolMatchModes)[number];

export type Tolerance = {
  kind: ToleranceKind;
  abs?: number;
  rel?: number;
  windowSeconds?: number;
  values?: unknown[];
};

export type ExpectedTool = {
  name: string;
  args?: Record<string, unknown>;
  rules?: Record<string, Tolerance>;
};

export type ExpectedProposal = {
  toolName: string;
  params?: Record<string, unknown>;
  rejected: boolean;
  rules?: Record<string, Tolerance>;
  sourceProposalId?: string;
};

export type Expected = {
  toolMode: ToolMatchMode;
  tools: ExpectedTool[];
  forbiddenTools: string[];
  proposals: ExpectedProposal[];
  expectRefusal: boolean;
  mustMention: string[];
  mustNotMention: string[];
};

export const TOLERANCE_LABEL: Record<ToleranceKind, string> = {
  exact: "Exactly",
  ci: "Same text, any case",
  numeric: "Number within",
  dateWindow: "Date within",
  oneOf: "One of",
  present: "Present",
  ignore: "Ignore",
  setEq: "Same set",
};

export const TOLERANCE_HELP: Record<ToleranceKind, string> = {
  exact: "The argument must be this value.",
  ci: "The same words, ignoring case and surrounding spaces.",
  numeric: "A number within an absolute amount or a share of the value.",
  dateWindow: "A date or time within this many hours of the value.",
  oneOf: "Any one of the listed values.",
  present: "Any non-empty value.",
  ignore: "Not checked.",
  setEq: "The same members in any order.",
};

export const TOOL_MODE_LABEL: Record<ToolMatchMode, string> = {
  AnyOrder: "Every tool, any order",
  Ordered: "Every tool, in this order",
  Subset: "Only these tools",
};

const CASE_TRANSITIONS: Record<AgentEvalCaseStatus, readonly AgentEvalCaseStatus[]> = {
  Candidate: ["Active", "Retired"],
  Active: ["Quarantined", "Retired"],
  Quarantined: ["Active", "Retired"],
  Retired: ["Active"],
};

export function nextStatuses(status: AgentEvalCaseStatus): readonly AgentEvalCaseStatus[] {
  return CASE_TRANSITIONS[status];
}

export function needsValue(kind: ToleranceKind): boolean {
  return (
    kind === "exact" ||
    kind === "ci" ||
    kind === "numeric" ||
    kind === "dateWindow" ||
    kind === "setEq"
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function strings(value: unknown): string[] {
  return Array.isArray(value)
    ? value.filter((item): item is string => typeof item === "string")
    : [];
}

function finite(value: unknown): number | undefined {
  return typeof value === "number" && Number.isFinite(value) ? value : undefined;
}

function readTolerance(value: unknown): Tolerance | null {
  if (!isRecord(value)) return null;
  const kind = value.kind;
  if (typeof kind !== "string" || !toleranceKinds.includes(kind as ToleranceKind)) return null;

  return {
    kind: kind as ToleranceKind,
    abs: finite(value.abs),
    rel: finite(value.rel),
    windowSeconds: finite(value.windowSeconds),
    values: Array.isArray(value.values) ? value.values : undefined,
  };
}

function readRules(value: unknown): Record<string, Tolerance> {
  if (!isRecord(value)) return {};
  const rules: Record<string, Tolerance> = {};
  for (const [key, raw] of Object.entries(value)) {
    const rule = readTolerance(raw);
    if (rule) rules[key] = rule;
  }

  return rules;
}

/** Reads what the server stores for a case, dropping anything it does not recognise. */
export function readExpected(value: unknown): Expected {
  const raw = isRecord(value) ? value : {};
  const mode = raw.toolMode;

  return {
    toolMode: toolMatchModes.includes(mode as ToolMatchMode) ? (mode as ToolMatchMode) : "AnyOrder",
    tools: Array.isArray(raw.tools)
      ? raw.tools.flatMap((item) =>
          isRecord(item) && typeof item.name === "string"
            ? [
                {
                  name: item.name,
                  args: isRecord(item.args) ? item.args : undefined,
                  rules: readRules(item.rules),
                },
              ]
            : [],
        )
      : [],
    forbiddenTools: strings(raw.forbiddenTools),
    proposals: Array.isArray(raw.proposals)
      ? raw.proposals.flatMap((item) =>
          isRecord(item) && typeof item.toolName === "string"
            ? [
                {
                  toolName: item.toolName,
                  params: isRecord(item.params) ? item.params : undefined,
                  rejected: item.rejected === true,
                  rules: readRules(item.rules),
                  sourceProposalId:
                    typeof item.sourceProposalId === "string" ? item.sourceProposalId : undefined,
                },
              ]
            : [],
        )
      : [],
    expectRefusal: raw.expectRefusal === true,
    mustMention: strings(raw.mustMention),
    mustNotMention: strings(raw.mustNotMention),
  };
}

/** What a written value means: JSON when it reads as JSON, the text itself otherwise. */
export function parseValue(text: string): unknown {
  const trimmed = text.trim();
  if (trimmed === "") return "";
  try {
    return JSON.parse(trimmed);
  } catch {
    return trimmed;
  }
}

export function formatValue(value: unknown): string {
  if (value === undefined || value === null) return "";
  if (typeof value === "string") return value;

  return JSON.stringify(value);
}

function parseList(text: string): unknown[] {
  const parsed = parseValue(text);
  if (Array.isArray(parsed)) return parsed;

  return text
    .split(",")
    .map((item) => item.trim())
    .filter((item) => item !== "");
}

const numberText = z.string().trim();

const ruleShape = z.object({
  key: z.string().trim().min(1, "Name the argument"),
  value: z.string(),
  kind: z.enum(toleranceKinds),
  abs: numberText,
  rel: numberText,
  windowHours: numberText,
  values: z.array(z.string()),
});

type RuleShape = z.infer<typeof ruleShape>;

function checkRule(rule: RuleShape, ctx: z.RefinementCtx, requireValue: boolean) {
  const numberAt = (path: "abs" | "rel" | "windowHours") => {
    const text = rule[path];
    if (text === "") return null;
    const parsed = Number(text);
    if (!Number.isFinite(parsed)) {
      ctx.addIssue({ code: "custom", path: [path], message: "Enter a number" });
      return null;
    }
    return parsed;
  };
  if (requireValue && needsValue(rule.kind) && rule.value.trim() === "") {
    ctx.addIssue({ code: "custom", path: ["value"], message: "Give the expected value" });
  }
  if (rule.kind === "numeric") {
    const abs = numberAt("abs");
    const rel = numberAt("rel");
    if (abs !== null && abs < 0) {
      ctx.addIssue({ code: "custom", path: ["abs"], message: "Cannot be negative" });
    }
    if (rel !== null && (rel < 0 || rel > 1)) {
      ctx.addIssue({ code: "custom", path: ["rel"], message: "A share between 0 and 1" });
    }
  }
  if (rule.kind === "dateWindow") {
    const hours = numberAt("windowHours");
    if (hours === null || hours <= 0) {
      ctx.addIssue({
        code: "custom",
        path: ["windowHours"],
        message: "A window longer than zero hours",
      });
    }
  }
  if (rule.kind === "oneOf" && rule.values.length === 0) {
    ctx.addIssue({ code: "custom", path: ["values"], message: "List at least one value" });
  }
}

const argumentRuleSchema = ruleShape.superRefine((rule, ctx) => checkRule(rule, ctx, true));

const proposalRuleSchema = ruleShape.superRefine((rule, ctx) => checkRule(rule, ctx, false));

export type ArgumentRuleValues = z.infer<typeof argumentRuleSchema>;

const expectedToolSchema = z.object({
  name: z.string().trim().min(1, "Name the tool"),
  args: z.array(argumentRuleSchema),
});

const expectedProposalSchema = z.object({
  toolName: z.string().trim().min(1, "Name the tool"),
  rejected: z.boolean(),
  params: z.string(),
  sourceProposalId: z.string(),
  rules: z.array(proposalRuleSchema),
});

export const evalCaseFormSchema = z
  .object({
    agentDefinitionId: z.string(),
    title: z.string().trim().max(200, "Keep the title to 200 characters"),
    trigger: z.enum(["Chat", "Continuous", "Event", "Manual", "Scheduled"]),
    input: z
      .string()
      .trim()
      .min(1, "Write the question the agent is asked")
      .max(20000, "Keep the question to 20,000 characters"),
    rubric: z.string().trim().max(4000, "Keep the rubric to 4,000 characters"),
    heldTools: z.array(z.string()),
    toolMode: z.enum(toolMatchModes),
    tools: z.array(expectedToolSchema),
    forbiddenTools: z.array(z.string()),
    proposals: z.array(expectedProposalSchema),
    expectRefusal: z.boolean(),
    mustMention: z.array(z.string()),
    mustNotMention: z.array(z.string()),
    expiresAt: z.number().int().positive().nullable(),
    version: z.number().int().nonnegative(),
  })
  .superRefine((values, ctx) => {
    values.tools.forEach((tool, index) => {
      if (values.forbiddenTools.includes(tool.name.trim())) {
        ctx.addIssue({
          code: "custom",
          path: ["tools", index, "name"],
          message: "A tool cannot be both expected and forbidden",
        });
      }
    });
    values.proposals.forEach((proposal, index) => {
      if (proposal.rejected) return;
      const params = parseValue(proposal.params);
      if (typeof params !== "object" || params === null || Array.isArray(params)) {
        ctx.addIssue({
          code: "custom",
          path: ["proposals", index, "params"],
          message: "The approved parameters are a JSON object",
        });
      }
    });
    if (values.expectRefusal && values.tools.length > 0) {
      ctx.addIssue({
        code: "custom",
        path: ["expectRefusal"],
        message: "A case that expects a refusal cannot also expect tool calls",
      });
    }
    const forbidden = values.mustNotMention.map((phrase) => phrase.toLowerCase());
    if (values.mustMention.some((phrase) => forbidden.includes(phrase.toLowerCase()))) {
      ctx.addIssue({
        code: "custom",
        path: ["mustNotMention"],
        message: "A phrase cannot be both required and forbidden",
      });
    }
  });

export type EvalCaseFormValues = z.infer<typeof evalCaseFormSchema>;

export const blankArgumentRule: ArgumentRuleValues = {
  key: "",
  value: "",
  kind: "exact",
  abs: "",
  rel: "",
  windowHours: "",
  values: [],
};

export const evalCaseFormDefaults: EvalCaseFormValues = {
  agentDefinitionId: "",
  title: "",
  trigger: "Chat",
  input: "",
  rubric: "",
  heldTools: [],
  toolMode: "AnyOrder",
  tools: [],
  forbiddenTools: [],
  proposals: [],
  expectRefusal: false,
  mustMention: [],
  mustNotMention: [],
  expiresAt: null,
  version: 0,
};

function numberInput(value: number | undefined): string {
  return value === undefined ? "" : String(value);
}

function argumentRules(
  values: Record<string, unknown> | undefined,
  rules: Record<string, Tolerance> | undefined,
): ArgumentRuleValues[] {
  const keys = new Set([...Object.keys(values ?? {}), ...Object.keys(rules ?? {})]);

  return [...keys].sort().map((key) => {
    const rule = rules?.[key] ?? { kind: "exact" as const };

    return {
      key,
      value: formatValue(values?.[key]),
      kind: rule.kind,
      abs: numberInput(rule.abs),
      rel: numberInput(rule.rel),
      windowHours: rule.windowSeconds === undefined ? "" : String(rule.windowSeconds / 3600),
      values: (rule.values ?? []).map(formatValue),
    };
  });
}

/** The editor's values for a stored case. */
export function toFormValues(evalCase: AgentEvalCaseDetail): EvalCaseFormValues {
  const expected = readExpected(evalCase.expected);

  return {
    agentDefinitionId: evalCase.agentDefinitionId,
    title: evalCase.title,
    trigger: evalCase.trigger,
    input: evalCase.input,
    rubric: evalCase.rubric,
    heldTools: evalCase.heldTools,
    toolMode: expected.toolMode,
    tools: expected.tools.map((tool) => ({
      name: tool.name,
      args: argumentRules(tool.args, tool.rules),
    })),
    forbiddenTools: expected.forbiddenTools,
    proposals: expected.proposals.map((proposal) => ({
      toolName: proposal.toolName,
      rejected: proposal.rejected,
      params: proposal.params ? JSON.stringify(proposal.params, null, 2) : "",
      sourceProposalId: proposal.sourceProposalId ?? "",
      rules: argumentRules(undefined, proposal.rules).filter((rule) => rule.kind !== "exact"),
    })),
    expectRefusal: expected.expectRefusal,
    mustMention: expected.mustMention,
    mustNotMention: expected.mustNotMention,
    expiresAt: evalCase.expiresAt ?? null,
    version: evalCase.version,
  };
}

function toTolerance(rule: ArgumentRuleValues): Tolerance {
  const tolerance: Tolerance = { kind: rule.kind };
  if (rule.kind === "numeric") {
    if (rule.abs !== "") tolerance.abs = Number(rule.abs);
    if (rule.rel !== "") tolerance.rel = Number(rule.rel);
  }
  if (rule.kind === "dateWindow") {
    tolerance.windowSeconds = Math.round(Number(rule.windowHours) * 3600);
  }
  if (rule.kind === "oneOf") {
    tolerance.values = rule.values.map(parseValue);
  }

  return tolerance;
}

function toArgs(
  rules: ArgumentRuleValues[],
  withValues: boolean,
): {
  args: Record<string, unknown>;
  rules: Record<string, Tolerance>;
} {
  const args: Record<string, unknown> = {};
  const tolerances: Record<string, Tolerance> = {};
  for (const rule of rules) {
    const key = rule.key.trim();
    if (withValues && needsValue(rule.kind)) {
      args[key] = rule.kind === "setEq" ? parseList(rule.value) : parseValue(rule.value);
    }
    if (rule.kind !== "exact") {
      tolerances[key] = toTolerance(rule);
    }
  }

  return { args, rules: tolerances };
}

/** What the scorer reads, from the editor's values. */
export function toExpected(values: EvalCaseFormValues): Expected {
  return {
    toolMode: values.toolMode,
    tools: values.tools.map((tool) => {
      const { args, rules } = toArgs(tool.args, true);
      return { name: tool.name.trim(), args, rules };
    }),
    forbiddenTools: values.forbiddenTools,
    proposals: values.proposals.map((proposal) => {
      const params = parseValue(proposal.params);
      const { rules } = toArgs(proposal.rules, false);
      return {
        toolName: proposal.toolName.trim(),
        params:
          typeof params === "object" && params !== null && !Array.isArray(params)
            ? (params as Record<string, unknown>)
            : undefined,
        rejected: proposal.rejected,
        rules,
        sourceProposalId: proposal.sourceProposalId || undefined,
      };
    }),
    expectRefusal: values.expectRefusal,
    mustMention: values.mustMention,
    mustNotMention: values.mustNotMention,
  };
}

export function toUpdateInput(
  values: EvalCaseFormValues,
  previousExpiry: number | null,
): UpdateAgentEvalCaseInput {
  const input: UpdateAgentEvalCaseInput = {
    version: values.version,
    title: values.title,
    input: values.input,
    rubric: values.rubric,
    heldTools: values.heldTools,
    expected: toExpected(values),
  };
  if (values.expiresAt !== previousExpiry) {
    input.expiresAt = values.expiresAt;
  }

  return input;
}

export function toCuratedInput(values: EvalCaseFormValues): CreateAgentEvalCaseInput {
  return {
    curated: {
      agentDefinitionId: values.agentDefinitionId,
      title: values.title,
      trigger: values.trigger satisfies AgentRunTrigger,
      input: values.input,
      rubric: values.rubric,
      heldTools: values.heldTools.length > 0 ? values.heldTools : null,
      expected: toExpected(values),
      expiresAt: values.expiresAt,
    },
  };
}

/** A one-line account of what a case checks, for the list. */
export function describeExpected(
  expected: Expected,
  t: (text: string, ...args: (string | number)[]) => string,
): string {
  const parts: string[] = [];
  if (expected.expectRefusal) parts.push(t("Expects a refusal"));
  if (expected.tools.length > 0) {
    parts.push(t("{0, plural, one {# tool} other {# tools}}", expected.tools.length));
  }
  if (expected.proposals.length > 0) {
    parts.push(t("{0, plural, one {# proposal} other {# proposals}}", expected.proposals.length));
  }
  if (expected.forbiddenTools.length > 0) {
    parts.push(t("{0} forbidden", expected.forbiddenTools.length));
  }
  const mentions = expected.mustMention.length + expected.mustNotMention.length;
  if (mentions > 0) {
    parts.push(t("{0, plural, one {# mention rule} other {# mention rules}}", mentions));
  }

  return parts.length === 0 ? t("Nothing expected yet") : parts.join(" · ");
}
