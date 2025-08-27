export function classifyLevel(
  line: string,
): "error" | "warn" | "info" | "debug" | "other" {
  const s = line.toLowerCase();
  if (/\b(error|err|fatal|crit)\b/.test(s)) return "error";
  if (/\b(warn|warning)\b/.test(s)) return "warn";
  if (/\b(info|information)\b/.test(s)) return "info";
  if (/\b(debug|trace)\b/.test(s)) return "debug";
  return "other";
}

export function summarizeLevels(lines: string[]) {
  let ERROR = 0,
    WARN = 0,
    INFO = 0,
    DEBUG = 0,
    OTHER = 0;
  for (const ln of lines) {
    const c = classifyLevel(ln);
    if (c === "error") ERROR++;
    else if (c === "warn") WARN++;
    else if (c === "info") INFO++;
    else if (c === "debug") DEBUG++;
    else OTHER++;
  }
  return { ERROR, WARN, INFO, DEBUG, OTHER } as const;
}

export function escapeRegex(s: string) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}
