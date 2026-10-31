import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { BulkRunAction } from "@trenova/shared/types/billing-queue";

export type BulkRunCopy = {
  running: (count: number) => string;
  done: (count: number) => string;
  partial: (succeeded: number, total: number) => string;
};

/** What a bulk run says while it waits, when it finishes, and when it only partly does. */
export function bulkRunCopy(action: BulkRunAction, t: TranslateFn): BulkRunCopy {
  switch (action) {
    case "Approve":
      return {
        running: (n) => t("{0, plural, one {Approving # item} other {Approving # items}}", n),
        done: (n) => t("{0, plural, one {Approved # item} other {Approved # items}}", n),
        partial: (s, n) => t("Approved {0} of {1}", s, n),
      };
    case "Post":
      return {
        running: (n) => t("{0, plural, one {Posting # invoice} other {Posting # invoices}}", n),
        done: (n) => t("{0, plural, one {Posted # invoice} other {Posted # invoices}}", n),
        partial: (s, n) => t("Posted {0} of {1}", s, n),
      };
    case "Hold":
      return {
        running: (n) => t("{0, plural, one {Holding # item} other {Holding # items}}", n),
        done: (n) => t("{0, plural, one {Put # item on hold} other {Put # items on hold}}", n),
        partial: (s, n) => t("Held {0} of {1}", s, n),
      };
    case "Assign":
      return {
        running: (n) => t("{0, plural, one {Assigning # item} other {Assigning # items}}", n),
        done: (n) => t("{0, plural, one {Assigned # item} other {Assigned # items}}", n),
        partial: (s, n) => t("Assigned {0} of {1}", s, n),
      };
    case "Cancel":
      return {
        running: (n) => t("{0, plural, one {Canceling # item} other {Canceling # items}}", n),
        done: (n) => t("{0, plural, one {Canceled # item} other {Canceled # items}}", n),
        partial: (s, n) => t("Canceled {0} of {1}", s, n),
      };
    case "RequestPaperwork":
      return {
        running: (n) =>
          t(
            "{0, plural, one {Asking for paperwork on # item} other {Asking for paperwork on # items}}",
            n,
          ),
        done: (n) =>
          t(
            "{0, plural, one {Asked for paperwork on # item} other {Asked for paperwork on # items}}",
            n,
          ),
        partial: (s, n) => t("Asked on {0} of {1}", s, n),
      };
    case "ResolveIssue":
      return {
        running: (n) => t("{0, plural, one {Settling # item} other {Settling # items}}", n),
        done: (n) => t("{0, plural, one {Settled # item} other {Settled # items}}", n),
        partial: (s, n) => t("Settled {0} of {1}", s, n),
      };
  }
}
