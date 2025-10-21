import { Label } from "@/components/ui/label";
import { SortableDragHandle } from "@/components/ui/sortable";
import { cn, toTitleCase } from "@/lib/utils";
import { Column } from "@tanstack/react-table";
import { Check, GripVertical } from "lucide-react";

export function ColumnSortItem({
  column,
  searchQuery,
  drag,
}: {
  column: Column<any, any>;
  searchQuery: string;
  drag: boolean;
}) {
  return (
    <ColumnSortItemOuter>
      <ColumnSortItemContent column={column} />
      {!searchQuery && (
        <SortableDragHandle
          variant="ghost"
          size="icon"
          className="size-6 text-muted-foreground hover:text-foreground"
          disabled={drag}
        >
          <GripVertical className="size-3" aria-hidden="true" />
        </SortableDragHandle>
      )}
    </ColumnSortItemOuter>
  );
}

function ColumnSortItemOuter({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-2 px-2 py-1.5 rounded hover:bg-muted/50">
      {children}
    </div>
  );
}

function ColumnSortItemcContentOuter({
  children,
}: {
  children: React.ReactNode;
}) {
  return <div className="flex items-center gap-2 flex-1">{children}</div>;
}

function ColumnSortItemContent({ column }: { column: Column<any, any> }) {
  return (
    <ColumnSortItemcContentOuter>
      <div
        className={cn(
          "flex h-4 w-4 items-center justify-center rounded-sm border border-primary",
          column.getIsVisible()
            ? "bg-primary text-primary-foreground"
            : "opacity-50 [&_svg]:invisible",
        )}
        onClick={() => column.toggleVisibility(!column.getIsVisible())}
      >
        <Check className={cn("h-4 w-4")} />
      </div>
      <Label
        htmlFor={column.id}
        className="flex-1 text-xs cursor-pointer"
        onClick={() => column.toggleVisibility(!column.getIsVisible())}
      >
        {toTitleCase(column.id)}
      </Label>
    </ColumnSortItemcContentOuter>
  );
}
