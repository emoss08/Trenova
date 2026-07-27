/**
 * The semantic colour a KPI carries. It lives with the KPI primitives rather
 * than with any one page's data shape, because every surface that renders a
 * KPI has to speak it.
 */
export type Tone = "success" | "danger" | "warning" | "brand" | "info" | "muted";

export type DeltaTone = Tone;

export function toneVar(tone: Tone | undefined): string {
  switch (tone) {
    case "success":
      return "var(--success)";
    case "danger":
      return "var(--destructive)";
    case "warning":
      return "var(--warning)";
    case "brand":
      return "var(--brand)";
    case "info":
      return "var(--info)";
    case "muted":
      return "var(--muted-foreground)";
    default:
      return "var(--muted-foreground)";
  }
}
