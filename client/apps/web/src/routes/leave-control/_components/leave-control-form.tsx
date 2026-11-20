import { SettingsForm } from "@/components/settings/settings-form";
import { updateLeaveControl, WORKER_LEAVE_KEY } from "@/lib/graphql/worker-leave";
import { leaveControl } from "@/lib/queries/leave-control";
import { useQueryClient, useSuspenseQuery } from "@tanstack/react-query";
import { leaveControlSchema, type LeaveControl } from "@trenova/shared/types/worker-leave";
import { useLeaveControlTabs, type LeaveControlTab } from "../leave-control-tabs";
import { PolicyTab } from "./policy-tab";

/**
 * Leave settings: one form, saved once. A save changes every worker's balance, which is
 * measured against these settings, so their leave files are read again after it.
 */
export default function LeaveControlForm() {
  const queryClient = useQueryClient();
  const { data } = useSuspenseQuery({ ...leaveControl.get() });
  const tabs = useLeaveControlTabs();

  const save = async (values: LeaveControl) => {
    const saved = await updateLeaveControl(values);
    void queryClient.invalidateQueries({ queryKey: [WORKER_LEAVE_KEY] });
    return saved;
  };

  return (
    <SettingsForm<LeaveControl, LeaveControlTab>
      values={data}
      schema={leaveControlSchema}
      queryKey={leaveControl.get._def}
      save={save}
      resourceName="Leave Settings"
      tabs={tabs}
      content={(tab) => {
        switch (tab) {
          case "policy":
            return <PolicyTab />;
        }
      }}
    />
  );
}
