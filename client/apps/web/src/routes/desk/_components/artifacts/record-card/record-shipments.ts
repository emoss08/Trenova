import type { RecordShipment, ShipmentScope } from "@/lib/graphql/record-shipments";
import { cityStateLabel } from "@/lib/place";
import { recordShipmentsQuery } from "@/lib/queries/record-shipments";
import { getDestinationStop, getOriginStop } from "@/lib/shipment-utils";
import { useInfiniteQuery } from "@tanstack/react-query";
import { useMemo } from "react";

/** A shipment's lane and when it picks up, from its first pickup and last delivery. */
export type ShipmentLane = {
  origin: string | null;
  destination: string | null;
  pickupAt: number | null;
};

export function shipmentLane(shipment: RecordShipment): ShipmentLane {
  const origin = getOriginStop(shipment);
  const destination = getDestinationStop(shipment);
  return {
    origin: cityStateLabel(origin?.location?.city, origin?.location?.state?.abbreviation),
    destination: cityStateLabel(
      destination?.location?.city,
      destination?.location?.state?.abbreviation,
    ),
    pickupAt: origin?.scheduledWindowStart ?? null,
  };
}

/** The shipments a card lists for one record, as its list section draws them. */
export type RecordShipments = {
  /** Every page read so far, newest first; undefined until the first page lands. */
  rows: RecordShipment[] | undefined;
  /** The listed shipments' ids in the order shown, for stepping through them. */
  ids: readonly string[];
  /** How many there are in all, from the first page. */
  total: number | null;
  hasMore: boolean;
  loadingMore: boolean;
  loadMore: () => void;
  pending: boolean;
  error: unknown;
};

const NO_IDS: readonly string[] = [];

/**
 * A record's shipments, newest first, a page at a time, read only when the
 * person may read shipments (`enabled`). The next page is read on request and
 * added under the ones shown.
 */
export function useRecordShipments({
  scope,
  pageSize,
  enabled,
}: {
  scope: ShipmentScope;
  pageSize: number;
  enabled: boolean;
}): RecordShipments {
  const query = useInfiniteQuery({
    ...recordShipmentsQuery(scope, pageSize),
    enabled,
    retry: false,
  });
  const rows = useMemo(() => query.data?.pages.flatMap((page) => page.rows), [query.data]);
  const ids = useMemo(() => rows?.map((row) => row.id) ?? NO_IDS, [rows]);
  const { fetchNextPage } = query;

  return {
    rows,
    ids,
    total: query.data?.pages[0]?.total ?? null,
    hasMore: query.hasNextPage,
    loadingMore: query.isFetchingNextPage,
    loadMore: () => void fetchNextPage(),
    pending: query.isPending,
    error: query.error,
  };
}
