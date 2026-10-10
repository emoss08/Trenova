import {
  AGENT_MEMORY_LIST_KEY,
  createAgentMemory,
  reviewAgentMemory,
  setAgentMemoryStatus,
  updateAgentMemory,
  type AgentMemoryRow,
} from "@/lib/graphql/agent-memories";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateTimeMedium } from "@trenova/shared/lib/date";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { useState } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { memorySourceLabel } from "../activity/agent-badges";
import { EditSheet, type EditSection } from "../edit/edit-sheet";
import type { EditFields } from "../edit/change-review";
import { useEditFlow } from "../edit/use-edit-flow";
import { Ic } from "../kit/ic";
import { invalidateAIControlCounts } from "../overview/use-ai-control-stats";
import { MemoryAbout, MemoryText, MemoryUntil } from "./memory-form";
import {
  memoryFormDefaults,
  memoryFormSchema,
  memoryValuesFromRow,
  toMemoryInput,
  type MemoryFormValues,
} from "./memory-form-schema";
import { MEMORY_KIND_LABELS } from "./memory-kind";
import { MemoryProvenance } from "./memory-provenance";

type MemoryPanelProps = DataTablePanelProps<AgentMemoryRow> & {
  /** What a new memory starts from, such as an example picked from the empty list. */
  preset?: Partial<MemoryFormValues> | null;
};

/** The table's editor for one memory: a new one, or one already kept. */
export function MemoryPanel({ open, onOpenChange, mode, row, preset }: MemoryPanelProps) {
  if (!open) {
    return null;
  }

  return (
    <MemoryEditor
      key={mode === "edit" && row ? row.id : "new"}
      row={mode === "edit" ? row : null}
      preset={preset ?? null}
      onClose={() => onOpenChange(false)}
    />
  );
}

type MemoryEditorProps = {
  row: AgentMemoryRow | null;
  preset: Partial<MemoryFormValues> | null;
  onClose: () => void;
};

function MemoryEditor({ row, preset, onClose }: MemoryEditorProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const [retiring, setRetiring] = useState(false);
  const [reviewing, setReviewing] = useState(false);
  const [reviewError, setReviewError] = useState<string | null>(null);
  const form = useForm<MemoryFormValues>({
    resolver: zodResolver(memoryFormSchema) as Resolver<MemoryFormValues>,
    defaultValues: row ? memoryValuesFromRow(row) : { ...memoryFormDefaults, ...preset },
    mode: "onChange",
  });

  const refresh = () =>
    Promise.all([
      queryClient.invalidateQueries({ queryKey: [AGENT_MEMORY_LIST_KEY] }),
      invalidateAIControlCounts(queryClient),
    ]);

  const flow = useEditFlow({
    form,
    create: row === null,
    resourceName: t("Memory"),
    onClose,
    versionField: "version",
    onSave: async (values) => {
      const saved = row
        ? await updateAgentMemory(row.id, toMemoryInput(values))
        : await createAgentMemory(toMemoryInput(values));
      await refresh();
      toast.success(
        row ? t("Memory saved") : t("Memory saved; agents read it from their next run"),
      );
      return memoryValuesFromRow(saved);
    },
  });

  const retire = async () => {
    if (!row) return;
    setRetiring(true);
    try {
      await setAgentMemoryStatus(row.id, row.status === "Retired" ? "Active" : "Retired");
      await refresh();
      toast.success(row.status === "Retired" ? t("Memory restored") : t("Memory retired"));
      onClose();
    } finally {
      setRetiring(false);
    }
  };

  const review = async () => {
    if (!row) return;
    setReviewing(true);
    setReviewError(null);
    try {
      await reviewAgentMemory(row.id, row.version);
      await refresh();
      toast.success(t("Memory reviewed; turns that read it can write on their own again"));
      onClose();
    } catch (error) {
      setReviewError(error instanceof Error ? error.message : t("That didn't go through"));
    } finally {
      setReviewing(false);
    }
  };

  const fields: EditFields = {
    content: { label: t("Memory") },
    kind: { label: t("Kind"), format: (value) => t(MEMORY_KIND_LABELS[value as never] ?? "") },
    subjectType: { label: t("About") },
    subjectId: { label: t("Record") },
    toolName: { label: t("Tool") },
    expiresAt: {
      label: t("Until"),
      format: (value) =>
        typeof value === "number" ? formatUnixDateTimeMedium(value) : t("No end"),
    },
  };

  const sections: EditSection[] = [
    { id: "memory", label: t("Memory"), keys: ["content", "kind"], content: <MemoryText /> },
    {
      id: "about",
      label: t("About"),
      keys: ["subjectType", "subjectId", "toolName"],
      note: t("Leave it on every agent for something every agent should know."),
      content: <MemoryAbout />,
    },
    {
      id: "until",
      label: t("Until"),
      keys: ["expiresAt"],
      note: t("After this day agents stop reading it."),
      content: <MemoryUntil />,
    },
  ];
  if (row?.taints) {
    sections.push({
      id: "review",
      label: t("Review"),
      note: t(
        "An agent wrote this after reading outside text. Keep it once you've read it, and the turns that read it can write on their own again.",
      ),
      content: (
        <div className="flex flex-col items-start gap-2">
          {reviewError ? (
            <Alert size="sm" variant="destructive">
              <AlertDescription>{reviewError}</AlertDescription>
            </Alert>
          ) : null}
          <Button
            type="button"
            size="sm"
            variant="outline"
            isLoading={reviewing}
            disabled={flow.dirty}
            onClick={() => void review()}
          >
            {t("Reviewed, keep it")}
          </Button>
          {flow.dirty ? (
            <span className="text-foreground-muted text-xs">
              {t("Save or discard your changes first, so what you keep is what you read.")}
            </span>
          ) : null}
        </div>
      ),
    });
  }
  if (row) {
    sections.push(
      {
        id: "provenance",
        label: t("Where it came from"),
        note: t("Recorded with the memory; editing it does not change this."),
        content: <MemoryProvenance memory={row} />,
      },
      {
        id: "retire",
        label: row.status === "Retired" ? t("Restore") : t("Retire"),
        note:
          row.status === "Retired"
            ? t("Agents read it again from their next run.")
            : t(
                "Agents stop reading it. The memory is kept, so what they were told stays readable.",
              ),
        content: (
          <button
            type="button"
            className={row.status === "Retired" ? "xa" : "xa d"}
            disabled={retiring}
            onClick={() => void retire()}
          >
            <Ic n={row.status === "Retired" ? "undo" : "trash"} s={13} />
            {row.status === "Retired" ? t("Restore memory") : t("Retire memory")}
          </button>
        ),
      },
    );
  }

  return (
    <FormProvider {...form}>
      <EditSheet
        open
        form={form}
        flow={flow}
        fields={fields}
        sections={sections}
        icon={
          <span className="src-i">
            <Ic n="brain" s={18} />
          </span>
        }
        title={row ? t("Edit memory") : t("New memory")}
        subtitle={
          row
            ? t(
                "{0} · {1} · {2, plural, one {read # time} other {read # times}}",
                memorySourceLabel(row.source, t),
                formatUnixDateTimeMedium(row.createdAt),
                row.useCount,
              )
            : t("Every agent that asks for memory reads it")
        }
        saveLabel={row ? t("Save changes") : t("Save memory")}
      />
    </FormProvider>
  );
}
