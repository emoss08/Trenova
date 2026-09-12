import { useT } from "@trenova/shared/i18n/use-t";
import { translate } from "@trenova/shared/i18n/runtime";
import { TabbedFormCreatePanel } from "@/components/tabbed-form-create-panel";
import { TabbedFormEditPanel, type FormTabConfig } from "@/components/tabbed-form-edit-panel";
import { Button } from "@trenova/shared/components/ui/button";
import type { EDICommunicationProfileRow } from "@/lib/graphql/edi-table";
import { apiService } from "@/services/api";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import type { EDIConnectionTestResult } from "@trenova/shared/types/edi";
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
      toast.warning(translate("Connection test passed with warnings"), { description: warnings });
      return;
    }
    toast.success(translate("Connection test passed"), { description: describe(["passed"]) });
    return;
  }
  toast.error(translate("Connection test failed"), {
    description: describe(["failed", "warning"]),
  });
}

type CommunicationProfileEditRow = CommunicationProfileFormValues &
  Pick<EDICommunicationProfileRow, "id" | "name" | "updatedAt" | "version">;

export function CommunicationProfilePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<EDICommunicationProfileRow>) {
  const t = useT();

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
    onError: () => toast.error(t("The connection test could not be run")),
  });

  const formTabs = useMemo<FormTabConfig[]>(
    () => [
      { value: "overview", label: t("Overview"), icon: ServerIcon, content: <OverviewTab /> },
      {
        value: "transport",
        label: t("Transport"),
        icon: RadioTowerIcon,
        content: <TransportTab />,
      },
      { value: "envelope", label: t("Envelope"), icon: ShieldCheckIcon, content: <EnvelopeTab /> },
      {
        value: "secrets",
        label: t("Secrets"),
        icon: KeyRoundIcon,
        content: <SecretsTab profile={profile} />,
      },
    ],
    [profile, t],
  );

  if (mode === "edit") {
    const editRow: CommunicationProfileEditRow | null = profile
      ? {
          ...getProfileFormDefaults(profile),
          id: profile.id,
          name: profile.name,
          updatedAt: profile.updatedAt,
          version: profile.version,
        }
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
        title={t("Communication Profile")}
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
              {t("Test Connection")}
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
      title={t("Communication Profile")}
      description={t(
        "Configure the transport profile and envelope values used for this organization.",
      )}
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
