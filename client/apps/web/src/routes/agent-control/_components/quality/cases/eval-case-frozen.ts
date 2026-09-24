export type FrozenFixture = {
  tool: string;
  argumentCount: number;
  failed: boolean;
};

export type CapturedFingerprint = {
  definitionVersion: number;
  promptVersion: string;
  tools: string[];
};

export type RedactedField = {
  tool: string;
  path: string;
  sensitivity: string;
};

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

export function readFixtures(value: unknown): FrozenFixture[] {
  if (!Array.isArray(value)) return [];

  return value.flatMap((item) =>
    isRecord(item) && typeof item.tool === "string"
      ? [
          {
            tool: item.tool,
            argumentCount: isRecord(item.args) ? Object.keys(item.args).length : 0,
            failed: item.failed === true,
          },
        ]
      : [],
  );
}

export function readFingerprint(value: unknown): CapturedFingerprint | null {
  if (!isRecord(value)) return null;

  return {
    definitionVersion: typeof value.definitionVersion === "number" ? value.definitionVersion : 0,
    promptVersion: typeof value.promptVersion === "string" ? value.promptVersion : "",
    tools: Array.isArray(value.tools)
      ? value.tools.filter((tool): tool is string => typeof tool === "string")
      : [],
  };
}

export function readRedaction(value: unknown): RedactedField[] {
  if (!isRecord(value) || !Array.isArray(value.fields)) return [];

  return value.fields.flatMap((item) =>
    isRecord(item) && typeof item.path === "string"
      ? [
          {
            tool: typeof item.tool === "string" ? item.tool : "",
            path: item.path,
            sensitivity: typeof item.sensitivity === "string" ? item.sensitivity : "",
          },
        ]
      : [],
  );
}
