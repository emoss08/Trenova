import { useRules } from "@trenova/shared/hooks/use-rules";
import { formatList } from "@trenova/shared/i18n/format";
import { translateMessage } from "@trenova/shared/i18n/message";
import { useT } from "@trenova/shared/i18n/use-t";
import { matrixCoverage } from "@trenova/shared/rules/matrix";
import { rateMatrix } from "@/lib/queries/rate-matrix";
import { apiService } from "@/services/api";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Input } from "@trenova/shared/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@trenova/shared/components/ui/select";
import {
  addPostalRange,
  axisLabel,
  buildMatrixGrid,
  cellCoordinate,
  coordinateKey,
  isPostalRangeAxis,
  matrixValueUnit,
  postalRangesOf,
  rerangePostalAxis,
  sliceBuckets,
  withoutPostalRange,
  type MatrixAxisPosition,
  type PostalRange,
} from "@trenova/shared/lib/rate-matrix";
import type {
  RateMatrix,
  RateMatrixCell,
  RateMatrixDimension,
  RateMatrixValueType,
} from "@trenova/shared/types/rate";
import { AlertCircleIcon, Save01Icon, SpinnerIcon } from "@trenova/shared/components/icons";
import { useCallback, useMemo, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { MatrixGridEditorSkeleton } from "./matrix-grid-editor-skeleton";
import { PostalRangeEditor } from "./postal-range-editor";
import { toast } from "@trenova/shared/components/ui/toast";
import { cn } from "@trenova/shared/lib/utils";

function orderByPosition(dimensions: RateMatrixDimension[]): RateMatrixDimension[] {
  return [...dimensions].sort((a, b) => a.position - b.position);
}

/** A published version to read instead of the draft: its axes and value type, read-only. */
export type MatrixGridVersion = {
  readonly versionNumber: number;
  readonly dimensions: readonly RateMatrixDimension[];
  readonly valueType: RateMatrixValueType;
};

type MatrixGridEditorProps = {
  /** Absent while the matrix is being created — there is nothing to hang cells off yet. */
  readonly rateMatrixId?: string;
  /** A published version to show read-only; absent edits the draft. */
  readonly version?: MatrixGridVersion;
};

/** The drafts moved to the coordinates the same cells have after an edit to the axes. */
function moveDrafts(
  drafts: Record<string, string>,
  before: readonly RateMatrixCell[],
  after: readonly RateMatrixCell[],
  dimensions: readonly RateMatrixDimension[],
): Record<string, string> {
  const moved: Record<string, string> = {};
  before.forEach((cell, index) => {
    const from = cellCoordinate(cell, dimensions);
    const next = after[index];
    if (next && drafts[from] !== undefined) moved[cellCoordinate(next, dimensions)] = drafts[from];
  });
  return moved;
}

/**
 * The tariff sheet.
 *
 * Rates live in their own resource rather than in the form, because a
 * four-axis class tariff runs to tens of thousands of cells and a form field
 * array would re-render all of them on every keystroke. The trade is that this
 * saves separately from the matrix header, which the button says out loud.
 */
/** Whether a cell's text is a value of the matrix's type: a number, and whole for an `Int` matrix. */
function cellTextFits(text: string, wholeOnly: boolean): boolean {
  const parsed = Number(text);
  if (text.trim() === "" || !Number.isFinite(parsed)) return false;
  return !wholeOnly || Number.isInteger(parsed);
}

export function MatrixGridEditor({ rateMatrixId, version }: MatrixGridEditorProps) {
  const t = useT();
  const rules = useRules();

  const { control } = useFormContext<RateMatrix>();
  const watchedDimensions = (useWatch({ control, name: "dimensions" }) ??
    []) as RateMatrixDimension[];
  const watchedValueType = useWatch({ control, name: "valueType" }) ?? "Money";
  const readOnly = version !== undefined;
  const dimensions = version ? [...version.dimensions] : watchedDimensions;
  const valueType = version?.valueType ?? watchedValueType;
  const valueUnit = matrixValueUnit(valueType);
  const wholeOnly = valueType === "Int";
  const queryClient = useQueryClient();

  const { data: loadedCells, isLoading } = useQuery({
    ...rateMatrix.cells(rateMatrixId ?? "", version?.versionNumber ?? null),
    enabled: Boolean(rateMatrixId),
  });

  const [cells, setCells] = useState<RateMatrixCell[]>([]);
  const [dirty, setDirty] = useState(false);

  // Text being typed is kept apart from the parsed number, because "2." and
  // "-" are states somebody passes through on the way to a valid rate and
  // coercing them on every keystroke would fight the person entering them.
  const [drafts, setDrafts] = useState<Record<string, string>>({});

  const [prevLoadedCells, setPrevLoadedCells] = useState<RateMatrixCell[] | undefined>(undefined);

  if (loadedCells !== prevLoadedCells) {
    setPrevLoadedCells(loadedCells);
    if (loadedCells) {
      setCells(loadedCells);
      setDrafts({});
      setDirty(false);
    }
  }

  // useWatch hands back a fresh array every render, so memoizing on it would
  // memoize nothing and every grid below would rebuild on each keystroke. What
  // actually changes is the shape of the axes, so that is what is compared.
  const signature = dimensions
    .map((dimension) => `${dimension.position}:${dimension.kind}:${dimension.matchMode}`)
    .join("|");
  const [axes, setAxes] = useState(() => ({
    signature,
    ordered: orderByPosition(dimensions),
  }));

  if (axes.signature !== signature) {
    setAxes({ signature, ordered: orderByPosition(dimensions) });
  }

  const ordered = axes.ordered;

  const rowPosition = (ordered[0]?.position ?? 0) as MatrixAxisPosition;
  const columnDimension = ordered[1];
  const columnPosition = (columnDimension?.position ?? null) as MatrixAxisPosition | null;
  const sliceDimensions = useMemo(() => ordered.slice(2), [ordered]);

  const [slice, setSlice] = useState<Record<number, string>>({});

  // A sheet selector pointing at a band that no longer exists renders an empty
  // grid that looks like missing data. Snapping to the first available bucket
  // keeps the editor showing something real.
  const [sliceSource, setSliceSource] = useState<{
    cells: RateMatrixCell[];
    sliceDimensions: RateMatrixDimension[];
  } | null>(null);

  if (sliceSource?.cells !== cells || sliceSource.sliceDimensions !== sliceDimensions) {
    setSliceSource({ cells, sliceDimensions });

    let changed = false;
    const next = { ...slice };

    for (const dimension of sliceDimensions) {
      const buckets = sliceBuckets(dimension, cells);
      const chosen = next[dimension.position];

      if (buckets.length === 0) continue;
      if (chosen !== undefined && buckets.some((bucket) => bucket.key === chosen)) continue;

      next[dimension.position] = buckets[0]?.key ?? "";
      changed = true;
    }

    if (changed) setSlice(next);
  }

  const grid = useMemo(
    () =>
      buildMatrixGrid({
        dimensions: ordered,
        cells,
        rowPosition,
        columnPosition,
        slice,
      }),
    [ordered, cells, rowPosition, columnPosition, slice],
  );

  // The whole matrix is checked in one call, again only when its axes or cells change. Each
  // axis goes with the name this screen shows, so the findings name it the same way.
  const issues = useMemo(
    () =>
      matrixCoverage(rules, {
        dimensions: ordered.map((dimension) => ({ ...dimension, label: axisLabel(dimension) })),
        cells,
      }),
    [rules, ordered, cells],
  );

  const updateValue = useCallback(
    (coordinate: string, text: string) => {
      setDrafts((current) => ({ ...current, [coordinate]: text }));

      const parsed = Number(text);
      if (!cellTextFits(text, wholeOnly)) return;

      setCells((current) =>
        current.map((cell) =>
          cellCoordinate(cell, ordered) === coordinate ? { ...cell, value: parsed } : cell,
        ),
      );
      setDirty(true);
    },
    [ordered, wholeOnly],
  );

  // A cell whose text is not a value of the matrix's type keeps its last good value, so the
  // grid cannot be saved until the person fixes what the screen shows.
  const unfit = Object.values(drafts).filter((text) => !cellTextFits(text, wholeOnly)).length;

  const save = useMutation({
    mutationFn: () => apiService.rateMatrixService.replaceCells(rateMatrixId as string, cells),
    onSuccess: async () => {
      setDirty(false);
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: rateMatrix._def }),
        queryClient.invalidateQueries({ queryKey: ["rate-matrix-list"] }),
      ]);
    },
    onError: () => toast.error(t("Could not save the rates")),
  });

  const postalAxes = readOnly ? [] : ordered.filter((dimension) => isPostalRangeAxis(dimension));

  // A range's cells change together on every sheet, and the text typed into them follows them
  // to their new coordinates. A new range's cells start blank, so the grid cannot be saved
  // until each holds a rate.
  const changeCells = (next: RateMatrixCell[], nextDrafts: Record<string, string>) => {
    setCells(next);
    setDrafts(nextDrafts);
    setDirty(true);
  };

  const addRange = (dimension: RateMatrixDimension, range: PostalRange) => {
    const added = addPostalRange({
      dimensions: ordered,
      cells,
      position: dimension.position as MatrixAxisPosition,
      range,
    });
    const blank = Object.fromEntries(added.map((cell) => [cellCoordinate(cell, ordered), ""]));
    changeCells([...cells, ...added], { ...drafts, ...blank });
  };

  const rerange = (dimension: RateMatrixDimension, from: PostalRange, to: PostalRange) => {
    const next = rerangePostalAxis(cells, dimension, from, to);
    changeCells(next, moveDrafts(drafts, cells, next, ordered));
  };

  const removeRange = (dimension: RateMatrixDimension, range: PostalRange) => {
    const next = withoutPostalRange(cells, dimension, range);
    const kept = new Set(next.map((cell) => cellCoordinate(cell, ordered)));
    changeCells(
      next,
      Object.fromEntries(Object.entries(drafts).filter(([coordinate]) => kept.has(coordinate))),
    );
  };

  const addBlocked =
    ordered.length > 1 && cells.length === 0
      ? t(
          "A range is added across the rates the other axes already hold, and this matrix holds none yet.",
        )
      : null;

  if (!rateMatrixId) {
    return (
      <p className="text-muted-foreground text-sm">
        {t(
          "Save the matrix first. Rates need something to belong to, and the axes above decide what shape the grid takes.",
        )}
      </p>
    );
  }

  if (ordered.length === 0) {
    return (
      <p className="text-muted-foreground text-sm">
        {t(
          "Add at least one axis before entering rates. Without one there is no coordinate to put a number at.",
        )}
      </p>
    );
  }

  if (isLoading) {
    return <MatrixGridEditorSkeleton />;
  }

  return (
    <div className="flex flex-col gap-4">
      {postalAxes.map((dimension) => (
        <PostalRangeEditor
          key={dimension.position}
          dimension={dimension}
          ranges={postalRangesOf(dimension, cells)}
          addBlocked={addBlocked}
          onAdd={(range) => addRange(dimension, range)}
          onChange={(from, to) => rerange(dimension, from, to)}
          onRemove={(range) => removeRange(dimension, range)}
        />
      ))}

      {sliceDimensions.length > 0 && (
        <div className="flex flex-wrap items-end gap-3">
          {sliceDimensions.map((dimension) => {
            const buckets = sliceBuckets(dimension, cells);

            return (
              <div key={dimension.position} className="flex flex-col gap-1">
                <span className="text-muted-foreground text-xs">{axisLabel(dimension)}</span>
                <Select
                  value={slice[dimension.position] ?? ""}
                  onValueChange={(value) =>
                    setSlice((current) => ({ ...current, [dimension.position]: value ?? "" }))
                  }
                >
                  <SelectTrigger className="w-56">
                    <SelectValue placeholder={axisLabel(dimension)} />
                  </SelectTrigger>
                  <SelectContent>
                    {buckets.map((bucket) => (
                      <SelectItem key={bucket.key} value={bucket.key}>
                        {t(bucket.label)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            );
          })}
        </div>
      )}

      {issues.length > 0 && (
        <Alert variant="destructive">
          <AlertCircleIcon className="size-4" />
          <AlertDescription>
            <ul className="list-disc pl-4">
              {issues.map((issue, index) => (
                <li key={`${issue.kind}-${issue.position}-${index}`}>
                  {translateMessage(issue.message, t)}
                  {issue.slices && issue.slices.length > 0 ? (
                    <span className="text-muted-foreground block text-xs">
                      {t("Seen in {0}", formatList(issue.slices))}
                    </span>
                  ) : null}
                </li>
              ))}
            </ul>
          </AlertDescription>
        </Alert>
      )}

      <div className="overflow-x-auto rounded-md border">
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="bg-muted/50">
              <th
                scope="col"
                className="text-muted-foreground border-b px-3 py-2 text-left text-xs font-medium"
              >
                {ordered[0] ? axisLabel(ordered[0]) : ""}
              </th>
              {grid.columns.map((column) => (
                <th
                  scope="col"
                  key={column.key}
                  className="text-muted-foreground border-b px-3 py-2 text-left text-xs font-medium whitespace-nowrap"
                >
                  {t(column.label)}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {grid.rows.map((row) => (
              <tr key={row.key}>
                <th
                  scope="row"
                  className="border-b px-3 py-2 text-left text-xs font-medium whitespace-nowrap"
                >
                  {t(row.label)}
                </th>
                {grid.columns.map((column) => {
                  const cell = grid.cells.get(coordinateKey(row.key, column.key));

                  const coordinate = cell ? cellCoordinate(cell, ordered) : "";

                  return (
                    <td key={column.key} className="border-b px-1 py-1">
                      {cell ? (
                        <div className="relative w-28">
                          <Input
                            className={cn(
                              "h-8 w-28 text-right font-mono text-xs",
                              valueUnit && "pr-12",
                            )}
                            aria-label={t("Value at {0}, {1}", t(row.label), t(column.label))}
                            aria-invalid={
                              drafts[coordinate] !== undefined &&
                              !cellTextFits(drafts[coordinate], wholeOnly)
                            }
                            value={drafts[coordinate] ?? String(cell.value ?? "")}
                            readOnly={readOnly}
                            onChange={(event) => updateValue(coordinate, event.target.value)}
                          />
                          {valueUnit ? (
                            <span className="text-muted-foreground text-2xs pointer-events-none absolute inset-y-0 right-2 flex items-center font-mono select-none">
                              {valueUnit}
                            </span>
                          ) : null}
                        </div>
                      ) : (
                        <span
                          className="text-muted-foreground block w-28 px-2 text-right text-xs"
                          title={t(
                            "Nothing prices this coordinate — a lane landing here rates at nothing",
                          )}
                        >
                          —
                        </span>
                      )}
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {grid.rows.length === 0 && (
        <p className="text-muted-foreground text-sm">
          {readOnly
            ? t("This version holds no rates.")
            : t(
                "This matrix has no rates yet. Every lane pointing at it prices nothing until it does.",
              )}
        </p>
      )}

      {version ? (
        <p className="text-muted-foreground text-xs">
          {t(
            "Version {0} is published and cannot change. Template versions that pin it keep reading these rates.",
            version.versionNumber,
          )}
        </p>
      ) : (
        <div className="flex items-center gap-3">
          <Button
            type="button"
            size="sm"
            disabled={!dirty || unfit > 0 || save.isPending}
            onClick={() => save.mutate()}
          >
            {save.isPending ? (
              <SpinnerIcon className="mr-1 size-3.5 animate-spin" />
            ) : (
              <Save01Icon className="mr-1 size-3.5" />
            )}
            {t("Save rates")}
          </Button>
          {unfit > 0 ? (
            <span role="alert" className="text-destructive text-xs">
              {wholeOnly
                ? t(
                    "{0, plural, one {# cell is not a whole number} other {# cells are not whole numbers}}. This matrix holds whole numbers.",
                    unfit,
                  )
                : t(
                    "{0, plural, one {# cell is not a number} other {# cells are not numbers}}.",
                    unfit,
                  )}
            </span>
          ) : (
            <span className="text-muted-foreground text-xs">
              {t(
                "Rates save to the draft, separately from the rest of this form. Tariffs read them once a reviewer publishes the draft.",
              )}
            </span>
          )}
        </div>
      )}
    </div>
  );
}
