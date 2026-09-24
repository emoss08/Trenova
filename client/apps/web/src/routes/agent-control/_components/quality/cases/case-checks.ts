export type CaseCheck = {
  name: string;
  kind: "hard" | "soft";
  applies: boolean;
  passed: boolean;
  score: number;
  weight: number;
  detail: string;
  findings: string[];
};

export type CaseChecks = {
  checks: CaseCheck[];
  refused: boolean;
  hardFailure: boolean;
  deterministic: number;
  final: number;
  passed: boolean;
};

export const CHECK_LABEL: Record<string, string> = {
  heldTools: "Only tools it held",
  forbiddenTools: "No forbidden tool",
  refusal: "Refused when it should",
  taintedEgress: "Nothing outbound ran on its own after untrusted input",
  toolChoice: "Tool choice",
  arguments: "Arguments",
  proposals: "Proposals",
  factGuard: "Figures it can back up",
  mentions: "What the reply mentions",
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function number(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

/** Reads the checks the server stores on a case replay, or null when there are none. */
export function readCaseChecks(value: unknown): CaseChecks | null {
  if (!isRecord(value) || !Array.isArray(value.checks)) {
    return null;
  }

  return {
    checks: value.checks.flatMap((item) =>
      isRecord(item) && typeof item.name === "string"
        ? [
            {
              name: item.name,
              kind: item.kind === "hard" ? "hard" : "soft",
              applies: item.applies === true,
              passed: item.passed === true,
              score: number(item.score),
              weight: number(item.weight),
              detail: typeof item.detail === "string" ? item.detail : "",
              findings: Array.isArray(item.findings)
                ? item.findings.filter((finding): finding is string => typeof finding === "string")
                : [],
            } satisfies CaseCheck,
          ]
        : [],
    ),
    refused: value.refused === true,
    hardFailure: value.hardFailure === true,
    deterministic: number(value.deterministic),
    final: number(value.final),
    passed: value.passed === true,
  };
}
