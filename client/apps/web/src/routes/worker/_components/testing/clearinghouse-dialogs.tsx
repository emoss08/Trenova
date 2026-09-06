import { AutoCompleteDateField } from "@/components/fields/date-field/date-field";
import { InputField } from "@/components/fields/input-field";
import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  completeClearinghouseQuery,
  recordClearinghouseQuery,
  type ClearinghouseQueryRow,
} from "@/lib/graphql/worker-drug-alcohol";
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
import { getTodayDate } from "@trenova/shared/lib/date";
import {
  clearinghouseQueryTypeLabel,
  clearinghouseResultLabel,
} from "@trenova/shared/lib/drug-alcohol";
import {
  clearinghouseAnswerFormSchema,
  clearinghouseQueryFormSchema,
  clearinghouseQueryTypeSchema,
  clearinghouseResultSchema,
  type ClearinghouseAnswerFormValues,
  type ClearinghouseQueryFormValues,
} from "@trenova/shared/types/worker-drug-alcohol";
import { useEffect } from "react";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";
import { useTestingInvalidation } from "./use-testing-invalidation";

const TYPE_OPTIONS = clearinghouseQueryTypeSchema.options.map((value) => ({
  value,
  label: clearinghouseQueryTypeLabel(value),
}));
// Pending is the state a query starts in, not an answer anyone reports.
const RESULT_OPTIONS = clearinghouseResultSchema.options
  .filter((value) => value !== "Pending")
  .map((value) => ({ value, label: clearinghouseResultLabel(value) }));

export type RecordQueryDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
};

function emptyQuery(): ClearinghouseQueryFormValues {
  return {
    queryType: "AnnualLimited",
    requestedAt: getTodayDate(),
    consentObtainedAt: null,
    consentExpiresAt: null,
    reference: null,
    notes: null,
  };
}

export function RecordQueryDialog({ open, onOpenChange, workerId }: RecordQueryDialogProps) {
  const invalidate = useTestingInvalidation(workerId);
  const form = useForm<ClearinghouseQueryFormValues>({
    resolver: zodResolver(clearinghouseQueryFormSchema) as Resolver<ClearinghouseQueryFormValues>,
    defaultValues: emptyQuery(),
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) return;
    reset(emptyQuery());
  }, [open, reset]);

  const [queryType] = useWatch({ control, name: ["queryType"] });
  const isFull = queryType === "PreEmploymentFull" || queryType === "Full";

  const { mutateAsync, isPending } = useApiMutation<
    ClearinghouseQueryRow,
    ClearinghouseQueryFormValues,
    unknown,
    ClearinghouseQueryFormValues
  >({
    form,
    resourceName: "Clearinghouse query",
    mutationFn: (values) =>
      recordClearinghouseQuery({
        workerId,
        queryType: values.queryType,
        requestedAt: values.requestedAt,
        consentObtainedAt: values.consentObtainedAt ?? undefined,
        consentExpiresAt: values.consentExpiresAt ?? undefined,
        reference: values.reference ?? undefined,
        notes: values.notes ?? undefined,
      }),
    onSuccess: () => {
      toast.success("Query logged", {
        description: "Record the answer here once the Clearinghouse responds.",
      });
      void invalidate();
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>Log a Clearinghouse query</DialogTitle>
          <DialogDescription>
            A full query before the driver&apos;s first dispatch, and a limited query every twelve
            months after (49 CFR 382 Subpart G).
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
                <SelectField<ClearinghouseQueryFormValues>
                  control={control}
                  name="queryType"
                  label="Query"
                  options={TYPE_OPTIONS}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<ClearinghouseQueryFormValues>
                  control={control}
                  name="requestedAt"
                  label="Requested on"
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<ClearinghouseQueryFormValues>
                  control={control}
                  name="consentObtainedAt"
                  label="Consent obtained"
                  rules={{ required: isFull }}
                  description={
                    isFull ? "A full query cannot be run without the driver's consent." : undefined
                  }
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<ClearinghouseQueryFormValues>
                  control={control}
                  name="consentExpiresAt"
                  label="Consent expires"
                />
              </FormControl>
              <FormControl>
                <InputField<ClearinghouseQueryFormValues>
                  control={control}
                  name="reference"
                  label="Reference"
                  placeholder="Clearinghouse query reference"
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<ClearinghouseQueryFormValues>
                  control={control}
                  name="notes"
                  label="Notes"
                  maxLength={2000}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" isLoading={isPending}>
                Log query
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}

export type AnswerQueryDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  query: ClearinghouseQueryRow | null;
};

function emptyAnswer(): ClearinghouseAnswerFormValues {
  return {
    result: "NoViolations",
    completedAt: getTodayDate(),
    violationCount: 0,
    reference: null,
    notes: null,
  };
}

export function AnswerQueryDialog({ open, onOpenChange, workerId, query }: AnswerQueryDialogProps) {
  const invalidate = useTestingInvalidation(workerId);
  const form = useForm<ClearinghouseAnswerFormValues>({
    resolver: zodResolver(clearinghouseAnswerFormSchema) as Resolver<ClearinghouseAnswerFormValues>,
    defaultValues: emptyAnswer(),
  });
  const { control, handleSubmit, reset } = form;

  useEffect(() => {
    if (!open) return;
    reset(emptyAnswer());
  }, [open, reset]);

  const [result] = useWatch({ control, name: ["result"] });
  const foundViolations = result === "ViolationsFound";

  const { mutateAsync, isPending } = useApiMutation<
    ClearinghouseQueryRow,
    ClearinghouseAnswerFormValues,
    unknown,
    ClearinghouseAnswerFormValues
  >({
    form,
    resourceName: "Clearinghouse answer",
    mutationFn: (values) => {
      if (!query) throw new Error("No query selected");
      return completeClearinghouseQuery({
        queryId: query.id,
        result: values.result,
        completedAt: values.completedAt,
        violationCount: foundViolations ? values.violationCount : 0,
        reference: values.reference ?? undefined,
        notes: values.notes ?? undefined,
      });
    },
    onSuccess: (saved) => {
      toast.success("Answer recorded", {
        description:
          saved.result === "ViolationsFound" || saved.result === "ConsentDenied"
            ? "The driver is prohibited from safety-sensitive duty."
            : "The twelve-month clock restarts from today.",
      });
      void invalidate();
      onOpenChange(false);
    },
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>Record the Clearinghouse answer</DialogTitle>
          <DialogDescription>
            {query ? clearinghouseQueryTypeLabel(query.queryType) : null}
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
                <SelectField<ClearinghouseAnswerFormValues>
                  control={control}
                  name="result"
                  label="Answer"
                  options={RESULT_OPTIONS}
                  rules={{ required: true }}
                />
              </FormControl>
              <FormControl>
                <AutoCompleteDateField<ClearinghouseAnswerFormValues>
                  control={control}
                  name="completedAt"
                  label="Answered on"
                  rules={{ required: true }}
                />
              </FormControl>
              {foundViolations ? (
                <FormControl>
                  <NumberField<ClearinghouseAnswerFormValues>
                    control={control}
                    name="violationCount"
                    label="Violations returned"
                    rules={{ required: true }}
                  />
                </FormControl>
              ) : null}
              <FormControl>
                <InputField<ClearinghouseAnswerFormValues>
                  control={control}
                  name="reference"
                  label="Reference"
                />
              </FormControl>
              <FormControl cols="full">
                <TextareaField<ClearinghouseAnswerFormValues>
                  control={control}
                  name="notes"
                  label="Notes"
                  maxLength={2000}
                />
              </FormControl>
            </FormGroup>
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="submit" isLoading={isPending}>
                Record answer
              </Button>
            </DialogFooter>
          </Form>
        </FormProvider>
      </DialogContent>
    </Dialog>
  );
}
