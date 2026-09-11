import { useT } from "@trenova/shared/i18n/use-t";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  markWorkerChecklistItemNotApplicable,
  skipWorkerChecklistItem,
  type WorkerChecklistItemRow,
  type WorkerChecklistRow,
} from "@/lib/graphql/worker-checklist";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { z } from "zod";
import { useEffect } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { useChecklistInvalidation } from "./use-checklist-invalidation";

export type ItemNoteMode = "skip" | "notApplicable";

const COPY: Record<
  ItemNoteMode,
  { title: string; description: string; submit: string; required: string }
> = {
  skip: {
    title: "Skip item",
    description:
      "The item settles without being done. It still counts toward closing the checklist, so say why.",
    submit: "Skip item",
    required: "Say why this item is being skipped",
  },
  notApplicable: {
    title: "Mark not applicable",
    description: "Use this when the item does not apply to this worker at all.",
    submit: "Mark not applicable",
    required: "Say why this item does not apply",
  },
};

function noteSchema(required: string) {
  return z.object({
    note: z
      .string()
      .trim()
      .min(1, { message: required })
      .max(500, { message: "Note cannot exceed 500 characters" }),
  });
}

type NoteValues = { note: string };

type ItemNoteDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  mode: ItemNoteMode;
  item: Pick<WorkerChecklistItemRow, "id" | "label" | "version"> | null;
};

export function ItemNoteDialog({ open, onOpenChange, workerId, mode, item }: ItemNoteDialogProps) {
  const t = useT();

  const invalidate = useChecklistInvalidation(workerId);
  const copy = COPY[mode];
  const form = useForm<NoteValues>({
    resolver: zodResolver(noteSchema(copy.required)) as Resolver<NoteValues>,
    defaultValues: { note: "" },
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (open) reset({ note: "" });
  }, [open, item, reset]);

  const { mutateAsync, isPending } = useApiMutation<
    WorkerChecklistRow,
    NoteValues,
    unknown,
    NoteValues
  >({
    form,
    resourceName: "Checklist item",
    mutationFn: (values) => {
      if (!item) throw new Error("No item selected");
      const input = { id: item.id, note: values.note, version: item.version };
      return mode === "skip"
        ? skipWorkerChecklistItem(input)
        : markWorkerChecklistItemNotApplicable(input);
    },
    onSuccess: () => {
      toast.success(mode === "skip" ? "Item skipped" : "Item marked not applicable");
      void invalidate();
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t(copy.title)}</DialogTitle>
          <DialogDescription>
            {item ? <span className="font-medium">{t(item.label)}. </span> : null}
            {t(copy.description)}
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(event) => {
              event.preventDefault();
              event.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(event);
            }}
          >
            <FormGroup cols={1}>
              <FormControl cols="full">
                <TextareaField<NoteValues>
                  control={control}
                  name="note"
                  label={t("Note")}
                  placeholder={t("e.g. Completed at the previous terminal")}
                  description={t("Kept on the item so an auditor can see why it was not done.")}
                  rules={{ required: true }}
                  maxLength={500}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter className="mt-4">
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                {t("Cancel")}
              </Button>
              <Button type="submit" disabled={isPending || !item}>
                {isPending ? "Saving..." : copy.submit}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
