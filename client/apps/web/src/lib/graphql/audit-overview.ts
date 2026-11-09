import { type FragmentType, getFragmentData } from "@trenova/graphql/fragment-data";
import {
  AuditEntryReviewDocument,
  AuditOverviewDocument,
  AuditReviewFieldsFragmentDoc,
  AuditReviewsDocument,
  MarkAuditReviewReviewedDocument,
  ReopenAuditReviewDocument,
  type AuditOverviewQuery,
  type AuditReviewFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

type RequestOptions = { signal?: AbortSignal };

type RawOverview = AuditOverviewQuery["auditOverview"];

export type AuditReview = Omit<AuditReviewFieldsFragment, " $fragmentName">;

/** The overview as the page draws it, with its reviews unmasked once, here. */
export type AuditOverview = Omit<RawOverview, "openReviews"> & { openReviews: AuditReview[] };

export type AuditOverviewSegment = AuditOverview["summary"][number];
export type AuditActivityBucket = AuditOverview["activity"][number];
export type AuditActorStat = AuditOverview["topActors"][number];
export type AuditOverviewFigures = AuditOverview["totals"];

export type AuditReviewList = {
  open: AuditReview[];
  openCount: number;
  reviewed: AuditReview[];
  reviewedCount: number;
};

/** The organization's realtime topic for reviews: raised by an alert, marked or reopened. */
export const AUDIT_REVIEWS_RESOURCE = "audit-reviews";

function unmaskReview(raw: FragmentType<typeof AuditReviewFieldsFragmentDoc>): AuditReview {
  return getFragmentData(AuditReviewFieldsFragmentDoc, raw);
}

/** The overview of the last `days` whole days of the organization's time zone, today included. */
export async function fetchAuditOverview(
  days: number,
  options?: RequestOptions,
): Promise<AuditOverview> {
  const data = await requestGraphQL({
    document: AuditOverviewDocument,
    operationName: "AuditOverview",
    variables: { days },
    signal: options?.signal,
  });
  const raw = data.auditOverview;
  return { ...raw, openReviews: raw.openReviews.map(unmaskReview) };
}

/** Open reviews, latest first, and the ones reviewed in the last seven days. */
export async function fetchAuditReviews(
  limit: number,
  options?: RequestOptions,
): Promise<AuditReviewList> {
  const data = await requestGraphQL({
    document: AuditReviewsDocument,
    operationName: "AuditReviews",
    variables: { limit },
    signal: options?.signal,
  });
  const raw = data.auditReviews;
  return {
    open: raw.open.map(unmaskReview),
    openCount: raw.openCount,
    reviewed: raw.reviewed.map(unmaskReview),
    reviewedCount: raw.reviewedCount,
  };
}

/** The review an entry set off, or null when it set none off. */
export async function fetchAuditEntryReview(
  entryId: string,
  options?: RequestOptions,
): Promise<AuditReview | null> {
  const data = await requestGraphQL({
    document: AuditEntryReviewDocument,
    operationName: "AuditEntryReview",
    variables: { entryId },
    signal: options?.signal,
  });
  return data.auditEntryReview ? unmaskReview(data.auditEntryReview) : null;
}

export async function markAuditReviewReviewed(id: string): Promise<AuditReview> {
  const data = await requestGraphQL({
    document: MarkAuditReviewReviewedDocument,
    operationName: "MarkAuditReviewReviewed",
    variables: { id },
  });
  return unmaskReview(data.markAuditReviewReviewed);
}

export async function reopenAuditReview(id: string): Promise<AuditReview> {
  const data = await requestGraphQL({
    document: ReopenAuditReviewDocument,
    operationName: "ReopenAuditReview",
    variables: { id },
  });
  return unmaskReview(data.reopenAuditReview);
}
