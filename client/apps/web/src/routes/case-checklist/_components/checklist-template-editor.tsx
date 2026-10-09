import { FormSaveDock } from "@/components/form-save-dock";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { newCustomStepKey } from "@/lib/case-checklist-steps";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import {
  checklistTemplateSchema,
  templateItemSchema,
  type ChecklistKind,
  type ChecklistTemplate,
  type TemplateItem,
} from "@/types/case-checklist";
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
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
} from "@dnd-kit/sortable";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { PlusIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import { useT } from "@trenova/shared/i18n/use-t";
import { useState } from "react";
import { FormProvider, useFieldArray, useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";
import { ChecklistStepRow } from "./checklist-step-row";

const MAX_ADDED_STEPS = 12;

/**
 * What a person may save: the server checks it whole again, but an added
 * step without a name, or a document step without its document type, is
 * said next to the field before anything is sent.
 */
const editorSchema = z.object({
  items: z.array(
    templateItemSchema.superRefine((item, ctx) => {
      const custom = item.custom;
      if (!custom) {
        return;
      }
      if (custom.label.trim() === "") {
        ctx.addIssue({ code: "custom", path: ["custom", "label"], message: "Name the step" });
      }
      if (custom.label.length > 80) {
        ctx.addIssue({
          code: "custom",
          path: ["custom", "label"],
          message: "A step's name can be at most 80 characters",
        });
      }
      if (custom.check === "Document" && !custom.documentTypeId) {
        ctx.addIssue({
          code: "custom",
          path: ["custom", "documentTypeId"],
          message: "Choose the document type that ticks it",
        });
      }
      if ((custom.stepLabel ?? "").length > 40) {
        ctx.addIssue({
          code: "custom",
          path: ["custom", "stepLabel"],
          message: "A step's button can say at most 40 characters",
        });
      }
      if ((custom.prompt ?? "").length > 500) {
        ctx.addIssue({
          code: "custom",
          path: ["custom", "prompt"],
          message: "What the step asks the agent can be at most 500 characters",
        });
      }
    }),
  ),
});

type EditorValues = z.infer<typeof editorSchema>;

export function ChecklistTemplateEditor({
  kind,
  template,
  locked,
  onSaved,
  onRemoved,
}: {
  kind: ChecklistKind;
  template: ChecklistTemplate;
  locked: readonly string[];
  onSaved: (template: ChecklistTemplate) => void;
  onRemoved: () => void;
}) {
  const t = useT();
  const queryClient = useQueryClient();
  const isCustomer = Boolean(template.customerId);
  const [confirming, setConfirming] = useState(false);
  const [expanded, setExpanded] = useState<string | null>(null);

  const form = useForm<EditorValues>({
    resolver: zodResolver(editorSchema),
    defaultValues: { items: template.items },
  });
  const { control, handleSubmit, reset } = form;
  const { fields, append, remove, move } = useFieldArray({ control, name: "items" });
  const added = fields.filter((field) => field.custom).length;

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );

  const refresh = () =>
    queryClient.invalidateQueries({ queryKey: queries.caseChecklist.list(kind).queryKey });

  const save = useApiMutation({
    mutationFn: (values: EditorValues) =>
      apiService.caseChecklistService.save({
        id: template.id || undefined,
        version: template.version,
        kind,
        customerId: template.customerId || undefined,
        items: values.items,
      }),
    form,
    resourceName: "Case Checklist",
    onSuccess: async (saved) => {
      const next = checklistTemplateSchema.parse({
        ...saved,
        customerName: saved.customerName || template.customerName,
      });
      reset({ items: next.items });
      toast.success(t("Checklist saved"));
      await refresh();
      onSaved(next);
    },
  });

  const removeTemplate = useApiMutation({
    mutationFn: () => apiService.caseChecklistService.remove(template.id),
    resourceName: "Case Checklist",
    onSuccess: async () => {
      toast.success(
        isCustomer
          ? t("{0} now uses your organization's checklist", template.customerName)
          : t("Your organization uses Trenova's default checklist again"),
      );
      await refresh();
      onRemoved();
    },
  });

  const onDragEnd = ({ active, over }: DragEndEvent) => {
    if (!over || active.id === over.id) {
      return;
    }
    const from = fields.findIndex((field) => field.id === active.id);
    const to = fields.findIndex((field) => field.id === over.id);
    if (from >= 0 && to >= 0) {
      move(from, to);
    }
  };

  const addStep = () => {
    const step: TemplateItem = {
      key: newCustomStepKey(),
      mode: "Required",
      custom: { label: "", check: "Manual", documentTypeId: "", stepLabel: "", prompt: "" },
    };
    append(step);
    setExpanded(step.key);
  };

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit((values) => save.mutateAsync(values))}>
        <section className="ring-foreground/10 flex flex-col rounded-lg ring-1">
          <header className="border-border flex flex-wrap items-start gap-3 border-b px-4 py-3">
            <div className="flex min-w-0 flex-1 flex-col gap-0.5">
              <h2 className="text-sm font-semibold">
                {isCustomer
                  ? t("{0}'s checklist", template.customerName || t("Customer"))
                  : t("Your organization's checklist")}
              </h2>
              <p className="text-muted-foreground text-xs">
                {isCustomer
                  ? t("Used for cases about this customer's records, in place of your organization's.")
                  : t("Used for every case unless the customer has its own. Drag steps to reorder them.")}
              </p>
            </div>
            {template.id && (
              <Button
                type="button"
                size="sm"
                variant={confirming ? "destructive" : "outline"}
                isLoading={removeTemplate.isPending}
                onBlur={() => setConfirming(false)}
                onClick={() => {
                  if (!confirming) {
                    setConfirming(true);
                    return;
                  }
                  setConfirming(false);
                  removeTemplate.mutate();
                }}
              >
                {confirming
                  ? t("Click again to confirm")
                  : isCustomer
                    ? t("Remove this customer's checklist")
                    : t("Go back to the default")}
              </Button>
            )}
          </header>

          <DndContext
            sensors={sensors}
            collisionDetection={closestCenter}
            modifiers={[restrictToVerticalAxis]}
            onDragEnd={onDragEnd}
          >
            <SortableContext
              items={fields.map((field) => field.id)}
              strategy={verticalListSortingStrategy}
            >
              <ol className="divide-border flex flex-col divide-y" aria-label={t("Steps")}>
                {fields.map((field, index) => (
                  <ChecklistStepRow
                    key={field.id}
                    sortId={field.id}
                    index={index}
                    kind={kind}
                    stepKey={field.key}
                    locked={locked.includes(field.key)}
                    expanded={expanded === field.key}
                    onToggle={() =>
                      setExpanded((current) => (current === field.key ? null : field.key))
                    }
                    onRemove={() => remove(index)}
                  />
                ))}
              </ol>
            </SortableContext>
          </DndContext>

          <footer className="border-border flex items-center gap-3 border-t px-4 py-3">
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={added >= MAX_ADDED_STEPS}
              onClick={addStep}
            >
              <PlusIcon className="size-4" />
              {t("Add a step")}
            </Button>
            <span className="text-muted-foreground text-xs">
              {added >= MAX_ADDED_STEPS
                ? t("A checklist can have at most 12 added steps")
                : t("A step you add is ticked by a person on the case, or by a document on file.")}
            </span>
          </footer>
        </section>
        <FormSaveDock
          saveButtonContent={template.id ? t("Save changes") : t("Save checklist")}
          alwaysVisible={!template.id && isCustomer}
        />
      </Form>
    </FormProvider>
  );
}
