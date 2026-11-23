/**
 * A stored code (an enum value) written as words for a person, for the moments the label its
 * owner gives it is not at hand: `AgentTurn` reads "Agent Turn", `MissingBOL` "Missing BOL",
 * `amount_mismatch` "Amount Mismatch", and `POD` stays "POD". Title case, like the labels the
 * server writes for enum values, so a fallback and the real label look alike. A code written
 * all in capitals with separators (`IN_TRANSIT`) is a constant's name, not initialisms, and
 * reads "In Transit".
 */
export function codeWords(code: string): string {
  const shouting = !/[a-z]/.test(code) && /[_\-\s]/.test(code);
  const words = code
    .replace(/[_-]+/g, " ")
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .replace(/([A-Z]+)([A-Z][a-z])/g, "$1 $2")
    .trim()
    .split(/\s+/)
    .filter(Boolean);
  if (words.length === 0) return code;
  return words
    .map((word) => {
      const base = shouting ? word.toLowerCase() : word;
      return base.charAt(0).toUpperCase() + base.slice(1);
    })
    .join(" ");
}
