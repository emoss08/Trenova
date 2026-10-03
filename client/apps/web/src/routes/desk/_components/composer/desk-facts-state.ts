/** How many facts a conversation keeps; the server refuses more. */
export const MAX_PINNED_FACTS = 12;
/** How long one fact may be; the server refuses longer. */
export const MAX_PINNED_FACT_LENGTH = 200;

/**
 * The list with a typed fact pinned at the end. Blank input, a fact already
 * pinned, or a full list leaves it as it was, the same list, so a caller can
 * tell nothing changed.
 */
export function pinFact(facts: readonly string[], typed: string): readonly string[] {
  const fact = typed.trim().replace(/\s+/g, " ").slice(0, MAX_PINNED_FACT_LENGTH);
  if (fact === "" || facts.includes(fact) || facts.length >= MAX_PINNED_FACTS) {
    return facts;
  }

  return [...facts, fact];
}

/** The list without one fact. */
export function unpinFact(facts: readonly string[], fact: string): readonly string[] {
  return facts.includes(fact) ? facts.filter((kept) => kept !== fact) : facts;
}
