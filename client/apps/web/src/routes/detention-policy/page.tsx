import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Button } from "@trenova/shared/components/ui/button";
import { Form } from "@trenova/shared/components/ui/form";
import { detentionPolicySchema, type DetentionPolicy } from "@trenova/shared/types/detention";
import { zodResolver } from "@hookform/resolvers/zod";
import { FormProvider, useForm, type Resolver } from "react-hook-form";
import { DetentionPolicyForm } from "./_components/detention-policy-form";
import { DetentionPolicyPreview } from "./_components/detention-policy-preview";

const DEFAULT_POLICY: Partial<DetentionPolicy> = {
  name: "",
  code: "",
  description: "",
  status: "Draft",
  isOrgDefault: false,
  priority: 0,
  specificityScore: 0,
  appointmentStopsOnly: false,
  clockStartBasis: "LaterOfArrivalOrAppointment",
  lateArrivalRule: "NoEffect",
  lateArrivalGraceMinutes: 0,
  billingFreeMinutes: 120,
  minimumBillableMinutes: 0,
  billingIncrementMinutes: 15,
  roundingMode: "Up",
  rateSource: "Accessorial",
  accessorialChargeId: "",
  tiers: [],
  dayBoundaryMode: "PerStop",
  notificationRequirement: "None",
  notificationLeadMinutes: 30,
  notificationDeadlineMinutes: 0,
  unnotifiedBehavior: "Bill",
  autoSendNotice: false,
  sendDepartureSummary: false,
  currency: "USD",
  comments: "",
};

export function DetentionPolicyPage() {
  const form = useForm<DetentionPolicy>({
    resolver: zodResolver(detentionPolicySchema) as Resolver<DetentionPolicy>,
    defaultValues: DEFAULT_POLICY as DetentionPolicy,
    mode: "onChange",
  });

  const { control, handleSubmit } = form;

  return (
    <PageLayout
      pageHeaderProps={{
        title: "Detention Policy",
        description:
          "Encode the contract's detention terms and see what they would charge before they touch a shipment",
      }}
    >
      <FormProvider {...form}>
        <Form onSubmit={handleSubmit(() => undefined)}>
          <div className="grid gap-4 xl:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
            <DetentionPolicyForm />
            <div>
              <DetentionPolicyPreview control={control} />
            </div>
          </div>

          <div className="mt-4 flex justify-end gap-2">
            <Button type="submit">Save policy</Button>
          </div>
        </Form>
      </FormProvider>
    </PageLayout>
  );
}
