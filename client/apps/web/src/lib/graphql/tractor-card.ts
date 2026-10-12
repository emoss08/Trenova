import {
  DeskTractorCardDocument,
  DeskTractorInspectionsDocument,
  DeskTractorPositionDocument,
  type DeskTractorCardQuery,
  type DeskTractorInspectionsQuery,
  type DeskTractorPositionQuery,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";

export type DeskTractor = NonNullable<DeskTractorCardQuery["tractor"]>;
export type DeskTractorPosition = NonNullable<DeskTractorPositionQuery["vehiclePosition"]>;
export type DeskTractorInspection = DeskTractorInspectionsQuery["vehicleInspections"][number];

/** Where the next page of inspections starts: before the last one shown. */
export type InspectionCursor = { before: number; beforeId: string };

/** A page of inspections, and whether there are older ones. */
export type TractorInspectionsPage = {
  rows: DeskTractorInspection[];
  next: InspectionCursor | null;
};

/**
 * One tractor as the Desk's card reads it. A tractor the server does not
 * return is an error rather than an empty card, so the card says it could not
 * be read.
 */
export async function fetchDeskTractor(
  id: string,
  options?: { signal?: AbortSignal },
): Promise<DeskTractor> {
  const data = await requestGraphQL({
    document: DeskTractorCardDocument,
    operationName: "DeskTractorCard",
    variables: { id },
    signal: options?.signal,
  });
  if (!data.tractor) {
    throw new Error("Tractor not found");
  }
  return data.tractor;
}

/** Where the tractor last reported itself; null when it never has. */
export async function fetchTractorPosition(
  tractorId: string,
  options?: { signal?: AbortSignal },
): Promise<DeskTractorPosition | null> {
  const data = await requestGraphQL({
    document: DeskTractorPositionDocument,
    operationName: "DeskTractorPosition",
    variables: { tractorId },
    signal: options?.signal,
  });
  return data.vehiclePosition ?? null;
}

/**
 * A page of the tractor's inspections, newest first. One more than the page is
 * asked for, so the card knows whether older ones remain without a count.
 */
export async function fetchTractorInspections(
  tractorId: string,
  page: { size: number; cursor: InspectionCursor | null },
  options?: { signal?: AbortSignal },
): Promise<TractorInspectionsPage> {
  const data = await requestGraphQL({
    document: DeskTractorInspectionsDocument,
    operationName: "DeskTractorInspections",
    variables: {
      tractorId,
      limit: page.size + 1,
      before: page.cursor?.before ?? null,
      beforeId: page.cursor?.beforeId ?? null,
    },
    signal: options?.signal,
  });
  const fetched = data.vehicleInspections;
  const rows = fetched.slice(0, page.size);
  const last = rows[rows.length - 1];
  return {
    rows,
    next: fetched.length > page.size && last ? { before: last.startedAt, beforeId: last.id } : null,
  };
}
