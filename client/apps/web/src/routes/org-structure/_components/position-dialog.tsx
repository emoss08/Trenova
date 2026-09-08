import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  createJobPosition,
  HEADCOUNT_KEY,
  JOB_POSITIONS_KEY,
  updateJobPosition,
  type JobPositionRow,
} from "@/lib/graphql/org-structure";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient } from "@tanstack/react-query";
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
import { JOB_DEPARTMENT_ORDER, jobDepartmentLabel } from "@trenova/shared/lib/org-structure";
import {
  jobPositionFormSchema,
  type JobPositionFormValues,
} from "@trenova/shared/types/org-structure";
import { useEffect, useMemo } from "react";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { toast } from "sonner";

const DEPARTMENT_OPTIONS = JOB_DEPARTMENT_ORDER.map((value) => ({
  value,
  label: jobDepartmentLabel(value),
}));

const STATUS_OPTIONS = [
  { value: "Active", label: "Active" },
  { value: "Inactive", label: "Archived" },
];

export type PositionDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  position: JobPositionRow | null;
  positions: JobPositionRow[];
  /** Where a new position hangs when it is added from a node on the chart. */
  defaultReportsToPositionId?: string | null;
};

function defaultsFor(
  position: JobPositionRow | null,
  defaultReportsToPositionId: string | null,
): JobPositionFormValues {
  if (!position) {
    return {
      code: "",
      title: "",
      description: null,
      department: "Operations",
      flsaExempt: false,
      isDrivingPosition: true,
      reportsToPositionId: defaultReportsToPositionId,
      status: "Active",
    };
  }
  return {
    code: position.code,
    title: position.title,
    description: position.description ?? null,
    department: position.department as JobPositionFormValues["department"],
    flsaExempt: position.flsaExempt,
    isDrivingPosition: position.isDrivingPosition,
    reportsToPositionId: position.reportsToPositionId ?? null,
    status: position.status === "Active" ? "Active" : "Inactive",
  };
}

export function PositionDialog({
  open,
  onOpenChange,
  position,
  positions,
  defaultReportsToPositionId = null,
}: PositionDialogProps) {
  const queryClient = useQueryClient();
  const isEdit = Boolean(position);
  const form = useForm<JobPositionFormValues>({
    resolver: zodResolver(jobPositionFormSchema) as Resolver<JobPositionFormValues>,
    defaultValues: defaultsFor(position, defaultReportsToPositionId),
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) return;
    reset(defaultsFor(position, defaultReportsToPositionId));
  }, [open, position, defaultReportsToPositionId, reset]);

  // A position cannot report to itself, so it is not offered as its own parent.
  // The deeper cycles are the server's to refuse — it is the only thing that
  // can see the whole chain.
  const reportsToOptions = useMemo(
    () =>
      positions
        .filter((candidate) => candidate.id !== position?.id)
        .map((candidate) => ({ value: candidate.id, label: candidate.title })),
    [positions, position],
  );

  const { mutateAsync, isPending } = useApiMutation<
    { id: string },
    JobPositionFormValues,
    unknown,
    JobPositionFormValues
  >({
    form,
    resourceName: "Position",
    mutationFn: (values) => {
      const shared = {
        code: values.code,
        title: values.title,
        description: values.description ?? undefined,
        department: values.department,
        flsaExempt: values.flsaExempt,
        isDrivingPosition: values.isDrivingPosition,
        reportsToPositionId: values.reportsToPositionId ?? undefined,
        status: values.status,
      };
      return position
        ? updateJobPosition({ ...shared, id: position.id, version: position.version })
        : createJobPosition(shared);
    },
    onSuccess: () => {
      toast.success(isEdit ? "Position updated" : "Position added");
      void queryClient.invalidateQueries({ queryKey: [JOB_POSITIONS_KEY] });
      void queryClient.invalidateQueries({ queryKey: [HEADCOUNT_KEY] });
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit the position" : "Add a position"}</DialogTitle>
          <DialogDescription>
            A terminal is where somebody works; a position is what they do. Headcount is read both
            ways.
          </DialogDescription>
        </DialogHeader>
        <FormProvider {...form}>
          <Form
            onSubmit={(submitEvent) => {
              submitEvent.preventDefault();
              submitEvent.stopPropagation();
              void handleSubmit((values) => mutateAsync(values))(submitEvent);
            }}
          >
            <FormGroup className="pb-2" cols={2}>
              <FormControl>
                <InputField<JobPositionFormValues>
                  control={control}
                  name="code"
                  label="Code"
                  placeholder="e.g. DRV-OTR"
                  rules={{ required: true }}
                  description="A short identifier that must be unique across the organisation, whatever its case."
                />
              </FormControl>
              <FormControl>
                <InputField<JobPositionFormValues>
                  control={control}
                  name="title"
                  label="Title"
                  placeholder="e.g. Over-the-Road Driver"
                  description="The name of the job as it appears on the chart and on each holder's record."
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <SelectField<JobPositionFormValues>
                  control={control}
                  name="department"
                  label="Department"
                  options={DEPARTMENT_OPTIONS}
                  placeholder="Pick a department"
                  description="The part of the business the position sits in."
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <SelectField<JobPositionFormValues>
                  control={control}
                  name="status"
                  label="Status"
                  options={STATUS_OPTIONS}
                  placeholder="Pick a status"
                  description="Archiving is refused while anybody still holds the position."
                />
              </FormControl>
              <FormControl cols="full">
                <SelectField<JobPositionFormValues>
                  control={control}
                  name="reportsToPositionId"
                  label="Reports to"
                  options={reportsToOptions}
                  placeholder="Pick a position"
                  isClearable
                  description="The position this one answers to; leave it empty for the top of the chart. A person's own manager is set on their record, because two people in the same position can report to different managers."
                />
              </FormControl>
              <FormControl>
                <SwitchField<JobPositionFormValues>
                  control={control}
                  name="isDrivingPosition"
                  label="Driving position"
                  description="Needs a CDL and is filled from the worker roster; a front-office position is filled by people who log in. This is the line most compliance rules are drawn along."
                />
              </FormControl>
              <FormControl>
                <SwitchField<JobPositionFormValues>
                  control={control}
                  name="flsaExempt"
                  label="Exempt from overtime"
                  description="Under the Fair Labor Standards Act. Recorded per position because that is where the duties test is applied."
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<JobPositionFormValues>
                  control={control}
                  name="description"
                  label="Description"
                  placeholder="e.g. Runs regional lanes out of the home terminal on a five-day schedule"
                  description="Optional notes on the duties and expectations of the role."
                  maxLength={2000}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" isLoading={isPending}>
                {isEdit ? "Save" : "Add"}
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
