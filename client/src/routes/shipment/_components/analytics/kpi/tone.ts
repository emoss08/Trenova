import type { Tone } from "../mock-data";

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
