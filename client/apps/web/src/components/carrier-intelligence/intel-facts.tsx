import { INTEL_EMPTY_VALUE, formatOptionalDecimalCurrency } from "@/lib/carrier-intelligence";
import { formatNumber } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { cn, formatPercent } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";

export function FactGrid({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <dl className={cn("grid grid-cols-1 gap-x-6 gap-y-3 sm:grid-cols-2", className)}>{children}</dl>
  );
}

export function Fact({
  label,
  children,
  className,
  wrap = false,
}: {
  label: ReactNode;
  children: ReactNode;
  className?: string;
  wrap?: boolean;
}) {
  return (
    <div className={cn("flex min-w-0 flex-col gap-0.5", className)}>
      <dt className="text-muted-foreground text-xs">{label}</dt>
      <dd className={cn("text-sm tabular-nums", wrap ? "break-words" : "truncate")}>{children}</dd>
    </div>
  );
}

export function Muted({ children }: { children: ReactNode }) {
  return <span className="text-muted-foreground">{children}</span>;
}

export function EmptyValue() {
  return <Muted>{INTEL_EMPTY_VALUE}</Muted>;
}

export function BooleanValue({ value }: { value: boolean | null | undefined }) {
  const t = useT();
  if (value === null || value === undefined) {
    return <EmptyValue />;
  }
  return <>{value ? t("Yes") : t("No")}</>;
}

export function textOrDash(value: string | null | undefined): ReactNode {
  return value && value.trim() !== "" ? value : <EmptyValue />;
}

export function numberOrDash(
  value: number | null | undefined,
  options?: Intl.NumberFormatOptions,
): ReactNode {
  return value === null || value === undefined ? <EmptyValue /> : formatNumber(value, options);
}

export function dateOrDash(value: number | null | undefined): ReactNode {
  return value ? formatUnixDateMedium(value) : <EmptyValue />;
}

export function currencyOrDash(value: string | null | undefined): ReactNode {
  return formatOptionalDecimalCurrency(value) ?? <EmptyValue />;
}

export function percentOrDash(value: number | null | undefined): ReactNode {
  return value === null || value === undefined ? <EmptyValue /> : formatPercent(value);
}

export function listOrDash(values: readonly string[] | null | undefined): ReactNode {
  return values && values.length > 0 ? values.join(", ") : <EmptyValue />;
}

export function SectionNote({ children }: { children: ReactNode }) {
  return <p className="text-muted-foreground text-xs">{children}</p>;
}

export function SubHeading({ children }: { children: ReactNode }) {
  return <h4 className="text-muted-foreground text-xs font-medium">{children}</h4>;
}

export type MiniTableColumn = { id: string; label: ReactNode; align?: "right" };
export type MiniTableRow = { id: string; cells: ReactNode[] };

export function MiniTable({ columns, rows }: { columns: MiniTableColumn[]; rows: MiniTableRow[] }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b">
            {columns.map((column) => (
              <th
                key={column.id}
                scope="col"
                className={cn(
                  "text-muted-foreground h-8 px-2 text-left text-xs font-normal whitespace-nowrap first:pl-0 last:pr-0",
                  column.align === "right" && "text-right",
                )}
              >
                {column.label}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((row) => (
            <tr key={row.id} className="border-b last:border-b-0">
              {row.cells.map((cell, index) => (
                <td
                  key={columns[index]?.id ?? index}
                  className={cn(
                    "h-9 px-2 align-middle tabular-nums first:pl-0 last:pr-0",
                    columns[index]?.align === "right" && "text-right",
                  )}
                >
                  {cell}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
