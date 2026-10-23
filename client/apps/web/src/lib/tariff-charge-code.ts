import type { TariffDiagnostic } from "@trenova/shared/types/tariff";

/** An accessorial charge code as the catalog stores it: capitals, digits and `_`, up to ten. */
const CODE = /^[A-Z][A-Z0-9_]{0,9}$/;

/** `charge "Label" as ` ending where the code starts, on the code's own line. */
const LABELLED = /charge\s+"((?:[^"\\{}]|\\.)*)"\s+as\s+$/;

/**
 * The code an `unknown-code` finding is about, with the label of the charge that bills it as a
 * description for the new catalog entry (empty when the code is only read, as in
 * `charge(FSC)`); null for any other finding or a code the catalog couldn't store.
 */
export function missingChargeCode(
  diagnostic: TariffDiagnostic,
  source: string,
): { code: string; description: string } | null {
  if (diagnostic.code !== "unknown-code" || !diagnostic.range || diagnostic.module) return null;
  const { from, to } = diagnostic.range;
  const code = source.slice(from, to);
  if (!CODE.test(code)) return null;
  const lineStart = source.lastIndexOf("\n", from - 1) + 1;
  const label = LABELLED.exec(source.slice(lineStart, from))?.[1] ?? "";
  return { code, description: label.replace(/\\(.)/g, "$1") };
}
