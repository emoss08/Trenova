import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
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
import { apiService } from "@/services/api";
import {
  forkRequestSchema,
  type ForkRequest,
  type FormulaTemplate,
} from "@trenova/shared/types/formula-template";
import { invalidateFormulaTemplate } from "@/lib/queries/formula-template";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

export type ForkableTemplate = Pick<FormulaTemplate, "id" | "name" | "currentVersionNumber">;

/** Form defaults for forking a template; recomputed whenever the target changes. */
export function forkDefaultsFor(template: ForkableTemplate | null): ForkRequest {
  return {
    newName: template ? `${template.name} (Fork)` : "",
    sourceVersion: template?.currentVersionNumber,
    changeMessage: "",
  };
}

type ForkTemplateDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  template: ForkableTemplate | null;
  onForkSuccess?: (forkedTemplate: FormulaTemplate) => void;
};

export function ForkTemplateDialog({
  open,
  onOpenChange,
  template,
  onForkSuccess,
}: ForkTemplateDialogProps) {
  const t = useT();

  const queryClient = useQueryClient();

  const form = useForm<ForkRequest>({
    resolver: zodResolver(forkRequestSchema),
    defaultValues: forkDefaultsFor(template),
  });

  const {
    control,
    handleSubmit,
    reset,
    formState: { isSubmitting },
  } = form;

  // The dialog is mounted once and pointed at different templates, so the
  // defaults captured at mount are the wrong template's by the time it opens.
  useEffect(() => {
    if (open) {
      reset(forkDefaultsFor(template));
    }
  }, [open, template, reset]);

  const handleClose = () => {
    onOpenChange(false);
    reset();
  };

  const onSubmit = async (values: ForkRequest) => {
    if (!template?.id) return;

    await apiService.formulaTemplateService
      .fork(template.id, values)
      .then((forkedTemplate) => {
        toast.success(t("Template forked successfully"), {
          description: `Created "${forkedTemplate.name}"`,
        });

        void invalidateFormulaTemplate(queryClient);
        handleClose();
        onForkSuccess?.(forkedTemplate);
      })
      .catch(() => {
        toast.error(t("Fork failed"), {
          description: t("Could not fork the template. Please try again."),
        });
      });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">{t("Fork Template")}</DialogTitle>
          <DialogDescription>
            {t(
              "Create a new template based on “{0}”. The forked template will start with its own version history.",
              template?.name,
            )}
          </DialogDescription>
        </DialogHeader>

        <Form id="fork-form" onSubmit={handleSubmit(onSubmit)}>
          <FormGroup>
            <FormControl cols="full">
              <InputField
                label={t("New Template Name")}
                name="newName"
                control={control}
                rules={{ required: true }}
                placeholder={t("Enter name for the forked template")}
              />
            </FormControl>

            <FormControl cols="full">
              <TextareaField
                label={t("Description")}
                name="changeMessage"
                control={control}
                placeholder={t("Why are you forking this template?")}
                rows={3}
              />
            </FormControl>
          </FormGroup>
        </Form>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>
            {t("Cancel")}
          </Button>
          <Button type="submit" form="fork-form" disabled={isSubmitting}>
            {isSubmitting ? t("Forking...") : t("Fork Template")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
