import {
  parseAsInteger,
  parseAsString,
  parseAsStringLiteral,
  useQueryState,
} from "nuqs";
import { useCallback, useTransition } from "react";

const DEFAULT_PAGE_SIZE = 10;
export const SHIPMENT_VIEWS = ["list", "map"] as const;
export type ShipmentView = (typeof SHIPMENT_VIEWS)[number];

export function useShipmentParams() {
  const [isTransitioning, startTransition] = useTransition();

  const [page, setPage] = useQueryState(
    "page",
    parseAsInteger.withDefault(1).withOptions({
      startTransition,
      shallow: true,
    }),
  );

  const [pageSize, setPageSize] = useQueryState(
    "pageSize",
    parseAsInteger.withDefault(DEFAULT_PAGE_SIZE).withOptions({
      startTransition,
      shallow: true,
    }),
  );

  const [selectedShipmentId, setSelectedShipmentId] = useQueryState(
    "selectedShipmentId",
    parseAsString.withDefault("").withOptions({
      startTransition,
      shallow: true,
    }),
  );

  const [view, setView] = useQueryState(
    "view",
    parseAsStringLiteral(SHIPMENT_VIEWS).withOptions({
      startTransition,
      shallow: true,
    }),
  );

  const updateParams = useCallback(
    (updates: {
      page?: number;
      pageSize?: number;
      selectedShipmentId?: string;
      view?: ShipmentView;
    }) => {
      startTransition(() => {
        const operations = [];
        if (updates.view !== undefined) {
          operations.push(() => setView(updates.view as ShipmentView));
        }
        if (updates.selectedShipmentId !== undefined) {
          operations.push(() =>
            setSelectedShipmentId(updates.selectedShipmentId as string),
          );
        }
        if (updates.pageSize !== undefined) {
          operations.push(() => setPageSize(updates.pageSize as number));
          operations.push(() => setPage(1)); // Reset page when changing page size
        } else if (updates.page !== undefined) {
          operations.push(() => setPage(updates.page as number));
        }

        // Execute all operations in sequence
        operations.forEach((op) => op());
      });
    },
    [setPage, setPageSize, setSelectedShipmentId, setView],
  );

  return {
    page,
    pageSize,
    selectedShipmentId,
    view,
    updateParams,
    isTransitioning,
    DEFAULT_PAGE_SIZE,
  };
}
