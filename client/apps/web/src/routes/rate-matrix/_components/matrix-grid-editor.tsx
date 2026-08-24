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
  axisLabel,
  buildMatrixGrid,
  cellCoordinate,
  coordinateKey,
  findMatrixCoverageIssues,
  sliceBuckets,
  type MatrixAxisPosition,
} from "@trenova/shared/lib/rate-matrix";
import type { RateMatrix, RateMatrixCell, RateMatrixDimension } from "@trenova/shared/types/rate";
import { CircleAlertIcon, LoaderCircleIcon, SaveIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";
import { toast } from "sonner";

type MatrixGridEditorProps = {
  /** Absent while the matrix is being created — there is nothing to hang cells off yet. */
  readonly rateMatrixId?: string;
};

/**
 * The tariff sheet.
 *
 * Rates live in their own resource rather than in the form, because a
 * four-axis class tariff runs to tens of thousands of cells and a form field
 * array would re-render all of them on every keystroke. The trade is that this
 * saves separately from the matrix header, which the button says out loud.
 */
export function MatrixGridEditor({ rateMatrixId }: MatrixGridEditorProps) {
  const { control } = useFormContext<RateMatrix>();
  const dimensions = (useWatch({ control, name: "dimensions" }) ?? []) as RateMatrixDimension[];
  const queryClient = useQueryClient();

  const { data: loadedCells, isLoading } = useQuery({
    queryKey: ["rate-matrix-cells", rateMatrixId],
    queryFn: () => apiService.rateMatrixService.listCells(rateMatrixId as string),
    enabled: Boolean(rateMatrixId),
  });

  const [cells, setCells] = useState<RateMatrixCell[]>([]);
  const [dirty, setDirty] = useState(false);

  // Text being typed is kept apart from the parsed number, because "2." and
  // "-" are states somebody passes through on the way to a valid rate and
  // coercing them on every keystroke would fight the person entering them.
  const [drafts, setDrafts] = useState<Record<string, string>>({});

  useEffect(() => {
    if (loadedCells) {
      setCells(loadedCells);
      setDrafts({});
      setDirty(false);
    }
  }, [loadedCells]);

  // useWatch hands back a fresh array every render, so memoizing on it would
  // memoize nothing and every grid below would rebuild on each keystroke. What
  // actually changes is the shape of the axes, so that is what is compared.
  const signature = dimensions
    .map((dimension) => `${dimension.position}:${dimension.kind}:${dimension.matchMode}`)
    .join("|");
  const orderedRef = useRef<RateMatrixDimension[]>([]);
  const signatureRef = useRef<string | null>(null);

  if (signatureRef.current !== signature) {
    signatureRef.current = signature;
    orderedRef.current = [...dimensions].sort((a, b) => a.position - b.position);
  }

  const ordered = orderedRef.current;

  const rowPosition = (ordered[0]?.position ?? 0) as MatrixAxisPosition;
  const columnDimension = ordered[1];
  const columnPosition = (columnDimension?.position ?? null) as MatrixAxisPosition | null;
  const sliceDimensions = useMemo(() => ordered.slice(2), [ordered]);

  const [slice, setSlice] = useState<Record<number, string>>({});

  // A sheet selector pointing at a band that no longer exists renders an empty
  // grid that looks like missing data. Snapping to the first available bucket
  // keeps the editor showing something real.
  useEffect(() => {
    setSlice((current) => {
      let changed = false;
      const next = { ...current };

      for (const dimension of sliceDimensions) {
        const buckets = sliceBuckets(dimension, cells);
        const chosen = next[dimension.position];

        if (buckets.length === 0) continue;
        if (chosen !== undefined && buckets.some((bucket) => bucket.key === chosen)) continue;

        next[dimension.position] = buckets[0]?.key ?? "";
        changed = true;
      }

      return changed ? next : current;
    });
  }, [cells, sliceDimensions]);

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

  const issues = useMemo(() => findMatrixCoverageIssues(ordered, cells), [ordered, cells]);

  const updateValue = useCallback(
    (coordinate: string, text: string) => {
      setDrafts((current) => ({ ...current, [coordinate]: text }));

      const parsed = Number(text);
      if (text.trim() === "" || !Number.isFinite(parsed)) return;

      setCells((current) =>
        current.map((cell) =>
          cellCoordinate(cell, ordered) === coordinate ? { ...cell, value: parsed } : cell,
        ),
      );
      setDirty(true);
    },
    [ordered],
  );

  const save = useMutation({
    mutationFn: () => apiService.rateMatrixService.replaceCells(rateMatrixId as string, cells),
    onSuccess: async () => {
      setDirty(false);
      toast.success("Rates saved");
      await queryClient.invalidateQueries({ queryKey: ["rate-matrix-cells", rateMatrixId] });
    },
    onError: () => toast.error("Could not save the rates"),
  });

  if (!rateMatrixId) {
    return (
      <p className="text-muted-foreground text-sm">
        Save the matrix first. Rates need something to belong to, and the axes above decide what
        shape the grid takes.
      </p>
    );
  }

  if (ordered.length === 0) {
    return (
      <p className="text-muted-foreground text-sm">
        Add at least one axis before entering rates. Without one there is no coordinate to put a
        number at.
      </p>
    );
  }

  if (isLoading) {
    return (
      <div className="text-muted-foreground flex items-center gap-2 text-sm">
        <LoaderCircleIcon className="size-4 animate-spin" />
        Loading rates
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
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
                        {bucket.label}
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
          <CircleAlertIcon className="size-4" />
          <AlertDescription>
            <ul className="list-disc pl-4">
              {issues.map((issue) => (
                <li key={`${issue.kind}-${issue.position}-${issue.message}`}>{issue.message}</li>
              ))}
            </ul>
          </AlertDescription>
        </Alert>
      )}

      <div className="overflow-x-auto rounded-md border">
        <table className="w-full border-collapse text-sm">
          <thead>
            <tr className="bg-muted/50">
              <th className="text-muted-foreground border-b px-3 py-2 text-left text-xs font-medium">
                {ordered[0] ? axisLabel(ordered[0]) : ""}
              </th>
              {grid.columns.map((column) => (
                <th
                  key={column.key}
                  className="text-muted-foreground border-b px-3 py-2 text-left text-xs font-medium whitespace-nowrap"
                >
                  {column.label}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {grid.rows.map((row) => (
              <tr key={row.key}>
                <th className="border-b px-3 py-2 text-left text-xs font-medium whitespace-nowrap">
                  {row.label}
                </th>
                {grid.columns.map((column) => {
                  const cell = grid.cells.get(coordinateKey(row.key, column.key));

                  const coordinate = cell ? cellCoordinate(cell, ordered) : "";

                  return (
                    <td key={column.key} className="border-b px-1 py-1">
                      {cell ? (
                        <Input
                          className="h-8 w-28 text-right font-mono text-xs"
                          value={drafts[coordinate] ?? String(cell.value ?? "")}
                          onChange={(event) => updateValue(coordinate, event.target.value)}
                        />
                      ) : (
                        <span
                          className="text-muted-foreground block w-28 px-2 text-right text-xs"
                          title="Nothing prices this coordinate — a lane landing here rates at nothing"
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
          This matrix has no rates yet. Every lane pointing at it prices nothing until it does.
        </p>
      )}

      <div className="flex items-center gap-3">
        <Button
          type="button"
          size="sm"
          disabled={!dirty || save.isPending}
          onClick={() => save.mutate()}
        >
          {save.isPending ? (
            <LoaderCircleIcon className="mr-1 size-3.5 animate-spin" />
          ) : (
            <SaveIcon className="mr-1 size-3.5" />
          )}
          Save rates
        </Button>
        <span className="text-muted-foreground text-xs">
          Rates save on their own, separately from the rest of this form.
        </span>
      </div>
    </div>
  );
}
