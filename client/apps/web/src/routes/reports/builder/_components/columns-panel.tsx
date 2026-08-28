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
  bandableType,
  columnValueType,
  computedOperand,
  computedOperandIsTarget,
  withComputedOperand,
  type ReportColumnSpec,
  type ReportComputedOp,
  type ReportComputedOperand,
  type ReportComputedSide,
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
import { FilterIcon, GripVerticalIcon, PaletteIcon, SigmaIcon, XIcon } from "lucide-react";
import { AnimatePresence, m } from "motion/react";
import { useState } from "react";
import { BandEditor } from "./band-editor";
import { ColumnFormatEditor } from "./column-format-editor";
import { MeasureFilterEditor } from "./measure-filter-editor";
import {
  columnDisplayLabel,
  defaultColumnLabel,
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

/** Sentinel option that switches an operand from a measure to a constant. */
const TARGET_OPTION = "__target__";

/**
 * One side of a calculation. An operand is either another measure in the
 * report or a fixed target — the target is what makes variance-to-goal and
 * attainment expressible, since no column holds the goal.
 */
function ComputedOperandField({
  label,
  operand,
  measures,
  disallowTarget,
  defaultTarget,
  onChange,
}: {
  label: string;
  operand: ReportComputedOperand;
  measures: { value: string; label: string }[];
  disallowTarget: boolean;
  defaultTarget: number;
  onChange: (operand: ReportComputedOperand) => void;
}) {
  const isTarget = computedOperandIsTarget(operand);
  const choices = disallowTarget
    ? measures
    : [...measures, { value: TARGET_OPTION, label: "Target value…" }];

  return (
    <div className="flex flex-col gap-1">
      <Label className="text-muted-foreground text-xs">{label}</Label>
      <Select
        value={isTarget ? TARGET_OPTION : (operand.columnId ?? "")}
        onValueChange={(next) => {
          if (!next) return;
          onChange(
            next === TARGET_OPTION ? { value: operand.value ?? defaultTarget } : { columnId: next },
          );
        }}
        items={choices}
      >
        <SelectTrigger className="h-7">
          <SelectValue placeholder="Select measure" />
        </SelectTrigger>
        <SelectContent>
          {choices.map((choice) => (
            <SelectItem key={choice.value} value={choice.value}>
              {choice.label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
      {isTarget && (
        <Input
          className="h-7"
          type="number"
          step="any"
          placeholder="0"
          value={operand.value ?? ""}
          onChange={(event) => onChange({ value: Number(event.target.value) })}
        />
      )}
    </div>
  );
}

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

  const left = computedOperand(computed, "left");
  const right = computedOperand(computed, "right");
  const setOperand = (side: ReportComputedSide, operand: ReportComputedOperand) =>
    onUpdate({ ...column, computed: withComputedOperand(computed, side, operand) });

  // Zero is the natural starting target for a variance but a useless one for a
  // ratio, where it would only ever divide to nothing.
  const defaultTarget = computed.op === "divide" || computed.op === "multiply" ? 1 : 0;

  return (
    <div className="grid grid-cols-2 gap-2">
      <ComputedOperandField
        label="First value"
        operand={left}
        measures={operandChoices}
        // Two constants are a number, not a calculation, so the side opposite
        // a target only ever offers measures.
        disallowTarget={computedOperandIsTarget(right)}
        defaultTarget={defaultTarget}
        onChange={(operand) => setOperand("left", operand)}
      />
      <div className="flex flex-col gap-1">
        <Label className="text-muted-foreground text-xs">Operation</Label>
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
      <ComputedOperandField
        label="Second value"
        operand={right}
        measures={operandChoices}
        disallowTarget={computedOperandIsTarget(left)}
        defaultTarget={defaultTarget}
        onChange={(operand) => setOperand("right", operand)}
      />
      <div className="flex flex-col gap-1">
        <Label className="text-muted-foreground text-xs">Format</Label>
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
        <Label className="text-muted-foreground text-xs">Column name</Label>
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
  const [formatOpen, setFormatOpen] = useState(false);
  const [filterOpen, setFilterOpen] = useState(false);
  const sortable = useSortable({ id: column.id });
  const field = column.ref ? resolveField(index, ir.entity, column.ref) : undefined;
  const crossesToMany = column.ref ? pathCrossesToMany(index, ir.entity, column.ref.path) : false;
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
        "border-border bg-background flex flex-col gap-2 rounded-md border p-2",
        sortable.isDragging && "z-10 opacity-80 shadow-md",
      )}
    >
      <div className="flex items-center gap-1.5">
        <button
          type="button"
          className="text-muted-foreground hover:text-foreground cursor-grab"
          aria-label="Reorder column"
          {...sortable.attributes}
          {...sortable.listeners}
        >
          <GripVerticalIcon className="size-4" />
        </button>
        <span className="min-w-0 flex-1 truncate text-sm font-medium">
          {column.ref ? refLabel(index, ir.entity, column.ref) : (column.label ?? "Calculation")}
        </span>
        <Badge variant={isComputed ? "orange" : column.kind === "measure" ? "purple" : "info"}>
          {isComputed ? "calc" : column.kind}
        </Badge>
        {column.kind === "measure" && (
          <Button
            variant={filterOpen ? "secondary" : "ghost"}
            size="icon"
            className="size-6"
            onClick={() => setFilterOpen((open) => !open)}
            aria-label="Measure filter"
            aria-expanded={filterOpen}
            title="Only count matching records"
          >
            <FilterIcon className={cn("size-3.5", column.filter && "text-primary")} />
          </Button>
        )}
        <Button
          variant={formatOpen ? "secondary" : "ghost"}
          size="icon"
          className="size-6"
          onClick={() => setFormatOpen((open) => !open)}
          aria-label="Formatting options"
          aria-expanded={formatOpen}
          title="Formatting"
        >
          <PaletteIcon
            className={cn("size-3.5", (column.display || column.transform) && "text-primary")}
          />
        </Button>
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
            <Label className="text-muted-foreground text-xs">Kind</Label>
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
              <Label className="text-muted-foreground text-xs">Aggregation</Label>
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
              <Label className="text-muted-foreground text-xs">Bucket</Label>
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
            <Label className="text-muted-foreground text-xs">Column name</Label>
            <Input
              className="h-7"
              value={column.label ?? ""}
              placeholder={defaultColumnLabel(index, ir, column)}
              onChange={(event) => onUpdate({ ...column, label: event.target.value || undefined })}
            />
          </div>
          {column.kind === "dimension" && bandableType(field?.type) && (
            <BandEditor
              column={column}
              valueType={columnValueType(column, field?.type)}
              formatHint={field?.format ?? undefined}
              onUpdate={onUpdate}
            />
          )}
        </div>
      )}
      <AnimatePresence initial={false}>
        {filterOpen && column.kind === "measure" && (
          <m.div
            key="filter"
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.16, ease: "easeOut" }}
            className="overflow-hidden"
          >
            <MeasureFilterEditor index={index} ir={ir} column={column} onUpdate={onUpdate} />
          </m.div>
        )}
      </AnimatePresence>
      <AnimatePresence initial={false}>
        {formatOpen && (
          <m.div
            key="format"
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.16, ease: "easeOut" }}
            className="overflow-hidden"
          >
            <ColumnFormatEditor
              column={column}
              fieldType={field?.type}
              formatHint={isComputed ? column.computed?.format : (field?.format ?? undefined)}
              onUpdate={onUpdate}
            />
          </m.div>
        )}
      </AnimatePresence>
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
    // With a second measure the useful default is a ratio between the two;
    // with only one, it is that measure measured against a target.
    const computed: NonNullable<ReportColumnSpec["computed"]> =
      measures.length > 1
        ? { op: "divide", leftId: measures[0].id, rightId: measures[1].id }
        : { op: "subtract", leftId: measures[0].id, rightValue: 0 };
    onChange([
      ...ir.columns,
      {
        id: uniqueColumnId(ir, "calc"),
        kind: "computed",
        label: "Calculation",
        computed,
      },
    ]);
  };

  if (ir.columns.length === 0) {
    return (
      <p className="text-muted-foreground px-2 py-4 text-center text-sm">
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
