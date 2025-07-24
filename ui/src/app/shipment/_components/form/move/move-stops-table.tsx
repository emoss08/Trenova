/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { ScrollArea } from "@/components/ui/scroll-area";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { generateDateTimeStringFromUnixTimestamp } from "@/lib/date";
import { mapToStopType, type StopSchema } from "@/lib/schemas/stop-schema";
import { faEdit, faLock, faTrash } from "@fortawesome/pro-solid-svg-icons";

type CompactStopsTableProps = {
  stops: StopSchema[];
  onEdit: (stopIdx: number) => void;
  onDelete: (stopIdx: number) => void;
};

export function CompactStopsTable({
  stops,
  onEdit,
  onDelete,
}: CompactStopsTableProps) {
  return (
    <div className="border rounded-md overflow-hidden">
      <ScrollArea className="max-h-[300px]">
        <Table>
          <TableHeader className="h-8">
            <TableRow className="bg-sidebar h-8">
              <TableHead className="w-[5%] text-2xs py-1 h-8">#</TableHead>
              <TableHead className="w-[15%] text-2xs py-1 h-8">Type</TableHead>
              <TableHead className="w-[30%] text-2xs py-1 h-8">
                Address
              </TableHead>
              <TableHead className="w-[20%] text-2xs py-1 h-8">
                Arrival
              </TableHead>
              <TableHead className="w-[20%] text-2xs py-1 h-8">
                Departure
              </TableHead>
              <TableHead className="w-[10%] text-2xs py-1 h-8 text-right">
                Actions
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {stops.map((stop, idx) => {
              const isFirstOrLastStop = idx === 0 || idx === stops.length - 1;

              return (
                <TableRow key={stop.id} className="h-9">
                  {/* Sequence */}
                  <TableCell className="py-1 px-4 text-2xs">
                    <span className="text-xs text-muted-foreground">
                      {idx + 1}
                    </span>
                  </TableCell>

                  {/* Type */}
                  <TableCell className="py-1 px-4 text-2xs">
                    <div className="flex items-center gap-1">
                      {mapToStopType(stop.type)}
                    </div>
                  </TableCell>

                  {/* Address */}
                  <TableCell className="py-1 px-4 text-2xs truncate max-w-[200px]">
                    {stop.addressLine || "No address specified"}
                  </TableCell>

                  {/* Arrival */}
                  <TableCell className="py-1 px-4 text-2xs">
                    {generateDateTimeStringFromUnixTimestamp(
                      stop.plannedArrival,
                    )}
                  </TableCell>

                  {/* Departure */}
                  <TableCell className="py-1 px-4 text-2xs">
                    {generateDateTimeStringFromUnixTimestamp(
                      stop.plannedDeparture,
                    )}
                  </TableCell>

                  {/* Actions */}
                  <TableCell className="py-1 px-4 text-right">
                    <div className="flex items-center justify-end space-x-1">
                      <Button
                        size="icon"
                        variant="ghost"
                        onClick={() => onEdit(idx)}
                        className="h-6 w-6"
                        title="Edit stop"
                      >
                        <Icon
                          icon={faEdit}
                          className="size-3 text-muted-foreground"
                        />
                      </Button>

                      {!isFirstOrLastStop ? (
                        <Button
                          size="icon"
                          variant="ghost"
                          onClick={() => onDelete(idx)}
                          className="size-6 hover:bg-red-500/30 text-red-600 hover:text-red-600"
                          title="Delete stop"
                        >
                          <Icon icon={faTrash} className="size-4" />
                        </Button>
                      ) : (
                        <div
                          className="h-6 w-6 flex items-center justify-center"
                          title={
                            idx === 0
                              ? "Origin stop cannot be deleted"
                              : "Destination stop cannot be deleted"
                          }
                        >
                          <Icon
                            icon={faLock}
                            className="size-3 text-muted-foreground opacity-50"
                          />
                        </div>
                      )}
                    </div>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      </ScrollArea>
    </div>
  );
}
