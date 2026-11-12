import type {
  AgentPreviewOperation,
  PreviewFieldChange,
  PreviewMessage,
  ProposalPreview,
} from "@/lib/graphql/agent-preview";
import { sharedTemplate, type SharedTemplate } from "@/lib/shared-template";
import { previewSubject } from "@/components/assistant/proposal-preview/preview-format";

/**
 * The most proposals one batch decides or previews: the server's
 * agentdecision.MaxBatch. A longer run of one kind is decided fifty at a time.
 */
export const MAX_BATCH = 50;

/** One proposal of a batch as the batch preview read returned it. */
export type BatchPreviewEntry = {
  proposalId: string;
  preview: ProposalPreview | null;
  error: string | null;
};

/** A value one change of the write would set, keyed alike across the batch. */
export type BatchField = {
  key: string;
  operation: AgentPreviewOperation;
  field: PreviewFieldChange;
};

export type BatchRow = {
  proposalId: string;
  preview: ProposalPreview | null;
  error: string | null;
  /** The record the change is about, by the name a person knows it by. */
  title: string;
  /** Who the message goes to; empty for a write that sends none. */
  to: string[];
  subject: string;
  body: string;
  /** What fills the subject's and body's template, or null to show them whole. */
  subjectValues: string[] | null;
  bodyValues: string[] | null;
  /** The values this row sets that not every row sets alike. */
  fields: BatchField[];
};

/**
 * A batch read as one review. `compact` when every change has the same
 * shape: what is alike is shown once, a message's wording as a template, and
 * each row shows only what is its own. `whole` when the changes differ in
 * shape, and each row shows its preview in full.
 */
export type BatchReview = {
  layout: "compact" | "whole";
  /** How the messages go out, alike for every row. */
  envelope: PreviewMessage | null;
  subject: SharedTemplate | null;
  body: SharedTemplate | null;
  shared: BatchField[];
  rows: BatchRow[];
};

function messageOf(preview: ProposalPreview): PreviewMessage | null {
  for (const change of preview.changes) {
    if (!change.withheld && change.message) {
      return change.message;
    }
  }

  return null;
}

function shapeOf(preview: ProposalPreview): string {
  return JSON.stringify(
    preview.changes.map((change) => [
      change.resource,
      change.operation,
      change.withheld,
      change.message !== null,
    ]),
  );
}

/** How a message goes out, apart from who it goes to and what it says. */
function envelopeOf(message: PreviewMessage | null): string {
  if (message === null) {
    return "";
  }

  return JSON.stringify([
    message.channel,
    message.from,
    message.cc,
    message.bcc,
    message.visibility,
    message.cadence,
    message.attachments,
    message.templateVersionId ?? null,
  ]);
}

function fieldsOf(preview: ProposalPreview): BatchField[] {
  return preview.changes.flatMap((change, index) =>
    change.fields.map((field) => ({
      key: `${index}:${field.path}`,
      operation: change.operation,
      field,
    })),
  );
}

function valueOf(field: PreviewFieldChange): string {
  return JSON.stringify([
    field.withheld,
    field.before ?? null,
    field.after ?? null,
    field.beforeRef?.label ?? null,
    field.afterRef?.label ?? null,
    field.truncated,
    field.changedSinceProposed,
  ]);
}

function titleOf(preview: ProposalPreview | null): string {
  if (preview === null) {
    return "";
  }
  const subject = previewSubject(preview);
  if (subject !== "") {
    return subject;
  }

  return preview.changes.find((change) => !change.withheld && change.label !== "")?.label ?? "";
}

function rowOf(entry: BatchPreviewEntry): BatchRow {
  const message = entry.preview ? messageOf(entry.preview) : null;

  return {
    proposalId: entry.proposalId,
    preview: entry.preview,
    error: entry.error,
    title: titleOf(entry.preview),
    to: message?.to ?? [],
    subject: message?.subject ?? "",
    body: message?.body ?? "",
    subjectValues: null,
    bodyValues: null,
    fields: [],
  };
}

function isCompact(previews: readonly ProposalPreview[]): boolean {
  if (previews.length < 2) {
    return false;
  }
  const shape = shapeOf(previews[0]);
  const envelope = envelopeOf(messageOf(previews[0]));

  return previews.every(
    (preview) =>
      shapeOf(preview) === shape &&
      envelopeOf(messageOf(preview)) === envelope &&
      preview.changes.filter((change) => change.message !== null).length <= 1,
  );
}

/** Which values every row sets alike, by key; a key some row lacks is not alike. */
function sharedKeys(rows: readonly BatchField[][]): Set<string> {
  const [first, ...rest] = rows;
  const shared = new Set<string>();
  for (const candidate of first ?? []) {
    const value = valueOf(candidate.field);
    const alike = rest.every((fields) =>
      fields.some((other) => other.key === candidate.key && valueOf(other.field) === value),
    );
    if (alike) {
      shared.add(candidate.key);
    }
  }

  return shared;
}

/**
 * Reads a batch's previews as one review. Rows keep the order asked; a row
 * whose preview could not be read keeps its error and takes no part in what
 * is alike.
 */
export function batchReview(entries: readonly BatchPreviewEntry[]): BatchReview {
  const rows = entries.map(rowOf);
  const read = rows.filter(
    (row): row is BatchRow & { preview: ProposalPreview } => row.preview !== null,
  );
  const previews = read.map((row) => row.preview);

  if (!isCompact(previews)) {
    return { layout: "whole", envelope: null, subject: null, body: null, shared: [], rows };
  }

  const fields = previews.map(fieldsOf);
  const alike = sharedKeys(fields);
  read.forEach((row, i) => {
    row.fields = fields[i].filter((field) => !alike.has(field.key));
  });

  const envelope = messageOf(previews[0]);
  let subject: SharedTemplate | null = null;
  let body: SharedTemplate | null = null;
  if (envelope !== null) {
    subject = sharedTemplate(read.map((row) => row.subject));
    body = sharedTemplate(read.map((row) => row.body));
    read.forEach((row, i) => {
      row.subjectValues = subject?.values[i] ?? null;
      row.bodyValues = body?.values[i] ?? null;
    });
  }

  return {
    layout: "compact",
    envelope,
    subject,
    body,
    shared: fields[0].filter((field) => alike.has(field.key)),
    rows,
  };
}

/**
 * The rows a batch approval can carry: read, not stale, and not left out by
 * the person. Each goes with the digest of the preview the sheet showed.
 */
export function approvableRows(
  review: BatchReview,
  excluded: ReadonlySet<string>,
): { proposalId: string; digest: string }[] {
  return review.rows.flatMap((row) =>
    row.preview !== null && !row.preview.stale && !excluded.has(row.proposalId)
      ? [{ proposalId: row.proposalId, digest: row.preview.digest }]
      : [],
  );
}
