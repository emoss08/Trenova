import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useAuthStore } from "@/stores/auth-store";
import { usePermissionStore } from "@/stores/permission-store";
import type { DataTablePanelProps } from "@/types/data-table";
import type { CreateEDIConnectionRequest, EDIPartner } from "@/types/edi";
import type { OrganizationSelectOption } from "@/types/organization";
import { Operation, Resource } from "@/types/permission";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Building2Icon, GitBranchIcon, HandshakeIcon, ListChecksIcon } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import {
  createInternalPartnerPairSchema,
  ediPartnerFormSchema,
  getPartnerFormDefaults,
  toPartnerRequest,
  type CreateInternalPartnerPairFormValues,
  type EDIPartnerFormValues,
} from "../edi-schemas";
import { invalidateEDIConnections, invalidateEDIPartners } from "./edi-panel-invalidation";
import { InternalPartnerPairForm } from "./edi-internal-partner-pair-form";
import { MappingProfilePanel } from "./edi-mapping-profile-panel";
import { PartnerDetailsForm } from "./edi-partner-details-form";

export function PartnerPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<EDIPartner>) {
  if (mode === "create") {
    return <CreatePartnerPanel open={open} onOpenChange={onOpenChange} />;
  }

  return <PartnerEditPanel open={open} onOpenChange={onOpenChange} partner={row} />;
}

function CreatePartnerPanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const currentOrganizationId = useAuthStore((state) => state.user?.currentOrganizationId) ?? "";
  const [activeTab, setActiveTab] = useState("external");
  const externalForm = useForm<EDIPartnerFormValues>({
    resolver: zodResolver(ediPartnerFormSchema),
    defaultValues: getPartnerFormDefaults(),
    mode: "onChange",
  });
  const pairForm = useForm<CreateInternalPartnerPairFormValues>({
    resolver: zodResolver(createInternalPartnerPairSchema),
    defaultValues: getCreatePairDefaults(),
    mode: "onChange",
  });
  const { control: pairControl, reset, setValue } = pairForm;
  const targetCode = useWatch({ control: pairControl, name: "targetCode" });
  const targetName = useWatch({ control: pairControl, name: "targetName" });
  const { data: currentOrganization } = useQuery({
    ...queries.organization.detail(currentOrganizationId),
    enabled: open && Boolean(currentOrganizationId),
  });
  const createExternalMutation = useApiMutation({
    mutationFn: (values: EDIPartnerFormValues) =>
      apiService.ediService.createPartner(toPartnerRequest(values)),
    setFormError: externalForm.setError,
    resourceName: "EDI Partner",
    onSuccess: async () => {
      toast.success("External EDI partner created");
      externalForm.reset(getPartnerFormDefaults());
      onOpenChange(false);
      await invalidateEDIPartners(queryClient);
    },
  });
  const createConnectionMutation = useApiMutation({
    mutationFn: (values: CreateInternalPartnerPairFormValues) =>
      apiService.ediService.createConnection(toConnectionRequest(values)),
    setFormError: pairForm.setError,
    resourceName: "EDI Connection",
    onSuccess: async () => {
      toast.success("EDI connection requested");
      reset(getCreatePairDefaults());
      onOpenChange(false);
      await invalidateEDIPartners(queryClient);
      await invalidateEDIConnections(queryClient);
    },
  });

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      externalForm.reset(getPartnerFormDefaults());
      reset(getCreatePairDefaults());
      setActiveTab("external");
    }
    onOpenChange(nextOpen);
  };
  const fillCurrentOrganizationPartner = useCallback(() => {
    if (!currentOrganization) return;

    setValue("targetCode", currentOrganization.scacCode, { shouldDirty: true });
    setValue("targetName", currentOrganization.name, { shouldDirty: true });
  }, [currentOrganization, setValue]);
  const handleTargetOrganizationChange = useCallback(
    (organization: OrganizationSelectOption | null) => {
      if (!organization) {
        setValue("sourceCode", "", { shouldDirty: true });
        setValue("sourceName", "", { shouldDirty: true });
        return;
      }

      setValue("sourceCode", organization.scacCode ?? "", { shouldDirty: true });
      setValue("sourceName", organization.name, { shouldDirty: true });
      fillCurrentOrganizationPartner();
    },
    [fillCurrentOrganizationPartner, setValue],
  );

  useEffect(() => {
    if (!open || !currentOrganization) return;
    if (targetCode || targetName) return;

    fillCurrentOrganizationPartner();
  }, [currentOrganization, fillCurrentOrganizationPartner, open, targetCode, targetName]);

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={handleOpenChange}
      title="New EDI Partner"
      description="Create an external trading partner or request an internal organization connection."
      size="xl"
      footer={
        <>
          <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
            Cancel
          </Button>
          {activeTab === "external" ? (
            <Button
              type="submit"
              form="edi-create-external-partner-form"
              isLoading={createExternalMutation.isPending}
            >
              Create Partner
            </Button>
          ) : (
            <Button
              type="submit"
              form="edi-create-pair-form"
              isLoading={createConnectionMutation.isPending}
            >
              Request Connection
            </Button>
          )}
        </>
      }
    >
      <Tabs value={activeTab} onValueChange={setActiveTab} className="min-h-0">
        <TabsList variant="underline" className="w-full border-b border-border">
          <TabsTrigger value="external">
            <Building2Icon className="size-4" />
            External Partner
          </TabsTrigger>
          <TabsTrigger value="internal">
            <HandshakeIcon className="size-4" />
            Internal Connection
          </TabsTrigger>
        </TabsList>
        <TabsContent value="external" className="pt-4">
          <PartnerDetailsForm
            id="edi-create-external-partner-form"
            form={externalForm}
            disabled={false}
            readOnlyInternalFields={false}
            onSubmit={(values) => createExternalMutation.mutate(values)}
          />
        </TabsContent>
        <TabsContent value="internal" className="pt-4">
          <InternalPartnerPairForm
            id="edi-create-pair-form"
            form={pairForm}
            onSubmit={(submittedValues) => createConnectionMutation.mutate(submittedValues)}
            onTargetOrganizationChange={handleTargetOrganizationChange}
          />
        </TabsContent>
      </Tabs>
    </DataTablePanelContainer>
  );
}

function toConnectionRequest(
  values: CreateInternalPartnerPairFormValues,
): CreateEDIConnectionRequest {
  return {
    targetOrganizationId: values.targetOrganizationId,
    method: "Internal",
    capabilities: {
      loadTenderOutbound: true,
      loadTenderInbound: true,
      shipmentStatus: true,
      invoice: false,
    },
    sourcePartnerConfig: {
      code: values.sourceCode,
      name: values.sourceName,
      description: values.sourceDescription,
      contactName: values.sourceContactName,
      contactEmail: values.sourceContactEmail,
      contactPhone: values.sourceContactPhone,
      enabledForInbound: values.sourceEnabledForInbound,
      enabledForOutbound: values.sourceEnabledForOutbound,
      settings: values.sourceSettings,
    },
    targetPartnerConfig: {
      code: values.targetCode,
      name: values.targetName,
      description: values.targetDescription,
      contactName: values.targetContactName,
      contactEmail: values.targetContactEmail,
      contactPhone: values.targetContactPhone,
      enabledForInbound: values.targetEnabledForInbound,
      enabledForOutbound: values.targetEnabledForOutbound,
      settings: values.targetSettings,
    },
  };
}

function getCreatePairDefaults(): CreateInternalPartnerPairFormValues {
  return {
    targetOrganizationId: "",
    sourceCode: "",
    sourceName: "",
    sourceDescription: "",
    sourceContactName: "",
    sourceContactEmail: "",
    sourceContactPhone: "",
    sourceEnabledForInbound: true,
    sourceEnabledForOutbound: true,
    sourceSettings: {},
    targetCode: "",
    targetName: "",
    targetDescription: "",
    targetContactName: "",
    targetContactEmail: "",
    targetContactPhone: "",
    targetEnabledForInbound: true,
    targetEnabledForOutbound: true,
    targetSettings: {},
  };
}

function PartnerEditPanel({
  partner,
  open,
  onOpenChange,
}: {
  partner: EDIPartner | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const canUpdate = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Update),
  );
  const form = useForm<EDIPartnerFormValues>({
    resolver: zodResolver(ediPartnerFormSchema),
    defaultValues: getPartnerFormDefaults(partner),
    mode: "onChange",
  });

  useEffect(() => {
    if (open) {
      form.reset(getPartnerFormDefaults(partner));
    }
  }, [form, open, partner]);

  const mutation = useApiMutation({
    mutationFn: (values: EDIPartnerFormValues) => {
      if (!partner) {
        throw new Error("Partner is required");
      }
      return apiService.ediService.updatePartner(partner.id, toPartnerRequest(values));
    },
    setFormError: form.setError,
    resourceName: "EDI Partner",
    onSuccess: async () => {
      toast.success("EDI partner updated");
      await invalidateEDIPartners(queryClient);
    },
  });
  const handleClose = () => {
    onOpenChange(false);
  };
  const isInternal = partner?.kind === "Internal";

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={onOpenChange}
      title={partner?.name ?? "EDI Partner"}
      description="Partner settings and saved mapping profile."
      size="xl"
      footer={
        <>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          {partner && canUpdate && (
            <Button type="submit" form="edi-edit-partner-form" isLoading={mutation.isPending}>
              Save Partner
            </Button>
          )}
        </>
      }
    >
      {partner && (
        <Tabs defaultValue="details" className="min-h-0">
          <TabsList variant="underline" className="w-full border-b border-border">
            <TabsTrigger value="details">
              <ListChecksIcon className="size-4" />
              Details
            </TabsTrigger>
            <TabsTrigger value="mappings">
              <GitBranchIcon className="size-4" />
              Mappings
            </TabsTrigger>
          </TabsList>
          <TabsContent value="details" className="pt-4">
            <PartnerDetailsForm
              id="edi-edit-partner-form"
              form={form}
              disabled={!canUpdate}
              readOnlyInternalFields={isInternal}
              onSubmit={(values) => mutation.mutate(values)}
            />
          </TabsContent>
          <TabsContent value="mappings" className="pt-4">
            <MappingProfilePanel partnerId={partner.id} canUpdate={canUpdate} />
          </TabsContent>
        </Tabs>
      )}
    </DataTablePanelContainer>
  );
}
