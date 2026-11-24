import type { DeskBoardWords } from "@/lib/graphql/desk-board-card";
import { cardDayAfter, cardMomentAfter } from "../../record-card/record-card-values";
import { formatBoardValue, measureMoment } from "./board-card";

const SLOT = "{0}";

/**
 * Words the server wrote, with `{0}` filled: by the value, or by the moment written as
 * precisely as its age needs (a day when the server says it is a date). The place is not
 * part of it: a row draws it on its own, the first thing to give way.
 */
export function fillWords(words: DeskBoardWords | null | undefined, now: number): string {
  if (!words) return "";
  if (!words.text.includes(SLOT)) return words.text;
  const at = words.at ?? 0;
  const filled = words.value
    ? formatBoardValue(words.value)
    : measureMoment(now, at, words.day ? "date" : "moment");
  return words.text.replaceAll(SLOT, filled);
}

/** The same words written whole, the moment to the minute and the place after it, for a title. */
export function wholeWords(words: DeskBoardWords | null | undefined, now: number): string {
  if (!words) return "";
  const at = words.at ?? 0;
  const moment = at ? (words.day ? cardDayAfter(now, at) : cardMomentAfter(now, at)) : "";
  const filled = words.value ? formatBoardValue(words.value) : moment;
  const text = words.text.replaceAll(SLOT, filled);
  return words.place ? `${text} ${words.place}` : text;
}
