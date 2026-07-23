import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import { Label } from "@trenova/shared/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import { cn } from "@trenova/shared/lib/utils";
import {
  REPORT_AGGREGATION_LABELS,
  REPORT_COMPUTED_FORMAT_CHOICES,
  REPORT_COMPUTED_OP_CHOICES,
  REPORT_DATE_BUCKET_CHOICES,
  aggregationsForField,
  type ReportColumnSpec,
  type ReportComputedOp,
  type ReportDateBucket,
  type ReportIR,
} from "@/types/report";
import {
  closestCenter,
  DndContext,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import { restrictToVerticalAxis } from "@dnd-kit/modifiers";
import {
  arrayMove,
  SortableContext,
  useSortable,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { GripVerticalIcon, SigmaIcon, XIcon } from "lucide-react";
import {
  columnDisplayLabel,
  measureColumns,
  pathCrossesToMany,
  refLabel,
  resolveField,
  uniqueColumnId,
  type CatalogIndex,
} from "./builder-state";

const BUCKET_CHOICES = [{ value: "none", label: "Exact" }, ...REPORT_DATE_BUCKET_CHOICES];

type ColumnsPanelProps = {
  index: CatalogIndex;
  ir: ReportIR;
  onChange: (columns: ReportColumnSpec[]) => void;
};

function ComputedColumnBody({
  ir,
  index,
  column,
  onUpdate,
}: {
  ir: ReportIR;
  index: CatalogIndex;
  column: ReportColumnSpec;
  onUpdate: (column: ReportColumnSpec) => void;
}) {
  const computed = column.computed;
  if (!computed) return null;

  const operandChoices = measureColumns(ir).map((measure) => ({
    value: measure.id,
    label: columnDisplayLabel(index, ir, measure),
  }));
  const updateComputed = (patch: Partial<NonNullable<ReportColumnSpec["computed"]>>) =>
    onUpdate({ ...column, computed: { ...computed, ...patch } });

  return (
    <div className="grid grid-cols-2 gap-2">
      <div className="flex flex-col gap-1">
        <Label className="text-xs text-muted-foreground">First measure</Label>
        <Select
          value={computed.leftId}
          onValueChange={(leftId) => {
            if (leftId) updateComputed({ leftId });
          }}
          items={operandChoices}
        >
          <SelectTrigger className="h-7">
            <SelectValue placeholder="Select measure" />
          </SelectTrigger>
          <SelectContent>
            {operandChoices.map((choice) => (
              <SelectItem key={choice.value} value={choice.value}>
                {choice.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="flex flex-col gap-1">
        <Label className="text-xs text-muted-foreground">Operation</Label>
        <Select
          value={computed.op}
          onValueChange={(op) => {
            if (op) updateComputed({ op: op as ReportComputedOp });
          }}
          items={REPORT_COMPUTED_OP_CHOICES}
        >
          <SelectTrigger className="h-7">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {REPORT_COMPUTED_OP_CHOICES.map((choice) => (
              <SelectItem key={choice.value} value={choice.value}>
                {choice.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="flex flex-col gap-1">
        <Label className="text-xs text-muted-foreground">Second measure</Label>
        <Select
          value={computed.rightId}
          onValueChange={(rightId) => {
            if (rightId) updateComputed({ rightId });
          }}
          items={operandChoices}
        >
          <SelectTrigger className="h-7">
            <SelectValue placeholder="Select measure" />
          </SelectTrigger>
          <SelectContent>
            {operandChoices.map((choice) => (
              <SelectItem key={choice.value} value={choice.value}>
                {choice.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="flex flex-col gap-1">
        <Label className="text-xs text-muted-foreground">Format</Label>
        <Select
          value={computed.format ?? "none"}
          onValueChange={(format) => {
            if (format) updateComputed({ format });
          }}
          items={REPORT_COMPUTED_FORMAT_CHOICES}
        >
          <SelectTrigger className="h-7">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {REPORT_COMPUTED_FORMAT_CHOICES.map((choice) => (
              <SelectItem key={choice.value} value={choice.value}>
                {choice.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <div className="col-span-2 flex flex-col gap-1">
        <Label className="text-xs text-muted-foreground">Label</Label>
        <Input
          className="h-7"
          value={column.label ?? ""}
          placeholder="Revenue per mile"
          onChange={(event) => onUpdate({ ...column, label: event.target.value || undefined })}
        />
      </div>
    </div>
  );
}

function SortableColumnRow({
  index,
  ir,
  column,
  onUpdate,
  onRemove,
}: {
  index: CatalogIndex;
  ir: ReportIR;
  column: ReportColumnSpec;
  onUpdate: (column: ReportColumnSpec) => void;
  onRemove: () => void;
}) {
  const sortable = useSortable({ id: column.id });
  const field = column.ref ? resolveField(index, ir.entity, column.ref) : undefined;
  const crossesToMany = column.ref
    ? pathCrossesToMany(index, ir.entity, column.ref.path)
    : false;
  const aggregations = field ? aggregationsForField(field) : [];
  const canBeDimension = !crossesToMany;
  const canBeMeasure = aggregations.length > 0;
  const kindChoices = [
    ...(canBeDimension ? [{ value: "dimension", label: "Dimension" }] : []),
    ...(canBeMeasure ? [{ value: "measure", label: "Measure" }] : []),
  ];
  const isComputed = column.kind === "computed";

  return (
    <div
      ref={sortable.setNodeRef}
      style={{
        transform: CSS.Transform.toString(sortable.transform),
        transition: sortable.transition,
      }}
      className={cn(
        "flex flex-col gap-2 rounded-md border border-border bg-background p-2",
        sortable.isDragging && "z-10 opacity-80 shadow-md",
      )}
    >
      <div className="flex items-center gap-1.5">
        <button
          type="button"
          className="cursor-grab text-muted-foreground hover:text-foreground"
          aria-label="Reorder column"
          {...sortable.attributes}
          {...sortable.listeners}
        >
          <GripVerticalIcon className="size-4" />
        </button>
        <span className="min-w-0 flex-1 truncate text-sm font-medium">
          {column.ref ? refLabel(index, ir.entity, column.ref) : (column.label ?? "Calculation")}
        </span>
        <Badge
          variant={isComputed ? "orange" : column.kind === "measure" ? "purple" : "info"}
        >
          {isComputed ? "calc" : column.kind}
        </Badge>
        <Button
          variant="ghost"
          size="icon"
          className="size-6"
          onClick={onRemove}
          aria-label="Remove column"
        >
          <XIcon className="size-3.5" />
        </Button>
      </div>
      {isComputed ? (
        <ComputedColumnBody ir={ir} index={index} column={column} onUpdate={onUpdate} />
      ) : (
      <div className="grid grid-cols-2 gap-2">
        <div className="flex flex-col gap-1">
          <Label className="text-xs text-muted-foreground">Kind</Label>
          <Select
            value={column.kind}
            onValueChange={(kind) => {
              if (!kind || kind === column.kind) return;
              if (kind === "measure") {
                onUpdate({ ...column, kind: "measure", agg: aggregations[0], bucket: undefined });
              } else {
                onUpdate({ ...column, kind: "dimension", agg: undefined });
              }
            }}
            items={kindChoices}
          >
            <SelectTrigger className="h-7">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {canBeDimension && <SelectItem value="dimension">Dimension</SelectItem>}
              {canBeMeasure && <SelectItem value="measure">Measure</SelectItem>}
            </SelectContent>
          </Select>
        </div>
        {column.kind === "measure" && (
          <div className="flex flex-col gap-1">
            <Label className="text-xs text-muted-foreground">Aggregation</Label>
            <Select
              value={column.agg ?? ""}
              onValueChange={(agg) => {
                if (agg) onUpdate({ ...column, agg: agg as ReportColumnSpec["agg"] });
              }}
              items={aggregations.map((agg) => ({
                value: agg,
                label: REPORT_AGGREGATION_LABELS[agg],
              }))}
            >
              <SelectTrigger className="h-7">
                <SelectValue placeholder="Select" />
              </SelectTrigger>
              <SelectContent>
                {aggregations.map((agg) => (
                  <SelectItem key={agg} value={agg}>
                    {REPORT_AGGREGATION_LABELS[agg]}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
        {column.kind === "dimension" && field?.type === "epoch" && (
          <div className="flex flex-col gap-1">
            <Label className="text-xs text-muted-foreground">Bucket</Label>
            <Select
              value={column.bucket ?? "none"}
              onValueChange={(bucket) => {
                if (!bucket) return;
                onUpdate({
                  ...column,
                  bucket: bucket === "none" ? undefined : (bucket as ReportDateBucket),
                });
              }}
              items={BUCKET_CHOICES}
            >
              <SelectTrigger className="h-7">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="none">Exact</SelectItem>
                {REPORT_DATE_BUCKET_CHOICES.map((choice) => (
                  <SelectItem key={choice.value} value={choice.value}>
                    {choice.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        )}
        <div
          className={cn(
            "flex flex-col gap-1",
            column.kind === "dimension" && field?.type !== "epoch" ? "" : "col-span-2",
          )}
        >
          <Label className="text-xs text-muted-foreground">Label</Label>
          <Input
            className="h-7"
            value={column.label ?? ""}
            placeholder={field?.label ?? column.ref?.field}
            onChange={(event) => onUpdate({ ...column, label: event.target.value || undefined })}
          />
        </div>
      </div>
      )}
    </div>
  );
}

export function ColumnsPanel({ index, ir, onChange }: ColumnsPanelProps) {
  const sensors = useSensors(useSensor(PointerSensor), useSensor(KeyboardSensor));

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;
    if (!over || active.id === over.id) return;
    const oldIndex = ir.columns.findIndex((column) => column.id === active.id);
    const newIndex = ir.columns.findIndex((column) => column.id === over.id);
    if (oldIndex < 0 || newIndex < 0) return;
    onChange(arrayMove(ir.columns, oldIndex, newIndex));
  };

  const addCalculation = () => {
    const measures = measureColumns(ir);
    if (measures.length === 0) return;
    onChange([
      ...ir.columns,
      {
        id: uniqueColumnId(ir, "calc"),
        kind: "computed",
        label: "Calculation",
        computed: {
          op: "divide",
          leftId: measures[0].id,
          rightId: (measures[1] ?? measures[0]).id,
        },
      },
    ]);
  };

  if (ir.columns.length === 0) {
    return (
      <p className="px-2 py-4 text-center text-sm text-muted-foreground">
        Add fields from the catalog to define the report&apos;s columns.
      </p>
    );
  }

  return (
    <div className="flex flex-col gap-2">
      <DndContext
        sensors={sensors}
        collisionDetection={closestCenter}
        modifiers={[restrictToVerticalAxis]}
        onDragEnd={handleDragEnd}
      >
        <SortableContext
          items={ir.columns.map((column) => column.id)}
          strategy={verticalListSortingStrategy}
        >
          <div className="flex flex-col gap-2">
            {ir.columns.map((column) => (
              <SortableColumnRow
                key={column.id}
                index={index}
                ir={ir}
                column={column}
                onUpdate={(updated) =>
                  onChange(ir.columns.map((c) => (c.id === column.id ? updated : c)))
                }
                onRemove={() => onChange(ir.columns.filter((c) => c.id !== column.id))}
              />
            ))}
          </div>
        </SortableContext>
      </DndContext>
      {measureColumns(ir).length > 0 && (
        <Button variant="outline" size="sm" className="h-7 self-start" onClick={addCalculation}>
          <SigmaIcon className="size-3.5" />
          Calculation
        </Button>
      )}
    </div>
  );
}
