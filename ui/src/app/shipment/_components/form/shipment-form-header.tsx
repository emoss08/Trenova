/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

/* eslint-disable react-hooks/exhaustive-deps */
import { useDataTable } from "@/components/data-table/data-table-provider";
import { Kbd } from "@/components/kbd";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { Separator } from "@/components/ui/separator";
import { SheetClose } from "@/components/ui/sheet";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import {
  faChevronDown,
  faChevronLeft,
  faChevronUp,
} from "@fortawesome/pro-solid-svg-icons";
import { memo, useCallback, useEffect, useMemo } from "react";
import { ShipmentActions } from "./shipment-menu-actions";

type ShipmentFormHeaderProps = {
  selectedShipment?: ShipmentSchema | null;
};

export function ShipmentFormHeader({
  selectedShipment,
}: ShipmentFormHeaderProps) {
  const { table, rowSelection, isLoading } = useDataTable();

  const selectedRowKey = Object.keys(rowSelection)?.[0];

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
  }, [nextId, isLoading]);

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
    <ShipmentFormHeaderInner>
      <HeaderBackButton />
      <div className="flex h-7 items-center gap-1">
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                size="icon"
                variant="ghost"
                className="h-7 w-7"
                disabled={!prevId}
                onClick={onPrev}
              >
                <Icon icon={faChevronUp} className="h-5 w-5" />
                <span className="sr-only">Previous</span>
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>
                Navigate <Kbd variant="outline">↑</Kbd>
              </p>
            </TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                size="icon"
                variant="ghost"
                className="h-7 w-7"
                disabled={!nextId}
                onClick={onNext}
              >
                <Icon icon={faChevronDown} className="h-5 w-5" />
                <span className="sr-only">Next</span>
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              <p>
                Navigate <Kbd variant="outline">↓</Kbd>
              </p>
            </TooltipContent>
          </Tooltip>
        </TooltipProvider>
        <Separator orientation="vertical" className="mx-1" />
        <ShipmentActions shipment={selectedShipment} />
      </div>
    </ShipmentFormHeaderInner>
  );
}

const HeaderBackButton = memo(function HeaderBackButton() {
  return (
    <SheetClose asChild>
      <Button variant="outline">
        <Icon icon={faChevronLeft} className="size-4" />
        <span className="text-sm">Back</span>
      </Button>
    </SheetClose>
  );
});

export function ShipmentFormHeaderInner({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between px-2 py-4">
      {children}
    </div>
  );
}
