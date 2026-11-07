import { getFragmentData } from "@trenova/graphql/fragment-data";
import {
  DispatchCallOffPositioningDocument,
  DispatchForecastQualityFieldsFragmentDoc,
  DispatchFreightAreaDocument,
  DispatchFreightAreaFieldsFragmentDoc,
  DispatchFreightMoveFieldsFragmentDoc,
  DispatchFreightOutlookDocument,
  DispatchFreightPaceFieldsFragmentDoc,
  DispatchPositionDriverDocument,
  DispatchPositioningFieldsFragmentDoc,
  type DispatchFreightAreaFieldsFragment,
  type DispatchFreightAreaQuery,
  type DispatchFreightMoveFieldsFragment,
  type DispatchFreightOutlookQuery,
  type DispatchFreightPaceFieldsFragment,
  type DispatchPositionDriverInput,
  type DispatchPositioningFieldsFragment,
} from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import type { DispatchForecastQuality } from "./dispatch-console";

export type FreightArea = DispatchFreightAreaFieldsFragment;
export type FreightSlot = FreightArea["slots"][number];
export type FreightMove = DispatchFreightMoveFieldsFragment;
export type FreightMoveKind = FreightMove["kind"];
export type FreightPace = DispatchFreightPaceFieldsFragment;
export type FreightPositioning = DispatchPositioningFieldsFragment;

type OutlookData = DispatchFreightOutlookQuery["dispatchFreightOutlook"];
export type FreightNetworkSlot = OutlookData["network"][number];
export type ForecastDay = OutlookData["record"][number];

export type FreightOutlook = Omit<
  OutlookData,
  "quality" | "areas" | "moves" | "pace" | "positionings"
> & {
  quality: DispatchForecastQuality;
  areas: FreightArea[];
  moves: FreightMove[];
  pace: FreightPace[];
  positionings: FreightPositioning[];
};

type AreaData = DispatchFreightAreaQuery["dispatchFreightArea"];
export type FreightReach = AreaData["reach"][number];
export type FreightAreaBounds = AreaData["bounds"];

export type FreightAreaDetail = Omit<
  AreaData,
  "quality" | "area" | "moves" | "pace" | "positionings"
> & {
  quality: DispatchForecastQuality;
  area: FreightArea;
  moves: FreightMove[];
  pace: FreightPace[];
  positionings: FreightPositioning[];
};

/**
 * The next 48 hours by area and four-hour window: where trucks will be short or idle, the moves
 * that fix it, customers off their usual pace and the forecast's track record.
 */
export async function fetchFreightOutlook(options?: {
  signal?: AbortSignal;
}): Promise<FreightOutlook> {
  const data = await requestGraphQL({
    document: DispatchFreightOutlookDocument,
    operationName: "DispatchFreightOutlook",
    signal: options?.signal,
  });
  const outlook = data.dispatchFreightOutlook;

  return {
    ...outlook,
    quality: getFragmentData(DispatchForecastQualityFieldsFragmentDoc, outlook.quality),
    areas: outlook.areas.map((area) => getFragmentData(DispatchFreightAreaFieldsFragmentDoc, area)),
    moves: outlook.moves.map((move) => getFragmentData(DispatchFreightMoveFieldsFragmentDoc, move)),
    pace: outlook.pace.map((pace) => getFragmentData(DispatchFreightPaceFieldsFragmentDoc, pace)),
    positionings: outlook.positionings.map((positioning) =>
      getFragmentData(DispatchPositioningFieldsFragmentDoc, positioning),
    ),
  };
}

/** One area of the outlook, at one window or its worst: who books there and who can get there. */
export async function fetchFreightArea(
  input: { cell: number; slotStart?: number | null },
  options?: { signal?: AbortSignal },
): Promise<FreightAreaDetail> {
  const data = await requestGraphQL({
    document: DispatchFreightAreaDocument,
    operationName: "DispatchFreightArea",
    variables: { cell: input.cell, slotStart: input.slotStart ?? null },
    signal: options?.signal,
  });
  const detail = data.dispatchFreightArea;

  return {
    ...detail,
    quality: getFragmentData(DispatchForecastQualityFieldsFragmentDoc, detail.quality),
    area: getFragmentData(DispatchFreightAreaFieldsFragmentDoc, detail.area),
    moves: detail.moves.map((move) => getFragmentData(DispatchFreightMoveFieldsFragmentDoc, move)),
    pace: detail.pace.map((pace) => getFragmentData(DispatchFreightPaceFieldsFragmentDoc, pace)),
    positionings: detail.positionings.map((positioning) =>
      getFragmentData(DispatchPositioningFieldsFragmentDoc, positioning),
    ),
  };
}

/** Sends an empty driver toward an area and tells them in Dash. */
export async function positionDriver(
  input: DispatchPositionDriverInput,
): Promise<FreightPositioning> {
  const data = await requestGraphQL({
    document: DispatchPositionDriverDocument,
    operationName: "DispatchPositionDriver",
    variables: { input },
  });

  return getFragmentData(DispatchPositioningFieldsFragmentDoc, data.dispatchPositionDriver);
}

/** Calls a positioning off and tells the driver. */
export async function callOffPositioning(id: string): Promise<FreightPositioning> {
  const data = await requestGraphQL({
    document: DispatchCallOffPositioningDocument,
    operationName: "DispatchCallOffPositioning",
    variables: { id },
  });

  return getFragmentData(DispatchPositioningFieldsFragmentDoc, data.dispatchCallOffPositioning);
}
