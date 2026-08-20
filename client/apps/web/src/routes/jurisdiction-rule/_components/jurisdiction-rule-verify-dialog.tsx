import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { api } from "@trenova/shared/lib/api";
import {
  verifyJurisdictionRuleSchema,
  type JurisdictionRule,
  type VerifyJurisdictionRuleInput,
} from "@/types/jurisdiction-rule";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { toast } from "sonner";

const STATE_CHOICES = [
  { value: "Verified", label: "Verified — matches the issuing state" },
  { value: "Disputed", label: "Disputed — the state contradicts this" },
];

export function JurisdictionRuleVerifyDialog({
  open,
  onOpenChange,
  rule,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  rule: JurisdictionRule | null;
}) {
  const queryClient = useQueryClient();

  const form = useForm<VerifyJurisdictionRuleInput>({
    resolver: zodResolver(verifyJurisdictionRuleSchema),
    defaultValues: { verificationState: "Verified", sourceNote: "", sourceUrl: "" },
  });

  const { control, handleSubmit, reset } = form;

  // The dialog stays mounted while the operator moves between rows, and
  // defaultValues only apply on mount — without this the note written for
  // Georgia would be submitted against Tennessee.
  useEffect(() => {
    if (open) {
      reset({
        verificationState: "Verified",
        sourceNote: rule?.sourceNote ?? "",
        sourceUrl: rule?.sourceUrl ?? "",
      });
    }
  }, [open, rule, reset]);

  const mutation = useApiMutation({
    mutationFn: async (values: VerifyJurisdictionRuleInput) =>
      api.post(`/jurisdiction-rules/${rule?.id}/verify/`, values),
    onSuccess: () => {
      toast.success("Verification recorded", {
        description: "This rule now shows who confirmed it and when.",
      });
      void queryClient.invalidateQueries({ queryKey: ["jurisdiction-rule-list"] });
      onOpenChange(false);
    },
    form,
    resourceName: "Jurisdiction Rule",
  });

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        if (!next) reset();
        onOpenChange(next);
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Verify Jurisdiction Rule</DialogTitle>
          <DialogDescription>
            {rule?.state
              ? `${rule.state.name} (${rule.state.abbreviation})`
              : "Confirm this rule against the issuing state"}
          </DialogDescription>
        </DialogHeader>

        <p className="text-muted-foreground text-xs">
          This records that someone checked these limits against the state, and does not change
          them. If the numbers are wrong, edit the rule instead — that clears the verification on
          its own.
        </p>

        <FormProvider {...form}>
          <FormGroup cols={1}>
            <FormControl cols="full">
              <SelectField
                control={control}
                name="verificationState"
                label="Outcome"
                options={STATE_CHOICES}
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl cols="full">
              <TextareaField
                control={control}
                name="sourceNote"
                label="What you checked"
                description="At least 10 characters. The next person reading this row follows your note."
                placeholder="Checked against the state permit office handbook, rev 2026-01"
                rules={{ required: true }}
              />
            </FormControl>
            <FormControl cols="full">
              <InputField
                control={control}
                name="sourceUrl"
                label="Source URL"
                placeholder="https://"
              />
            </FormControl>
          </FormGroup>
        </FormProvider>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={mutation.isPending}
          >
            Cancel
          </Button>
          <Button
            type="button"
            onClick={handleSubmit((values) => mutation.mutate(values))}
            disabled={mutation.isPending}
          >
            {mutation.isPending ? "Recording..." : "Record Verification"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
