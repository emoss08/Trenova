/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import * as React from "react";

import { cn } from "@/lib/utils";

interface TableProps extends React.ComponentProps<"table"> {
  containerClassName?: string;
}

function Table({ containerClassName, className, ...props }: TableProps) {
  return (
    <div data-slot="table-container" className={containerClassName}>
      <table
        data-slot="table"
        className={cn(
          "-my-1 w-full caption-bottom border-separate border-spacing-y-1 text-xs max-md:flex max-md:w-full max-md:flex-col max-md:py-2",
          className,
        )}
        {...props}
      />
    </div>
  );
}

function TableHeader({ className, ...props }: React.ComponentProps<"thead">) {
  return (
    <thead
      data-slot="table-header"
      className={cn(
        "z-20 rounded-md backdrop-blur-md md:sticky md:top-0 before:absolute before:inset-x-0 before:-top-[2px] before:-z-10 before:h-[5px] before:bg-background max-md:hidden",
        className,
      )}
      {...props}
    />
  );
}

function TableBody({ className, ...props }: React.ComponentProps<"tbody">) {
  return (
    <tbody
      data-slot="table-body"
      className={cn(
        "pb-4 md:pb-12 content-visibility-auto contain-intrinsic-size-auto max-md:flex max-md:w-full max-md:flex-col max-md:gap-4",
        className,
      )}
      {...props}
    />
  );
}

function TableFooter({ className, ...props }: React.ComponentProps<"tfoot">) {
  return (
    <tfoot
      data-slot="table-footer"
      className={cn(
        "border-t bg-muted/50 font-medium [&>tr]:last:border-b-0",
        className,
      )}
      {...props}
    />
  );
}

function TableRow({ className, ...props }: React.ComponentProps<"tr">) {
  return (
    <tr
      data-slot="table-row"
      className={cn(
        "group/row whitespace-nowrap md:[&_td:first-child]:border-l md:[&_td:last-child]:border-r md:[&_td]:border-y max-md:flex max-md:w-full max-md:flex-col bg-card [&:hover_td]:md:bg-muted-foreground/10 [&_td]:md:border-border max-md:overflow-hidden max-md:rounded-lg max-md:border content-visibility-auto",
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
        "h-10 px-2 text-left align-middle font-mono font-medium uppercase text-muted-foreground first:rounded-l-md first:pl-5 last:rounded-r-md last:pr-5 md:px-3.5 [&:has([role=checkbox])]:pr-0 first:border-l last:border-r border-y border-border max-md:hidden",
        className,
      )}
      {...props}
    />
  );
}

// Memoize TableCell to prevent unnecessary re-renders
const TableCell = React.memo(function TableCellInner({
  className,
  children,
  ...props
}: React.ComponentProps<"td">) {
  return (
    <td
      data-slot="table-cell"
      className={cn(
        "h-11 px-2 py-2 align-middle font-mono first:pl-5 last:rounded-r-md last:pr-5 [&:has([role=checkbox])]:pr-0 max-md:first:rounded-y-lg max-md max-md:flex max-md:h-9 max-md:w-full max-md:items-center max-md:justify-between max-md:gap-3 max-md:border-t max-md:!px-3 max-md:text-right max-md:first:border-t-0 md:px-3.5 md:first:rounded-l-md overflow-hidden md:max-w-px",
        className,
      )}
      {...props}
    >
      {children}
    </td>
  );
});
TableCell.displayName = "TableCell";

function TableCaption({
  className,
  ...props
}: React.ComponentProps<"caption">) {
  return (
    <caption
      data-slot="table-caption"
      className={cn("mt-4 text-sm text-muted-foreground", className)}
      {...props}
    />
  );
}

export {
  Table,
  TableBody,
  TableCaption,
  TableCell,
  TableFooter,
  TableHead,
  TableHeader,
  TableRow
};

