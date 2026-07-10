import { TabbedFormCreatePanel } from "@/components/tabbed-form-create-panel";
import { TabbedFormEditPanel, type FormTabConfig } from "@/components/tabbed-form-edit-panel";
import { Button } from "@/components/ui/button";
import { apiService } from "@/services/api";
import type { DataTablePanelProps } from "@/types/data-table";
import type { EDICommunicationProfile, EDIConnectionTestResult } from "@/types/edi";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { KeyRoundIcon, RadioTowerIcon, ServerIcon, ShieldCheckIcon } from "lucide-react";
import { useMemo } from "react";
import { useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import {
  communicationProfileFormSchema,
  type CommunicationProfileFormValues,
} from "../edi-schemas";
import {
  EnvelopeTab,
  OverviewTab,
  SecretsTab,
  TransportTab,
} from "./edi-communication-profile-form-content";
import {
  getProfileFormDefaults,
  toCommunicationProfileRequest,
} from "./edi-communication-profile-form";
import { invalidateEDICommunicationProfiles } from "./edi-panel-invalidation";

function notifyConnectionTestResult(result: EDIConnectionTestResult) {
  const describe = (statuses: string[]) =>
    result.checks
      .filter((check) => statuses.includes(check.status))
      .map((check) => `${check.name}: ${check.message ?? check.status}`)
      .join("\n");
  if (result.success) {
    const warnings = describe(["warning"]);
    if (warnings) {
      toast.warning("Connection test passed with warnings", { description: warnings });
      return;
    }
    toast.success("Connection test passed", { description: describe(["passed"]) });
    return;
  }
  toast.error("Connection test failed", { description: describe(["failed", "warning"]) });
}

export function CommunicationProfilePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<EDICommunicationProfile>) {
  const queryClient = useQueryClient();
  const profile = mode === "edit" ? row : null;
  const form = useForm<CommunicationProfileFormValues>({
    resolver: zodResolver(communicationProfileFormSchema),
    defaultValues: getProfileFormDefaults(profile),
    mode: "onChange",
  });
  const method = useWatch({ control: form.control, name: "method" });
  const isDirty = form.formState.isDirty;

  const testConnectionMutation = useMutation({
    mutationFn: () => {
      if (!profile) throw new Error("Profile is required");
      return apiService.ediService.testProfileConnection(profile.id);
    },
    onSuccess: (result) => notifyConnectionTestResult(result),
    onError: () => toast.error("The connection test could not be run"),
  });

  const formTabs = useMemo<FormTabConfig[]>(
    () => [
      { value: "overview", label: "Overview", icon: ServerIcon, content: <OverviewTab /> },
      { value: "transport", label: "Transport", icon: RadioTowerIcon, content: <TransportTab /> },
      { value: "envelope", label: "Envelope", icon: ShieldCheckIcon, content: <EnvelopeTab /> },
      {
        value: "secrets",
        label: "Secrets",
        icon: KeyRoundIcon,
        content: <SecretsTab profile={profile} />,
      },
    ],
    [profile],
  );

  if (mode === "edit") {
    const editRow = profile
      ? ({
          ...getProfileFormDefaults(profile),
          id: profile.id,
          name: profile.name,
          updatedAt: profile.updatedAt,
          version: profile.version,
        } as unknown as EDICommunicationProfile)
      : null;

    const canTestConnection = method !== "Internal";

    return (
      <TabbedFormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={editRow}
        form={form}
        url="/edi/communication-profiles/"
        queryKey="edi-communication-profile-list"
        title="Communication Profile"
        fieldKey="name"
        size="xl"
        formTabs={formTabs}
        headerActions={
          canTestConnection ? (
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => testConnectionMutation.mutate()}
              isLoading={testConnectionMutation.isPending}
              disabled={isDirty}
              title={
                isDirty
                  ? "Save your changes before testing the connection"
                  : "Verify certificates, credentials, and endpoint reachability"
              }
            >
              Test Connection
            </Button>
          ) : undefined
        }
        mutationFn={async (values) => {
          const result = await apiService.ediService.updateCommunicationProfile(
            profile!.id,
            toCommunicationProfileRequest(values, profile),
          );
          await invalidateEDICommunicationProfiles(queryClient);
          return result as unknown as CommunicationProfileFormValues;
        }}
      />
    );
  }

  return (
    <TabbedFormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/edi/communication-profiles/"
      queryKey="edi-communication-profile-list"
      title="Communication Profile"
      description="Configure the transport profile and envelope values used for this organization."
      size="xl"
      formTabs={formTabs}
      mutationFn={async (values) => {
        const result = await apiService.ediService.createCommunicationProfile(
          toCommunicationProfileRequest(values, null),
        );
        await invalidateEDICommunicationProfiles(queryClient);
        return result as unknown as CommunicationProfileFormValues;
      }}
    />
  );
}
