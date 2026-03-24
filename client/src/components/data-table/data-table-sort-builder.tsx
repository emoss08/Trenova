import { Button } from "@/components/ui/button";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { cn } from "@/lib/utils";
import type { SortDirection, SortField } from "@/types/data-table";
import {
  closestCenter,
  DndContext,
  DragOverlay,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from "@dnd-kit/core";
import { restrictToVerticalAxis } from "@dnd-kit/modifiers";
import {
  arrayMove,
  SortableContext,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import type { ColumnDef } from "@tanstack/react-table";
import {
  ArrowDownIcon,
  ArrowUpDownIcon,
  ArrowUpIcon,
  GripVerticalIcon,
  PlusIcon,
  TrashIcon,
} from "lucide-react";
import { useCallback, useMemo, useState } from "react";

type SortableColumn = {
  id: string;
  apiField: string;
  label: string;
};

type DataTableSortBuilderProps<TData> = {
  columns: ColumnDef<TData>[];
  sort: SortField[];
  onSortChange: (sort: SortField[]) => void;
};

export default function DataTableSortBuilder<TData>({
  columns,
  sort,
  onSortChange,
}: DataTableSortBuilderProps<TData>) {
  const [open, setOpen] = useState(false);
  const [activeId, setActiveId] = useState<string | null>(null);

  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: 8,
      },
    }),
    useSensor(KeyboardSensor),
  );

  const sortableColumns = useMemo<SortableColumn[]>(() => {
    return columns
      .filter((col) => {
        const meta = col.meta;
        return meta?.sortable !== false && meta?.apiField;
      })
      .map((col) => {
        const meta = col.meta!;
        return {
          id: String("accessorKey" in col ? col.accessorKey : col.id),
          apiField: meta.apiField!,
          label:
            meta.label ||
            String("accessorKey" in col ? col.accessorKey : col.id),
        };
      });
  }, [columns]);

  const handleDragStart = useCallback((event: DragStartEvent) => {
    setActiveId(event.active.id as string);
  }, []);

  const handleDragEnd = useCallback(
    (event: DragEndEvent) => {
      const { active, over } = event;
      if (over && active.id !== over.id) {
        const oldIndex = sort.findIndex((s) => s.field === active.id);
        const newIndex = sort.findIndex((s) => s.field === over.id);
        onSortChange(arrayMove(sort, oldIndex, newIndex));
      }
      setActiveId(null);
    },
    [sort, onSortChange],
  );

  const handleAddSort = useCallback(() => {
    const availableColumns = sortableColumns.filter(
      (col) => !sort.some((s) => s.field === col.apiField),
    );
    if (availableColumns.length === 0) return;

    const column = availableColumns[0];
    onSortChange([...sort, { field: column.apiField, direction: "asc" }]);
  }, [sortableColumns, sort, onSortChange]);

  const handleSortFieldChange = useCallback(
    (oldField: string, newField: string) => {
      onSortChange(
        sort.map((s) => (s.field === oldField ? { ...s, field: newField } : s)),
      );
    },
    [sort, onSortChange],
  );

  const handleSortDirectionChange = useCallback(
    (field: string, direction: SortDirection) => {
      onSortChange(
        sort.map((s) => (s.field === field ? { ...s, direction } : s)),
      );
    },
    [sort, onSortChange],
  );

  const handleSortRemove = useCallback(
    (field: string) => {
      onSortChange(sort.filter((s) => s.field !== field));
    },
    [sort, onSortChange],
  );

  const handleResetSort = useCallback(() => {
    onSortChange([]);
  }, [onSortChange]);

  const sortCount = sort.length;
  const sortIds = useMemo(() => sort.map((s) => s.field), [sort]);
  const activeSort = activeId ? sort.find((s) => s.field === activeId) : null;
  const activeSortIndex = activeId
    ? sort.findIndex((s) => s.field === activeId)
    : -1;

  const availableColumns = sortableColumns.filter(
    (col) => !sort.some((s) => s.field === col.apiField),
  );

  const getColumnLabel = (field: string) => {
    const col = sortableColumns.find((c) => c.apiField === field);
    return col?.label || field;
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={
          <Button variant="outline" size="sm" className="h-8">
            <ArrowUpDownIcon className="size-3.5" />
            Sort
            {sortCount > 0 && (
              <span className="ml-1.5 flex size-5 items-center justify-center rounded-md bg-muted font-mono text-xs">
                {sortCount}
              </span>
            )}
          </Button>
        }
      />
      <PopoverContent
        className={cn(
          "dark w-auto overflow-hidden p-0",
          sort.length === 0 && "min-w-[400px]",
        )}
        align="start"
      >
        {sort.length == 0 ? (
          <div className="flex flex-col items-start gap-3 p-3">
            <div className="flex flex-col items-start">
              <h3 className="text-xl font-semibold">No sorts applied</h3>
              <p className="text-sm text-muted-foreground">
                Add sorts to narrow down your results.
              </p>
            </div>
            <Button
              onClick={handleAddSort}
              disabled={availableColumns.length === 0}
            >
              <PlusIcon className="size-3.5" />
              Add Sort
            </Button>
          </div>
        ) : (
          <>
            <div className="flex flex-col gap-2 p-3">
              <DndContext
                sensors={sensors}
                collisionDetection={closestCenter}
                onDragStart={handleDragStart}
                onDragEnd={handleDragEnd}
                modifiers={[restrictToVerticalAxis]}
              >
                <SortableContext
                  items={sortIds}
                  strategy={verticalListSortingStrategy}
                >
                  {sort.map((sortField, index) => (
                    <SortableSortRow
                      key={sortField.field}
                      sortField={sortField}
                      index={index}
                      columns={sortableColumns}
                      usedFields={sort.map((s) => s.field)}
                      onFieldChange={handleSortFieldChange}
                      onDirectionChange={handleSortDirectionChange}
                      onRemove={handleSortRemove}
                      getColumnLabel={getColumnLabel}
                    />
                  ))}
                </SortableContext>
                <DragOverlay>
                  {activeSort && (
                    <SortRowOverlay
                      sortField={activeSort}
                      index={activeSortIndex}
                      getColumnLabel={getColumnLabel}
                    />
                  )}
                </DragOverlay>
              </DndContext>
            </div>
            <div className="flex items-center gap-2 border-t bg-sidebar p-2 dark:bg-background">
              <Button
                variant="outline"
                size="sm"
                onClick={handleAddSort}
                disabled={availableColumns.length === 0}
              >
                <PlusIcon className="size-3.5" />
                Add Sort
              </Button>
              {sort.length > 0 && (
                <Button variant="ghost" size="sm" onClick={handleResetSort}>
                  Reset Sort
                </Button>
              )}
            </div>
          </>
        )}
      </PopoverContent>
    </Popover>
  );
}

type SortableSortRowProps = {
  sortField: SortField;
  index: number;
  columns: SortableColumn[];
  usedFields: string[];
  onFieldChange: (oldField: string, newField: string) => void;
  onDirectionChange: (field: string, direction: SortDirection) => void;
  onRemove: (field: string) => void;
  getColumnLabel: (field: string) => string;
};

function SortableSortRow({
  sortField,
  index,
  columns,
  usedFields,
  onFieldChange,
  onDirectionChange,
  onRemove,
}: SortableSortRowProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({
    id: sortField.field,
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
  };

  const availableForThis = columns.filter(
    (col) =>
      col.apiField === sortField.field || !usedFields.includes(col.apiField),
  );

  return (
    <div
      ref={setNodeRef}
      style={style}
      className={cn(
        "flex items-center gap-2 rounded-md py-1",
        isDragging && "opacity-50",
      )}
    >
      <span className="w-10 shrink-0 text-sm text-muted-foreground">
        {index === 0 ? "By" : "Then"}
      </span>

      <Select
        value={sortField.field}
        onValueChange={(val) => onFieldChange(sortField.field, val ?? "")}
        items={availableForThis.map((col) => ({
          value: col.apiField,
          label: col.label,
        }))}
      >
        <SelectTrigger size="sm" className="w-36">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {availableForThis.map((col) => (
            <SelectItem key={col.apiField} value={col.apiField}>
              {col.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Select
        value={sortField.direction}
        onValueChange={(val) =>
          onDirectionChange(sortField.field, val as SortDirection)
        }
        items={[
          { value: "asc", label: "Ascending" },
          { value: "desc", label: "Descending" },
        ]}
      >
        <SelectTrigger size="sm" className="w-36">
          <SelectValue>
            <span className="flex items-center gap-2">
              {sortField.direction === "asc" ? (
                <>
                  <ArrowUpIcon className="size-3.5" />
                  Ascending
                </>
              ) : (
                <>
                  <ArrowDownIcon className="size-3.5" />
                  Descending
                </>
              )}
            </span>
          </SelectValue>
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="asc">
            <span className="flex items-center gap-2">
              <ArrowUpIcon className="size-3.5" />
              Ascending
            </span>
          </SelectItem>
          <SelectItem value="desc">
            <span className="flex items-center gap-2">
              <ArrowDownIcon className="size-3.5" />
              Descending
            </span>
          </SelectItem>
        </SelectContent>
      </Select>

      <Button
        variant="ghost"
        size="icon"
        className="size-7 text-muted-foreground hover:text-destructive"
        onClick={() => onRemove(sortField.field)}
      >
        <TrashIcon className="size-4" />
      </Button>

      <Button
        variant="ghost"
        size="icon"
        className="size-7 cursor-grab touch-none text-muted-foreground"
        {...attributes}
        {...listeners}
      >
        <GripVerticalIcon className="size-4" />
      </Button>
    </div>
  );
}

type SortRowOverlayProps = {
  sortField: SortField;
  index: number;
  getColumnLabel: (field: string) => string;
};

function SortRowOverlay({
  sortField,
  index,
  getColumnLabel,
}: SortRowOverlayProps) {
  return (
    <div className="flex items-center gap-2 rounded-md border bg-popover px-2 py-1 shadow-lg">
      <span className="w-10 shrink-0 text-sm text-muted-foreground">
        {index === 0 ? "By" : "Then"}
      </span>

      <div className="flex h-7 w-36 items-center rounded-md border bg-muted px-2 text-sm">
        {getColumnLabel(sortField.field)}
      </div>

      <div className="flex h-7 w-36 items-center gap-2 rounded-md border bg-muted px-2 text-sm">
        {sortField.direction === "asc" ? (
          <>
            <ArrowUpIcon className="size-3.5" />
            Ascending
          </>
        ) : (
          <>
            <ArrowDownIcon className="size-3.5" />
            Descending
          </>
        )}
      </div>

      <div className="size-7" />

      <div className="flex size-7 items-center justify-center text-muted-foreground">
        <GripVerticalIcon className="size-4" />
      </div>
    </div>
  );
}
