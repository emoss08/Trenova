import { InputField } from "@/components/fields/input-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { apiService } from "@/services/api";
import {
  forkRequestSchema,
  type ForkRequest,
  type FormulaTemplate,
} from "@/types/formula-template";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

type ForkTemplateDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  template: FormulaTemplate | null;
  onForkSuccess?: (forkedTemplate: FormulaTemplate) => void;
};

export function ForkTemplateDialog({
  open,
  onOpenChange,
  template,
  onForkSuccess,
}: ForkTemplateDialogProps) {
  const queryClient = useQueryClient();

  const form = useForm<ForkRequest>({
    resolver: zodResolver(forkRequestSchema),
    defaultValues: {
      newName: template ? `${template.name} (Fork)` : "",
      sourceVersion: template?.currentVersionNumber,
      changeMessage: "",
    },
  });

  const {
    control,
    handleSubmit,
    reset,
    formState: { isSubmitting },
  } = form;

  const handleClose = () => {
    onOpenChange(false);
    reset();
  };

  const onSubmit = async (values: ForkRequest) => {
    if (!template?.id) return;

    await apiService.formulaTemplateService
      .fork(template.id, values)
      .then((forkedTemplate) => {
        toast.success("Template forked successfully", {
          description: `Created "${forkedTemplate.name}"`,
        });

        void queryClient.invalidateQueries({ queryKey: ["formula-template-list"] });
        handleClose();
        onForkSuccess?.(forkedTemplate);
      })
      .catch(() => {
        toast.error("Fork failed", {
          description: "Could not fork the template. Please try again.",
        });
      });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            Fork Template
          </DialogTitle>
          <DialogDescription>
            Create a new template based on &ldquo;{template?.name}&rdquo;. The
            forked template will start with its own version history.
          </DialogDescription>
        </DialogHeader>

        <Form id="fork-form" onSubmit={handleSubmit(onSubmit)}>
          <FormGroup>
            <FormControl cols="full">
              <InputField
                label="New Template Name"
                name="newName"
                control={control}
                rules={{ required: true }}
                placeholder="Enter name for the forked template"
              />
            </FormControl>

            <FormControl cols="full">
              <TextareaField
                label="Description"
                name="changeMessage"
                control={control}
                placeholder="Why are you forking this template?"
                rows={3}
              />
            </FormControl>
          </FormGroup>
        </Form>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" form="fork-form" disabled={isSubmitting}>
            {isSubmitting ? "Forking..." : "Fork Template"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
