import type {
  FactoringReserveAction,
  FactoringSubmissionDetail,
  FactoringSubmissionItem,
} from "@/lib/graphql/factoring";
import type {
  FactoringFundingItemInput,
  FactoringReserveMovementInput,
  RecordFactoringFundingInput,
} from "@trenova/graphql/generated/graphql";
import { translate } from "@trenova/shared/i18n/runtime";
import { z } from "zod";

/** The longest reason or memo the server keeps on a funding, release or chargeback. */
export const MAX_FACTORING_NOTE_LENGTH = 500;

/** An invoice on a sent schedule the factor has not funded or rejected yet. */
export function undecidedItems(submission: FactoringSubmissionDetail): FactoringSubmissionItem[] {
  return submission.items.filter((item) => item.status === "Pending");
}

const fundingItemSchema = z
  .object({
    itemId: z.string(),
    faceMinor: z.number().int(),
    outcome: z.enum(["Funded", "Rejected", "Undecided"]),
    advanceMinor: z
      .number({ error: () => translate("Enter the advance the factor paid") })
      .int()
      .min(0, { error: () => translate("The advance cannot be negative") }),
    feeMinor: z
      .number({ error: () => translate("Enter the fee the factor kept") })
      .int()
      .min(0, { error: () => translate("The fee cannot be negative") }),
    reason: z.string().max(MAX_FACTORING_NOTE_LENGTH, {
      error: () => translate("Keep the reason under 500 characters"),
    }),
  })
  .superRefine((item, ctx) => {
    if (item.outcome === "Rejected" && item.reason.trim() === "") {
      ctx.addIssue({
        code: "custom",
        path: ["reason"],
        message: translate("Say why the factor rejected it"),
      });
    }
    if (item.outcome === "Funded" && item.advanceMinor + item.feeMinor > item.faceMinor) {
      ctx.addIssue({
        code: "custom",
        path: ["feeMinor"],
        message: translate("The advance and fee come to more than the invoice"),
      });
    }
  });

export const fundingFormSchema = z
  .object({
    fundedOn: z
      .number({ error: () => translate("Enter the day the factor funded") })
      .positive({ error: () => translate("Enter the day the factor funded") }),
    memo: z.string().max(MAX_FACTORING_NOTE_LENGTH, {
      error: () => translate("Keep the memo under 500 characters"),
    }),
    items: z.array(fundingItemSchema),
  })
  .superRefine((values, ctx) => {
    if (!values.items.some((item) => item.outcome !== "Undecided")) {
      ctx.addIssue({
        code: "custom",
        path: ["items"],
        message: translate("Mark at least one invoice funded or rejected"),
      });
    }
  });

export type FundingFormValues = z.infer<typeof fundingFormSchema>;
export type FundingItemFormValues = FundingFormValues["items"][number];

/**
 * A funding form for a schedule: every undecided invoice marked Funded at the
 * advance and fee the agreement expected, so a report that matches what was
 * expected is recorded without retyping it.
 */
export function fundingFormDefaults(
  submission: FactoringSubmissionDetail,
  fundedOn: number,
): FundingFormValues {
  return {
    fundedOn,
    memo: "",
    items: undecidedItems(submission).map((item) => ({
      itemId: item.id,
      faceMinor: item.faceMinor,
      outcome: "Funded",
      advanceMinor: item.expectedAdvanceMinor,
      feeMinor: item.expectedFeeMinor,
      reason: "",
    })),
  };
}

/**
 * What the server is asked to plan or record: only the invoices the person
 * decided, each funded one with its advance and fee and each rejected one
 * with its reason. Returns null while nothing is decided or no day is set,
 * since the server would refuse it.
 */
export function fundingFormToInput(
  submission: Pick<FactoringSubmissionDetail, "id" | "version">,
  values: FundingFormValues,
): RecordFactoringFundingInput | null {
  if (!(values.fundedOn > 0)) {
    return null;
  }
  const items = values.items.flatMap((item): FactoringFundingItemInput[] => {
    switch (item.outcome) {
      case "Funded":
        return [
          {
            itemId: item.itemId,
            outcome: "Funded",
            advanceMinor: item.advanceMinor,
            feeMinor: item.feeMinor,
          },
        ];
      case "Rejected":
        return [{ itemId: item.itemId, outcome: "Rejected", reason: item.reason.trim() }];
      case "Undecided":
        return [];
    }
  });
  if (items.length === 0) {
    return null;
  }
  const memo = values.memo.trim();
  return {
    submissionId: submission.id,
    version: submission.version,
    fundedOn: values.fundedOn,
    items,
    memo: memo === "" ? null : memo,
  };
}

export const reserveMovementFormSchema = z
  .object({
    action: z.enum(["release", "chargeback", "reverse"]),
    occurredOn: z
      .number({ error: () => translate("Enter the day it happened") })
      .positive({ error: () => translate("Enter the day it happened") }),
    overrideCash: z.boolean(),
    cashMinor: z
      .number()
      .int()
      .min(0, { error: () => translate("The amount cannot be negative") }),
    reason: z.string().max(MAX_FACTORING_NOTE_LENGTH, {
      error: () => translate("Keep it under 500 characters"),
    }),
  })
  .superRefine((values, ctx) => {
    if (reserveActionNeedsReason(values.action) && values.reason.trim() === "") {
      ctx.addIssue({
        code: "custom",
        path: ["reason"],
        message:
          values.action === "chargeback"
            ? translate("Say why the factor charged it back")
            : translate("Say why the chargeback is reversed"),
      });
    }
  });

export type ReserveMovementFormValues = z.infer<typeof reserveMovementFormSchema>;

/** A chargeback and its reversal say why; a release's note is optional. */
export function reserveActionNeedsReason(action: FactoringReserveAction): boolean {
  return action !== "release";
}

/** A reversal moves the opposite of the chargeback it undoes, so it takes no amount. */
export function reserveActionTakesCash(action: FactoringReserveAction): boolean {
  return action !== "reverse";
}

/**
 * What the server is asked to plan or book for one invoice. The amount is
 * sent only when the person says the factor paid something other than the
 * default (the reserve held for a release, the advance and fee for a
 * chargeback), so the server's default stands otherwise.
 */
export function reserveMovementFormToInput(
  submission: Pick<FactoringSubmissionDetail, "id" | "version">,
  itemId: string,
  values: ReserveMovementFormValues,
): FactoringReserveMovementInput | null {
  if (!(values.occurredOn > 0)) {
    return null;
  }
  const reason = values.reason.trim();
  return {
    submissionId: submission.id,
    itemId,
    version: submission.version,
    occurredOn: values.occurredOn,
    cashMinor:
      reserveActionTakesCash(values.action) && values.overrideCash ? values.cashMinor : null,
    reason: reason === "" ? null : reason,
  };
}
