import { isRecordEntityType, recordPath } from "@/config/record-links";
import type {
  AgentPreviewMessageChannel,
  AgentPreviewOperation,
  PreviewRecordLink,
  ProposalPreview,
} from "@/lib/graphql/agent-preview";
import { formatNumber } from "@trenova/shared/i18n/format";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { DISPLAY_TYPES, type DisplayType } from "../readable-values";

/** What a write does to a record, as the word a person reads before its name. */
export function operationLabel(operation: AgentPreviewOperation, t: TranslateFn): string {
  switch (operation) {
    case "Create":
      return t("New");
    case "Update":
      return t("Change");
    case "Delete":
      return t("Delete");
    case "Archive":
      return t("Archive");
    case "Send":
      return t("Send");
    case "Run":
      return t("Run");
  }
}

/** How a message reaches the people it is for. */
export function channelLabel(channel: AgentPreviewMessageChannel, t: TranslateFn): string {
  switch (channel) {
    case "Email":
      return t("Email");
    case "SMS":
      return t("Text message");
    case "Dash":
      return t("Driver portal");
    case "EDI":
      return t("EDI");
    case "Comment":
      return t("Comment");
  }
}

/** Who can read a comment the write would leave. */
export function visibilityLabel(visibility: string, t: TranslateFn): string {
  switch (visibility.toLowerCase()) {
    case "internal":
      return t("Your organization only");
    case "customer":
      return t("The customer");
    case "driver":
      return t("The driver");
    default:
      return visibility;
  }
}

const DISPLAY_TYPE_SET: ReadonlySet<string> = new Set(DISPLAY_TYPES);

/**
 * The server types every value with the same classifier the artifacts use; a
 * type this client does not know reads as text rather than as nothing.
 */
export function displayTypeOf(valueType: string): DisplayType {
  return DISPLAY_TYPE_SET.has(valueType) ? (valueType as DisplayType) : "text";
}

/** Where a previewed record opens, when its kind has a page in the app. */
export function previewRecordPath(record: PreviewRecordLink | null | undefined): string | null {
  if (!record || record.id === "" || !isRecordEntityType(record.entityType)) {
    return null;
  }

  return recordPath(record.entityType, record.id);
}

const CURRENCY_CODE = /^[A-Za-z]{3}$/;

/**
 * An exact amount as the reader's locale writes it. Decimals arrive as
 * strings so nothing is lost on the wire; they become numbers only to be
 * drawn. A currency the server left blank reads as dollars rather than
 * throwing, which is what Intl does with an empty code.
 */
export function formatPreviewAmount(
  value: string | null | undefined,
  currency: string,
  { signed = false }: { signed?: boolean } = {},
): string | null {
  if (value === null || value === undefined || value.trim() === "") {
    return null;
  }
  const amount = Number(value);
  if (!Number.isFinite(amount)) {
    return value;
  }

  return formatNumber(amount, {
    style: "currency",
    currency: CURRENCY_CODE.test(currency) ? currency.toUpperCase() : "USD",
    signDisplay: signed ? "exceptZero" : "auto",
  });
}

/**
 * The earlier plan steps a step's records start from, in order and once each:
 * "uses the record step 1 changes" is said once however many of its fields
 * step 1 also touched.
 */
export function dependencySteps(preview: ProposalPreview): number[] {
  const steps = new Set<number>();
  for (const change of preview.changes) {
    if (change.dependsOnStep > 0) {
      steps.add(change.dependsOnStep);
    }
  }

  return [...steps].sort((a, b) => a - b);
}

/**
 * Whether the preview carries a message as it would go out, rendered and
 * addressed. A surface that shows a draft prefers it to the model's own copy.
 */
export function previewSendsMessage(preview: ProposalPreview): boolean {
  return preview.changes.some((change) => !change.withheld && change.message !== null);
}
