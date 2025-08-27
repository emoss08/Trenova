/* eslint-disable react-hooks/exhaustive-deps */
/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

"use no memo";

import { useDataTable } from "@/components/data-table/data-table-provider";
import {
  Sheet,
  SheetBody,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { VisuallyHidden } from "@/components/ui/visually-hidden";
import { type ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { EditTableSheetProps } from "@/types/data-table";
import { memo, useCallback, useEffect, useMemo, useRef } from "react";
import { ShipmentEditForm } from "./form/shipment-form";

export function ShipmentEditSheet({
  currentRecord,
}: EditTableSheetProps<ShipmentSchema>) {
  const { table, rowSelection, isLoading } = useDataTable();
  const sheetRef = useRef<HTMLDivElement>(null);

  const selectedRowKey = Object.keys(rowSelection)[0];

  const selectedRow = useMemo(() => {
    if (isLoading && !selectedRowKey) return;
    return table
      .getCoreRowModel()
      .flatRows.find((row) => row.id === selectedRowKey);
  }, [selectedRowKey, isLoading]);

  const index = table
    .getCoreRowModel()
    .flatRows.findIndex((row) => row.id === selectedRow?.id);

  const nextId = useMemo(
    () => table.getCoreRowModel().flatRows[index + 1]?.id,
    [index, isLoading],
  );

  const prevId = useMemo(
    () => table.getCoreRowModel().flatRows[index - 1]?.id,
    [index, isLoading],
  );

  const onPrev = useCallback(() => {
    if (prevId) table.setRowSelection({ [prevId]: true });
  }, [prevId, isLoading]);

  const onNext = useCallback(() => {
    if (nextId) table.setRowSelection({ [nextId]: true });
  }, [nextId, isLoading, table]);

  useEffect(() => {
    const down = (e: KeyboardEvent) => {
      if (!selectedRowKey) return;

      // REMINDER: prevent dropdown navigation inside of sheet to change row selection
      const activeElement = document.activeElement;
      const isMenuActive = activeElement?.closest('[role="menu"]');

      if (isMenuActive) return;

      if (e.key === "ArrowUp") {
        e.preventDefault();
        onPrev();
      }
      if (e.key === "ArrowDown") {
        e.preventDefault();
        onNext();
      }
    };

    document.addEventListener("keydown", down);
    return () => document.removeEventListener("keydown", down);
  }, [selectedRowKey, onNext, onPrev]);

  return (
    <>
      <Sheet
        open={!!selectedRowKey}
        onOpenChange={(open) => {
          if (!open) {
            const el = selectedRowKey
              ? document.getElementById(selectedRowKey)
              : null;

            table.resetRowSelection();
            setTimeout(() => el?.focus(), 0);
          }
        }}
      >
        <SheetContent
          className="w-[500px] sm:max-w-[540px] p-0"
          withClose={false}
          ref={sheetRef}
        >
          <VisuallyHidden>
            <SheetHeader>
              <SheetTitle>Shipment Details</SheetTitle>
            </SheetHeader>
            <SheetDescription>{currentRecord?.bol}</SheetDescription>
          </VisuallyHidden>
          <MemoizedShipmentSheetBody
            selectedShipment={currentRecord}
            isLoading={isLoading}
          />
        </SheetContent>
      </Sheet>
    </>
  );
}

function ShipmentSheetBody({
  selectedShipment,
  isLoading,
}: {
  selectedShipment?: ShipmentSchema | null;
  isLoading: boolean;
}) {
  return (
    <SheetBody className="p-0">
      <ShipmentEditForm
        selectedShipment={selectedShipment}
        isLoading={isLoading}
      />
    </SheetBody>
  );
}

const MemoizedShipmentSheetBody = memo(ShipmentSheetBody, (prev, next) => {
  // * we only check if the selectedShipment is the same, rest is useless
  return prev.selectedShipment === next.selectedShipment;
}) as typeof ShipmentSheetBody;
