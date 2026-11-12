import type { DeskPlace } from "../rail/desk-rail";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { lensFromParam, lensLabel } from "../attention/lens";

export type DeskTitleParts = {
  place: DeskPlace;
  /** The open conversation's name; null when no conversation is in front. */
  threadTitle: string | null;
  /** The agent whose page is in front, or the conversation's. */
  agentName: string | null;
  /** The record of the case in front. */
  caseLabel: string | null;
  /** What a view or a handled line calls itself. */
  pageTitle: string | null;
  /** The `?lens=` of Needs you. */
  lens: string | null;
};

/** The Desk's simple places, by name. */
export function placeName(t: TranslateFn, place: DeskPlace): string {
  switch (place) {
    case "triage":
      return t("Triage");
    case "decisions":
      return t("Decisions");
    case "scheduled":
      return t("Scheduled");
    case "memory":
      return t("Memory");
    default:
      return t("Today");
  }
}

/**
 * What the browser tab says while the Desk is in front: the thing being
 * looked at, then that it is the Desk, so a row of tabs tells a case from a
 * conversation from the queue. The organization's name is added by `Metadata`.
 */
export function deskTitle(t: TranslateFn, parts: DeskTitleParts): string {
  const here = deskTitleHere(t, parts);
  return here === "" ? t("Desk") : t("{0} · Desk", here);
}

function deskTitleHere(t: TranslateFn, parts: DeskTitleParts): string {
  switch (parts.place) {
    case "thread":
      return parts.threadTitle === null
        ? ""
        : parts.threadTitle.trim() || t("Untitled conversation");
    case "agent":
      return parts.agentName ?? t("Agent");
    case "needs": {
      const lens = lensFromParam(parts.lens);
      return lens === "Everything" ? t("Needs you") : t("{0} · Needs you", lensLabel(t, lens));
    }
    case "case":
      return parts.caseLabel ? t("{0} · Case", parts.caseLabel) : t("Case");
    case "handled":
      return parts.pageTitle ?? t("Agents handled");
    case "view":
      return parts.pageTitle ?? t("View");
    default:
      return placeName(t, parts.place);
  }
}
