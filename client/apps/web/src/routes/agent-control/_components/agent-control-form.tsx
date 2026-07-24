import { NumberField } from "@/components/fields/number-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormSaveDock } from "@/components/form-save-dock";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { fetchAgentControl, updateAgentControl } from "@/lib/graphql/agent-control";
import { agentControlSchema, type AgentControlFormValues } from "@/types/agent-control";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@trenova/shared/components/ui/card";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { useCallback } from "react";
import { FormProvider, type Resolver, useForm, useFormContext, useWatch } from "react-hook-form";
import { toast } from "sonner";

const AGENT_CONTROL_QUERY_KEY = ["agent-control"];

export default function AgentControlForm() {
  const queryClient = useQueryClient();
  const { data } = useSuspenseQuery({
    queryKey: AGENT_CONTROL_QUERY_KEY,
    queryFn: fetchAgentControl,
  });

  const defaultValues: AgentControlFormValues = {
    billingAgentEnabled: data.billingAgentEnabled,
    shadowMode: data.shadowMode,
    decisionTimeoutSeconds: data.decisionTimeoutSeconds,
  };

  const form = useForm<AgentControlFormValues>({
    resolver: zodResolver(agentControlSchema) as Resolver<AgentControlFormValues>,
    defaultValues,
    values: defaultValues,
  });
  const { handleSubmit, setError, reset } = form;

  const mutation = useApiMutation({
    mutationFn: (values: AgentControlFormValues) => updateAgentControl(values),
    onSuccess: (_, values) => {
      toast.success("Agent control updated");
      reset(values);
      void queryClient.invalidateQueries({ queryKey: AGENT_CONTROL_QUERY_KEY });
    },
    setFormError: setError,
    resourceName: "Agent Control",
  });

  const onSubmit = useCallback(
    (values: AgentControlFormValues) => mutation.mutate(values),
    [mutation],
  );

  return (
    <FormProvider {...form}>
      <Form onSubmit={handleSubmit(onSubmit)}>
        <div className="flex flex-col gap-4 pb-14">
          <BillingAgentCard />
          <FormSaveDock saveButtonContent="Save Changes" />
        </div>
      </Form>
    </FormProvider>
  );
}

function BillingAgentCard() {
  const { control } = useFormContext<AgentControlFormValues>();
  const shadowMode = useWatch({ control, name: "shadowMode" });
  const billingAgentEnabled = useWatch({ control, name: "billingAgentEnabled" });

  return (
    <Card>
      <CardHeader>
        <CardTitle>Billing Exception Agent</CardTitle>
        <CardDescription>
          The billing exception agent inspects blocked billing queue items, diagnoses why they are
          held, and proposes resolutions for a human to approve. It never approves or transitions an
          item itself.
        </CardDescription>
      </CardHeader>
      <CardContent className="max-w-prose">
        <FormGroup cols={1}>
          <FormControl>
            <SwitchField
              control={control}
              name="billingAgentEnabled"
              label="Enable Billing Exception Agent"
              description="Allow the agent to run against this organization's blocked billing queue items."
              position="left"
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="shadowMode"
              label="Shadow Mode"
              description="While on, the agent runs and stores its proposals for observation but they are never surfaced or actionable. Turn off only once you trust the agent's suggestions."
              position="left"
            />
          </FormControl>
          {shadowMode && billingAgentEnabled ? (
            <Alert variant="warning">
              <AlertTitle>Proposals are hidden</AlertTitle>
              <AlertDescription>
                Shadow mode is on, so runs complete and persist proposals but nothing appears for
                review and no decisions are awaited. Turn shadow mode off to surface proposals and
                enable human decisions.
              </AlertDescription>
            </Alert>
          ) : null}
          <FormControl className="max-w-[420px]">
            <NumberField
              control={control}
              name="decisionTimeoutSeconds"
              label="Decision Timeout (seconds)"
              description="How long a proposal waits for a human decision before its proposals expire and the run is parked. Defaults to 86400 (24 hours)."
              rules={{ required: true }}
            />
          </FormControl>
        </FormGroup>
      </CardContent>
    </Card>
  );
}
