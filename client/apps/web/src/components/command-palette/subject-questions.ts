import type { TranslateFn } from "@trenova/shared/i18n/use-t";

/**
 * The questions worth asking about one record before anything is typed, three
 * per kind. Each is a whole sentence so it translates as one.
 */
export function subjectQuestions(type: string, t: TranslateFn): string[] {
  switch (type) {
    case "shipment":
      return [
        t("Why is it running late?"),
        t("Draft an update for the customer"),
        t("What's left before we can bill it?"),
      ];
    case "customer":
      return [
        t("What's overdue and why?"),
        t("How has their volume changed this quarter?"),
        t("Draft a payment reminder"),
      ];
    case "worker":
      return [
        t("Can they take another load today?"),
        t("What's expiring soon?"),
        t("Summarize this week's hours"),
      ];
    case "document":
      return [
        t("What did Nova pull from this?"),
        t("Does it match the shipment?"),
        t("Who sent it?"),
      ];
    default:
      return [];
  }
}
