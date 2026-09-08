import { NumberField } from "@/components/fields/number-field";
import { SelectField } from "@/components/fields/select-field";
import { FormSaveDock } from "@/components/form-save-dock";
import { useApiMutation } from "@/hooks/use-api-mutation";
import {
  fetchLeaveControl,
  LEAVE_CONTROL_KEY,
  updateLeaveControl,
} from "@/lib/graphql/worker-leave";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { measurementMethodHint, measurementMethodLabel } from "@trenova/shared/lib/leave";
import {
  leaveControlFormSchema,
  leaveMeasurementMethodSchema,
  type LeaveControlFormValues,
} from "@trenova/shared/types/worker-leave";
import { useCallback } from "react";
import { FormProvider, useForm, useFormContext, useWatch, type Resolver } from "react-hook-form";
import { toast } from "sonner";

const METHOD_OPTIONS = leaveMeasurementMethodSchema.options.map((value) => ({
  value,
  label: measurementMethodLabel(value),
  description: measurementMethodHint(value),
}));

export default function LeaveControlForm() {
  const queryClient = useQueryClient();
  const { data } = useSuspenseQuery({
    queryKey: [LEAVE_CONTROL_KEY],
    queryFn: ({ signal }) => fetchLeaveControl({ signal }),
  });

  const form = useForm<LeaveControlFormValues>({
    resolver: zodResolver(leaveControlFormSchema) as Resolver<LeaveControlFormValues>,
    defaultValues: {
      measurementMethod: data.measurementMethod,
      entitlementWeeks: Number(data.entitlementWeeks),
      militaryCaregiverWeeks: Number(data.militaryCaregiverWeeks),
      workweekHours: Number(data.workweekHours),
      eligibilityMonths: data.eligibilityMonths,
      eligibilityHours: data.eligibilityHours,
      certificationDueDays: data.certificationDueDays,
    },
  });
  const { handleSubmit, reset } = form;

  const mutation = useApiMutation({
    mutationFn: (values: LeaveControlFormValues) =>
      updateLeaveControl({
        measurementMethod: values.measurementMethod,
        entitlementWeeks: String(values.entitlementWeeks),
        militaryCaregiverWeeks: String(values.militaryCaregiverWeeks),
        workweekHours: String(values.workweekHours),
        eligibilityMonths: values.eligibilityMonths,
        eligibilityHours: values.eligibilityHours,
        certificationDueDays: values.certificationDueDays,
      }),
    onSuccess: (_, values) => {
      toast.success("Leave settings updated", {
        description:
          "Every balance is derived on read, so the change applies to existing cases as well as new ones.",
      });
      reset(values);
      void queryClient.invalidateQueries({ queryKey: [LEAVE_CONTROL_KEY] });
      void queryClient.invalidateQueries({ queryKey: ["worker-leave"] });
    },
    form,
    resourceName: "Leave Settings",
  });

  const onSubmit = useCallback(
    (values: LeaveControlFormValues) => mutation.mutate(values),
    [mutation],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4 pb-14">
          <MeasurementCard />
          <EntitlementCard />
          <EligibilityCard />
          <FormSaveDock saveButtonContent="Save Changes" />
        </div>
      </Form>
    </FormProvider>
  );
}

function MeasurementCard() {
  const { control } = useFormContext<LeaveControlFormValues>();
  const method = useWatch({ control, name: "measurementMethod" });
  return (
    <Card>
      <CardHeader>
        <CardTitle>The twelve-month period</CardTitle>
        <CardDescription>
          An employer picks one of the four methods in 29 CFR 825.200(b) and must apply it to every
          employee alike. Changing it is a change of policy, not a correction — employees are
          entitled to sixty days&apos; notice, and until then whichever method gives the greater
          benefit applies.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <FormGroup cols={1}>
          <FormControl>
            <SelectField<LeaveControlFormValues>
              control={control}
              name="measurementMethod"
              label="How the year is measured"
              options={METHOD_OPTIONS}
              placeholder="Choose a method"
              description="The twelve-month period every employee's entitlement is counted against."
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
        {measurementMethodHint(method) ? (
          <Alert variant={method === "RollingBackward" ? "info" : "warning"}>
            <AlertDescription>{measurementMethodHint(method)}</AlertDescription>
          </Alert>
        ) : null}
      </CardContent>
    </Card>
  );
}

function EntitlementCard() {
  const { control } = useFormContext<LeaveControlFormValues>();
  return (
    <Card>
      <CardHeader>
        <CardTitle>What the entitlement is worth</CardTitle>
        <CardDescription>
          Leave is granted in weeks and taken in hours, so a week needs a length. Intermittent leave
          draws on the same entitlement in whatever increment the organisation uses for any other
          absence (29 CFR 825.205).
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={3}>
          <FormControl>
            <NumberField<LeaveControlFormValues>
              control={control}
              name="entitlementWeeks"
              label="Weeks of leave"
              placeholder="12"
              description="Weeks of leave each eligible employee gets per period; the statute is a floor of twelve and a more generous figure is allowed."
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <NumberField<LeaveControlFormValues>
              control={control}
              name="militaryCaregiverWeeks"
              label="Military caregiver weeks"
              placeholder="26"
              description="Weeks allowed for military caregiver leave; twenty-six in a single twelve-month period under 29 CFR 825.127."
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <NumberField<LeaveControlFormValues>
              control={control}
              name="workweekHours"
              label="Hours in a workweek"
              placeholder="40"
              description="What one week of the entitlement converts to when leave is taken in hours."
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}

function EligibilityCard() {
  const { control } = useFormContext<LeaveControlFormValues>();
  return (
    <Card>
      <CardHeader>
        <CardTitle>Eligibility and certification</CardTitle>
        <CardDescription>
          The tenure half of the eligibility test is answered from the hire date. The hours-worked
          half cannot be — there is no timeclock here — so it is recorded on each case by hand.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <FormGroup cols={3}>
          <FormControl>
            <NumberField<LeaveControlFormValues>
              control={control}
              name="eligibilityMonths"
              label="Months of service"
              placeholder="12"
              description="Months since hire an employee needs before they qualify; twelve under 29 CFR 825.110(a), and they need not be consecutive."
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <NumberField<LeaveControlFormValues>
              control={control}
              name="eligibilityHours"
              label="Hours worked in the prior year"
              placeholder="1250"
              description="Hours an employee must have worked in the prior twelve months; 1,250 under the statute, shown beside each case for the office to check."
              rules={{ required: true }}
            />
          </FormControl>
          <FormControl>
            <NumberField<LeaveControlFormValues>
              control={control}
              name="certificationDueDays"
              label="Days to return a certification"
              placeholder="15"
              description="Calendar days an employee has to return a medical certification once it is requested; at least fifteen under 29 CFR 825.305(b)."
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
