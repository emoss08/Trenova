import type { SequenceConfig } from "@/types/sequence-config";

export function buildSequencePreview(cfg?: SequenceConfig): string {
  if (!cfg) {
    return "";
  }

  if (cfg.sequenceType === "location_code") {
    return buildLocationCodePreview(cfg);
  }

  if (cfg.allowCustomFormat && cfg.customFormat?.trim()) {
    return applyTemplate(cfg.customFormat, cfg);
  }

  const separator = cfg.useSeparators ? cfg.separatorChar || "-" : "";
  const parts: string[] = [];

  if (cfg.prefix) parts.push(cfg.prefix);
  if (cfg.includeYear) parts.push(cfg.yearDigits === 4 ? "2026" : "26");
  if (cfg.includeMonth) parts.push("02");
  if (cfg.includeWeekNumber) parts.push("09");
  if (cfg.includeDay) parts.push("28");
  if (cfg.includeLocationCode) parts.push("LOC");
  if (cfg.includeBusinessUnitCode) parts.push("BU");
  parts.push("9".repeat(Math.max(1, cfg.sequenceDigits || 1)));
  if (cfg.includeRandomDigits) {
    parts.push("7".repeat(Math.max(1, cfg.randomDigitsCount || 1)));
  }
  if (cfg.includeCheckDigit) parts.push("3");

  return parts.join(separator);
}

export function buildLocationCodePreview(cfg: SequenceConfig): string {
  const strategy = cfg.locationCodeStrategy ?? {
    components: ["name", "city", "state"] as const,
    componentWidth: 3,
    sequenceDigits: 3,
    separator: "-",
    casing: "upper" as const,
    fallbackPrefix: "LOC",
  };
  const sample = {
    name: "Acme Warehouse",
    city: "Dallas",
    state: "TX",
    postal_code: "75201",
  } as const;
  const width = Math.max(1, strategy.componentWidth || 1);
  const fallback = normalizeToken(strategy.fallbackPrefix || "LOC", strategy.casing).slice(0, width);
  const parts: string[] = [];
  let lastWasFallback = false;

  for (const component of strategy.components) {
    const normalized = normalizeToken(sample[component] ?? "", strategy.casing).slice(0, width);
    const usedFallback = normalized.length === 0;
    if (usedFallback && lastWasFallback) {
      continue;
    }
    parts.push(normalized || fallback);
    lastWasFallback = usedFallback;
  }

  const sequence = "1".padStart(Math.max(1, strategy.sequenceDigits || 1), "0");

  return [...parts, sequence].join(strategy.separator ?? "");
}

function normalizeToken(value: string, casing: "upper" | "lower"): string {
  const normalized = value.replace(/[^a-zA-Z0-9]/g, "");
  return casing === "lower" ? normalized.toLowerCase() : normalized.toUpperCase();
}

export function applyTemplate(template: string, cfg: SequenceConfig): string {
  const tokenMap: Record<string, string> = {
    P: cfg.prefix || "",
    Y: cfg.yearDigits === 4 ? "2026" : "26",
    M: "02",
    W: "09",
    D: "28",
    L: "LOC",
    B: "BU",
    S: "9".repeat(Math.max(1, cfg.sequenceDigits || 1)),
    R: "7".repeat(Math.max(1, cfg.randomDigitsCount || 1)),
    C: "3",
  };

  return template.replace(/\{([PYMWDLBSRC])\}/g, (_, token: string) => {
    return tokenMap[token] ?? "";
  });
}
