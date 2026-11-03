import type { CardCheckState } from "./record-card";

/*
What a record's checks are, apart from `CardDue`, which draws them: a card's
pure helpers and the command palette's previews read them without bringing the
card frame along.
*/

/** One thing a card checks on its record, drawn as a `CardCheck`. */
export type DueCheck = {
  key: string;
  state: CardCheckState;
  title: string;
  detail: string;
};

const RANK: Record<CardCheckState, number> = { bad: 0, warn: 1, info: 2, ok: 3 };

/**
 * Checks with what is wrong first, then what is close, then what is only worth
 * knowing, then what holds, each kept in its order.
 */
export function worstFirst(checks: readonly DueCheck[]): DueCheck[] {
  return checks
    .map((check, index) => ({ check, index }))
    .sort((a, b) => RANK[a.check.state] - RANK[b.check.state] || a.index - b.index)
    .map(({ check }) => check);
}
