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
import { Switch } from "@trenova/shared/components/ui/switch";
import { graphQLErrorMessage } from "@trenova/shared/lib/graphql";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import {
  useCreateReportView,
  useDeleteReportView,
  useReportViews,
} from "@/hooks/use-reports";
import { BookmarkIcon, TrashIcon, XIcon } from "lucide-react";
import { AnimatePresence, m } from "motion/react";
import { useCallback, useMemo, useState } from "react";
import { toast } from "sonner";

type SavedViewFieldProps = {
  definitionId: string;
  params: Record<string, unknown>;
  format: string;
  selectedViewId: string | null;
  onSelect: (view: { id: string; params: Record<string, unknown>; format: string } | null) => void;
};

/** Sentinel for "run with whatever is typed", distinct from any real view id. */
const NO_VIEW = "__none__";

/**
 * Picks a saved set of parameter values for a report, and saves the current
 * ones as a new view. A parameterized report is one question asked many ways;
 * without this every asking means retyping every parameter.
 */
export function SavedViewField({
  definitionId,
  params,
  format,
  selectedViewId,
  onSelect,
}: SavedViewFieldProps) {
  const currentUserId = useAuthStore((state) => state.user?.id);
  const { data: views, isLoading } = useReportViews(definitionId);
  const createView = useCreateReportView();
  const deleteView = useDeleteReportView();

  const [naming, setNaming] = useState(false);
  const [name, setName] = useState("");
  const [shared, setShared] = useState(false);

  const choices = useMemo(
    () => [
      { value: NO_VIEW, label: "No saved view" },
      ...(views ?? []).map((view) => ({
        value: view.id,
        label: view.pinned ? `★ ${view.name}` : view.name,
      })),
    ],
    [views],
  );

  const selected = useMemo(
    () => (views ?? []).find((view) => view.id === selectedViewId) ?? null,
    [views, selectedViewId],
  );

  const handleSelect = useCallback(
    (value: string | null) => {
      if (!value) return;
      if (value === NO_VIEW) {
        onSelect(null);
        return;
      }
      const view = (views ?? []).find((candidate) => candidate.id === value);
      if (!view) return;
      onSelect({
        id: view.id,
        params: (view.params as Record<string, unknown> | null) ?? {},
        format: view.format ?? format,
      });
    },
    [views, onSelect, format],
  );

  const handleSave = useCallback(() => {
    const trimmed = name.trim();
    if (!trimmed) return;

    createView.mutate(
      {
        definitionId,
        name: trimmed,
        params: Object.keys(params).length > 0 ? params : undefined,
        shared,
        format,
      },
      {
        onSuccess: (view) => {
          setNaming(false);
          setName("");
          setShared(false);
          onSelect({
            id: view.id,
            params: (view.params as Record<string, unknown> | null) ?? {},
            format: view.format ?? format,
          });
          toast.success(`Saved "${view.name}"`);
        },
        onError: (error) => {
          toast.error(graphQLErrorMessage(error, "Failed to save the view"));
        },
      },
    );
  }, [name, shared, createView, definitionId, params, format, onSelect]);

  const handleDelete = useCallback(() => {
    if (!selected) return;
    deleteView.mutate(selected.id, {
      onSuccess: () => {
        onSelect(null);
        toast.success(`Deleted "${selected.name}"`);
      },
      onError: (error) => {
        toast.error(graphQLErrorMessage(error, "Failed to delete the view"));
      },
    });
  }, [selected, deleteView, onSelect]);

  // Only the owner may delete, which is what the server enforces too — showing
  // the control to anyone else would only produce an error they cannot act on.
  const canDelete = !!selected && !!currentUserId && selected.ownerId === currentUserId;

  return (
    <div className="flex flex-col gap-1.5">
      <Label htmlFor="report-run-view">Saved view</Label>
      <div className="flex items-center gap-1.5">
        <Select
          value={selectedViewId ?? NO_VIEW}
          onValueChange={handleSelect}
          items={choices}
          disabled={isLoading}
        >
          <SelectTrigger className="w-full" id="report-run-view">
            <SelectValue placeholder="No saved view" />
          </SelectTrigger>
          <SelectContent>
            {choices.map((choice) => (
              <SelectItem key={choice.value} value={choice.value}>
                {choice.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {canDelete && (
          <Button
            variant="outline"
            size="icon"
            aria-label="Delete view"
            disabled={deleteView.isPending}
            onClick={handleDelete}
          >
            <TrashIcon className="size-3.5" />
          </Button>
        )}
        <Button
          variant="outline"
          size="icon"
          aria-label={naming ? "Cancel saving view" : "Save current values as a view"}
          onClick={() => setNaming((prev) => !prev)}
        >
          {naming ? <XIcon className="size-3.5" /> : <BookmarkIcon className="size-3.5" />}
        </Button>
      </div>

      <AnimatePresence initial={false}>
        {naming && (
          <m.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.15 }}
            className="overflow-hidden"
          >
            <div className="flex flex-col gap-2 pt-1.5">
              <Input
                autoFocus
                placeholder="West region, last quarter"
                value={name}
                onChange={(event) => setName(event.target.value)}
                onKeyDown={(event) => {
                  if (event.key === "Enter") handleSave();
                }}
              />
              <div className="flex items-center justify-between gap-2">
                <Label
                  className="text-xs font-normal text-muted-foreground"
                  htmlFor="report-view-shared"
                >
                  Share with everyone who can read this report
                </Label>
                <Switch id="report-view-shared" checked={shared} onCheckedChange={setShared} />
              </div>
              <Button
                size="sm"
                onClick={handleSave}
                disabled={createView.isPending || name.trim() === ""}
              >
                {createView.isPending ? "Saving..." : "Save view"}
              </Button>
            </div>
          </m.div>
        )}
      </AnimatePresence>
    </div>
  );
}
