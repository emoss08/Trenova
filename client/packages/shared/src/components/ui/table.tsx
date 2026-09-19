import { cn } from "@trenova/shared/lib/utils";
import * as React from "react";
import { ScrollArea, ScrollBar } from "./scroll-area";

interface TableProps extends React.ComponentProps<"table"> {
  containerClassName?: string;
  maskHeight?: number;
}

function Table({ className, containerClassName, maskHeight, ...props }: TableProps) {
  return (
    <ScrollArea
      data-slot="table-container"
      className={cn("w-full", containerClassName)}
      maskHeight={maskHeight}
    >
      <table
        data-slot="table"
        className={cn("w-full caption-bottom text-sm", className)}
        {...props}
      />
      <ScrollBar orientation="horizontal" />
    </ScrollArea>
  );
}

function TableHeader({ className, ...props }: React.ComponentProps<"thead">) {
  return <thead data-slot="table-header" className={cn("[&_tr]:border-b", className)} {...props} />;
}

function TableBody({ className, ...props }: React.ComponentProps<"tbody">) {
  return (
    <tbody
      data-slot="table-body"
      className={cn("[&_tr:last-child]:border-0", className)}
      {...props}
    />
  );
}

function TableFooter({ className, ...props }: React.ComponentProps<"tfoot">) {
  return (
    <tfoot
      data-slot="table-footer"
      className={cn("border-t bg-muted/50 font-medium [&>tr]:last:border-b-0", className)}
      {...props}
    />
  );
}

/* Row height comes from --row-h rather than from whatever padding a cell
   happens to carry, so two tables on the same screen line up. A caller that
   wants a denser table repoints the token (`[--row-h:var(--row-h-compact)]`)
   instead of patching padding onto every cell. */
function TableRow({ className, ...props }: React.ComponentProps<"tr">) {
  return (
    <tr
      data-slot="table-row"
      className={cn(
        "h-(--row-h) border-b transition-colors hover:bg-surface-hover data-[state=selected]:bg-surface-selected",
        className,
      )}
      {...props}
    />
  );
}

function TableHead({ className, ...props }: React.ComponentProps<"th">) {
  return (
    <th
      data-slot="table-head"
      className={cn(
        "h-(--row-head-h) bg-sunken px-(--cell-px) text-left align-middle font-table text-2xs font-medium tracking-wider whitespace-nowrap text-foreground-subtle uppercase [&:has([role=checkbox])]:pr-0 [&>[role=checkbox]]:translate-y-[2px]",
        className,
      )}
      {...props}
    />
  );
}

function TableCell({ className, ...props }: React.ComponentProps<"td">) {
  return (
    <td
      data-slot="table-cell"
      className={cn(
        "px-(--cell-px) py-1 align-middle whitespace-nowrap [&:has([role=checkbox])]:pr-0 [&>[role=checkbox]]:translate-y-[2px]",
        className,
      )}
      {...props}
    />
  );
}

function TableCaption({ className, ...props }: React.ComponentProps<"caption">) {
  return (
    <caption
      data-slot="table-caption"
      className={cn("mt-4 text-sm text-muted-foreground", className)}
      {...props}
    />
  );
}

export { Table, TableBody, TableCaption, TableCell, TableFooter, TableHead, TableHeader, TableRow };
