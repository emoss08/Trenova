import type { WatchtowerPlayChannel, WatchtowerPlayKind } from "@trenova/graphql/generated/graphql";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/**
 * The words on the button that makes a condition's first move. The server
 * says which move a kind of condition has (`DeskCaseHandOff.play`); the
 * person sends it themselves from `CasePlayDialog`.
 */
export function casePlayLabel(play: WatchtowerPlayKind, t: TranslateFn): string {
  switch (play) {
    case "AskForPod":
      return t("Ask for the POD");
    case "AskForStatus":
      return t("Ask where it is");
    case "TenderAgain":
      return t("Tender it again");
  }
}

/** The same move, once it has been made before: the button that makes it again. */
export function casePlayAgainLabel(play: WatchtowerPlayKind, t: TranslateFn): string {
  switch (play) {
    case "AskForPod":
    case "AskForStatus":
      return t("Ask again");
    case "TenderAgain":
      return t("Tender it again");
  }
}

/** How a play reaches whoever it is for, as a person calls it. */
export function casePlayChannelLabel(channel: WatchtowerPlayChannel, t: TranslateFn): string {
  switch (channel) {
    case "Dash":
      return t("Dash message");
    case "Text":
      return t("Text message");
    case "Email":
      return t("Email");
    case "Tender":
      return t("Tender");
  }
}
