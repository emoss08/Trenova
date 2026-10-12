const CUSTOM_PREFIX = "custom:";
const KEY_ALPHABET = "abcdefghijklmnopqrstuvwxyz0123456789";
const KEY_LENGTH = 12;

/** Whether a checklist step is one the organization added. */
export function isCustomStep(key: string): boolean {
  return key.startsWith(CUSTOM_PREFIX);
}

/**
 * A key for a step the organization adds: random, lower case, and in the
 * shape the server accepts (`custom:` and 6–32 letters or digits), so it
 * stays the step's identity, and its ticks', across saves and renames.
 */
export function newCustomStepKey(): string {
  const bytes = new Uint8Array(KEY_LENGTH);
  crypto.getRandomValues(bytes);
  let key = CUSTOM_PREFIX;
  for (const byte of bytes) {
    key += KEY_ALPHABET[byte % KEY_ALPHABET.length];
  }

  return key;
}

type StepLike = {
  key: string;
  mode: string;
  custom?: {
    label: string;
    check: string;
    documentTypeId?: string;
    stepLabel?: string;
    prompt?: string;
  } | null;
};

function stepSignature(step: StepLike): string {
  const custom = step.custom;
  return [
    step.mode,
    custom
      ? [
          custom.label,
          custom.check,
          custom.documentTypeId ?? "",
          custom.stepLabel ?? "",
          custom.prompt ?? "",
        ].join("\u0000")
      : "",
  ].join("\u0001");
}

/**
 * How many steps differ between two checklists: one whose place, mode or
 * added-step fields changed, and one only one of them has. It is the count
 * a customer's checklist shows against the organization's, and the unsaved
 * changes the editor counts.
 */
export function checklistDifferences(a: readonly StepLike[], b: readonly StepLike[]): number {
  const left = new Map(a.map((step, index) => [step.key, `${index}\u0002${stepSignature(step)}`]));
  const right = new Map(b.map((step, index) => [step.key, `${index}\u0002${stepSignature(step)}`]));
  let differences = 0;
  for (const key of new Set([...left.keys(), ...right.keys()])) {
    if (left.get(key) !== right.get(key)) {
      differences += 1;
    }
  }

  return differences;
}

/** Each step's place on the case, counting only steps that are not off. */
export function casePositions(steps: readonly StepLike[]): Map<string, number> {
  const positions = new Map<string, number>();
  let position = 0;
  for (const step of steps) {
    if (step.mode !== "Off") {
      position += 1;
      positions.set(step.key, position);
    }
  }

  return positions;
}

/** A position as the checklist prints it: two digits, or a dash for a step that is off. */
export function formatCasePosition(position: number | undefined): string {
  return position === undefined ? "—" : String(position).padStart(2, "0");
}
