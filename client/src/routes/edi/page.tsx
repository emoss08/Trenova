import { OrganizationAutocompleteField } from "@/components/autocomplete-fields";
import { DataTable } from "@/components/data-table/data-table";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { DataTablePlaceholder } from "@/components/data-table/_components/data-table-components";
import { InputField } from "@/components/fields/input-field";
import { SwitchField } from "@/components/fields/switch-field";
import { HoverCardTimestamp } from "@/components/hover-card-timestamp";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Form,
  FormControl,
  FormGroup,
  FormSection,
} from "@/components/ui/form";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Switch } from "@/components/ui/switch";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { api } from "@/lib/api";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { useAuthStore } from "@/stores/auth-store";
import { usePermissionStore } from "@/stores/permission-store";
import type {
  CreateInternalPartnerPairRequest,
  EDIMappingEntityType,
  EDIMappingProfileItem,
  EDIPartner,
  EDITransfer,
} from "@/types/edi";
import type { DataTablePanelProps } from "@/types/data-table";
import type { OrganizationSelectOption } from "@/types/organization";
import { Operation, Resource } from "@/types/permission";
import type { GenericLimitOffsetResponse } from "@/types/server";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { ColumnDef } from "@tanstack/react-table";
import { CheckIcon, EyeIcon, LinkIcon, Trash2Icon, XIcon } from "lucide-react";
import { type Control, useForm } from "react-hook-form";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router";
import { toast } from "sonner";

const mappingEntityTypes = [
  "Customer",
  "ServiceType",
  "ShipmentType",
  "FormulaTemplate",
  "Location",
  "Commodity",
  "AccessorialCharge",
] as const;

const mappingTargetEndpoints: Record<(typeof mappingEntityTypes)[number], string> = {
  Customer: "/customers/select-options/",
  ServiceType: "/service-types/select-options/",
  ShipmentType: "/shipment-types/select-options/",
  FormulaTemplate: "/formula-templates/select-options/",
  Location: "/locations/select-options/",
  Commodity: "/commodities/select-options/",
  AccessorialCharge: "/accessorial-charges/select-options/",
};

type SelectOption = {
  id: string;
  name?: string;
  code?: string;
  description?: string;
};

type EDIPageKind = "partners" | "inbound" | "outbound";

export function EDIPartnersPage() {
  return <EDIPage kind="partners" />;
}

export function EDIInboundTransfersPage() {
  return <EDIPage kind="inbound" />;
}

export function EDIOutboundTransfersPage() {
  return <EDIPage kind="outbound" />;
}

function EDIPage({ kind }: { kind: EDIPageKind }) {
  const titles = {
    partners: "EDI Partners",
    inbound: "Inbound EDI Transfers",
    outbound: "Outbound EDI Transfers",
  };

  return (
    <PageLayout
      pageHeaderProps={{
        title: titles[kind],
        description: "Internal load tender exchange, mapping, and lifecycle visibility",
      }}
    >
      {kind === "partners" ? <PartnersWorkspace /> : <TransfersWorkspace direction={kind} />}
    </PageLayout>
  );
}

function PartnersWorkspace() {
  const columns = useMemoPartnerColumns();

  return (
    <DataTable<EDIPartner>
      name="EDI Partner Pair"
      link="/edi/partners/"
      queryKey="edi-partner-list"
      exportModelName="edi-partner"
      resource={Resource.EDI}
      columns={columns}
      TablePanel={PartnerPanel}
      preferDetailRowForEdit
    />
  );
}

function useMemoPartnerColumns() {
  return useMemo<ColumnDef<EDIPartner>[]>(
    () => [
      {
        accessorKey: "code",
        header: "Code",
        cell: ({ row }) => <span className="font-medium">{row.original.code}</span>,
        size: 140,
        meta: {
          label: "Code",
          apiField: "code",
          filterable: true,
          sortable: true,
          filterType: "text",
          defaultFilterOperator: "contains",
        },
      },
      {
        accessorKey: "name",
        header: "Name",
        cell: ({ row }) => row.original.name,
        size: 220,
        meta: {
          label: "Name",
          apiField: "name",
          filterable: true,
          sortable: true,
          filterType: "text",
          defaultFilterOperator: "contains",
        },
      },
      {
        accessorKey: "internalOrganization.name",
        header: "Target Organization",
        cell: ({ row }) =>
          row.original.internalOrganization?.name ??
          row.original.internalOrganizationId ?? <DataTablePlaceholder />,
        size: 240,
        meta: {
          label: "Target Organization",
          apiField: "internalOrganizationId",
          filterable: false,
          sortable: false,
        },
      },
      {
        id: "direction",
        header: "Direction",
        cell: ({ row }) => (
          <div className="flex gap-1">
            <Badge variant={row.original.enabledForInbound ? "secondary" : "outline"}>Inbound</Badge>
            <Badge variant={row.original.enabledForOutbound ? "secondary" : "outline"}>Outbound</Badge>
          </div>
        ),
        size: 180,
        meta: {
          label: "Direction",
          apiField: "direction",
          filterable: false,
          sortable: false,
        },
      },
      {
        accessorKey: "status",
        header: "Status",
        cell: ({ row }) => (
          <Badge variant={row.original.status === "Active" ? "active" : "outline"}>
            {row.original.status}
          </Badge>
        ),
        size: 120,
        meta: {
          label: "Status",
          apiField: "status",
          filterable: true,
          sortable: true,
          filterType: "select",
          filterOptions: [
            { label: "Active", value: "Active" },
            { label: "Inactive", value: "Inactive" },
          ],
          defaultFilterOperator: "eq",
        },
      },
      {
        accessorKey: "updatedAt",
        header: "Updated",
        cell: ({ row }) => <HoverCardTimestamp timestamp={row.original.updatedAt ?? undefined} />,
        size: 180,
        meta: {
          label: "Updated",
          apiField: "updatedAt",
          filterable: false,
          sortable: true,
          filterType: "date",
        },
      },
    ],
    [],
  );
}

function PartnerPanel({ open, onOpenChange, mode, row }: DataTablePanelProps<EDIPartner>) {
  if (mode === "create") {
    return <CreatePairPanel open={open} onOpenChange={onOpenChange} />;
  }

  return <PartnerEditPanel open={open} onOpenChange={onOpenChange} partner={row} />;
}

function CreatePairPanel({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const currentOrganizationId = useAuthStore((state) => state.user?.currentOrganizationId) ?? "";
  const form = useForm<CreateInternalPartnerPairRequest>({
    defaultValues: getCreatePairDefaults(),
  });
  const { control, handleSubmit, reset, setValue, watch } = form;
  const values = watch();
  const { data: currentOrganization } = useQuery({
    queryKey: ["organization", "edi-current", currentOrganizationId],
    queryFn: () => apiService.organizationService.getByID(currentOrganizationId),
    enabled: open && Boolean(currentOrganizationId),
  });
  const mutation = useMutation({
    mutationFn: (values: CreateInternalPartnerPairRequest) =>
      apiService.ediService.createInternalPair(values),
    onSuccess: async () => {
      toast.success("Internal EDI partner pair created");
      reset(getCreatePairDefaults());
      onOpenChange(false);
      await queryClient.invalidateQueries({ queryKey: ["edi-partner-list"] });
      await queryClient.invalidateQueries({ queryKey: queries.edi.partners._def });
    },
    onError: () => toast.error("Failed to create partner pair"),
  });
  const canSubmit =
    values.targetOrganizationId &&
    values.sourceCode &&
    values.sourceName &&
    values.targetCode &&
    values.targetName;

  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) {
      reset(getCreatePairDefaults());
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
    if (values.targetCode || values.targetName) return;

    fillCurrentOrganizationPartner();
  }, [currentOrganization, fillCurrentOrganizationPartner, open, values.targetCode, values.targetName]);

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={handleOpenChange}
      title="Create Internal Partner Pair"
      description="Configure both sides of the reciprocal relationship."
      size="xl"
      footer={
        <>
          <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
            Cancel
          </Button>
          <Button
            type="submit"
            form="edi-create-pair-form"
            disabled={!canSubmit}
            isLoading={mutation.isPending}
          >
            Create Pair
          </Button>
        </>
      }
    >
        <Form
          id="edi-create-pair-form"
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.stopPropagation();
            void handleSubmit((submittedValues) => mutation.mutate(submittedValues))(event);
          }}
        >
          <FormGroup cols={2} className="gap-x-5 gap-y-3">
            <FormControl cols="full">
              <OrganizationAutocompleteField
                control={control}
                name="targetOrganizationId"
                label="Target Organization"
                placeholder="Select organization"
                description="Only organizations in the current business unit are available."
                rules={{ required: true }}
                extraSearchParams={{
                  scope: "business-unit",
                  excludeCurrent: "true",
                }}
                onOptionChange={handleTargetOrganizationChange}
              />
            </FormControl>
            <PartnerSideFields
              title="Current Organization View"
              prefix="source"
              control={control}
            />
            <PartnerSideFields
              title="Target Organization View"
              prefix="target"
              control={control}
            />
          </FormGroup>
        </Form>
    </DataTablePanelContainer>
  );
}

function getCreatePairDefaults(): CreateInternalPartnerPairRequest {
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

function PartnerSideFields({
  title,
  prefix,
  control,
}: {
  title: string;
  prefix: "source" | "target";
  control: Control<CreateInternalPartnerPairRequest>;
}) {
  const codeName = `${prefix}Code` as const;
  const partnerName = `${prefix}Name` as const;
  const contactName = `${prefix}ContactName` as const;
  const contactEmail = `${prefix}ContactEmail` as const;
  const contactPhone = `${prefix}ContactPhone` as const;
  const inboundName = `${prefix}EnabledForInbound` as const;
  const outboundName = `${prefix}EnabledForOutbound` as const;

  return (
    <FormSection title={title} className="rounded-md border bg-muted/20 p-3">
      <FormGroup cols={2}>
        <FormControl>
          <InputField
            control={control}
            name={codeName}
            label="Partner Code"
            placeholder="Partner code"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name={partnerName}
            label="Partner Name"
            placeholder="Partner name"
            rules={{ required: true }}
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name={contactName}
            label="Contact Name"
            placeholder="Contact name"
          />
        </FormControl>
        <FormControl>
          <InputField
            control={control}
            name={contactEmail}
            label="Contact Email"
            placeholder="ops@example.com"
          />
        </FormControl>
        <FormControl cols="full">
          <InputField
            control={control}
            name={contactPhone}
            label="Contact Phone"
            placeholder="Contact phone"
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name={inboundName}
            label="Inbound Enabled"
            description="Allow this partner record to receive load tenders."
            outlined
          />
        </FormControl>
        <FormControl>
          <SwitchField
            control={control}
            name={outboundName}
            label="Outbound Enabled"
            description="Allow this partner record to send load tenders."
            outlined
          />
        </FormControl>
      </FormGroup>
    </FormSection>
  );
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
  const canUpdate = usePermissionStore((state) => state.hasPermission(Resource.EDI, Operation.Update));
  const [draft, setDraft] = useState<EDIPartner | null>(partner);

  useEffect(() => {
    if (open) {
      setDraft(partner);
    }
  }, [open, partner]);

  const mutation = useMutation({
    mutationFn: (values: EDIPartner) => apiService.ediService.updatePartner(values),
    onSuccess: async () => {
      toast.success("EDI partner updated");
      await queryClient.invalidateQueries({ queryKey: ["edi-partner-list"] });
      await queryClient.invalidateQueries({ queryKey: queries.edi.partners._def });
    },
    onError: () => toast.error("Failed to update EDI partner"),
  });
  const handleClose = () => {
    onOpenChange(false);
    setDraft(null);
  };

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
          {draft && canUpdate && (
            <Button
              isLoading={mutation.isPending}
              onClick={() => mutation.mutate(draft)}
            >
              Save Partner
            </Button>
          )}
        </>
      }
    >
      {draft && (
        <Tabs defaultValue="details" className="min-h-0">
          <TabsList>
            <TabsTrigger value="details">Details</TabsTrigger>
            <TabsTrigger value="mappings">Mappings</TabsTrigger>
          </TabsList>
          <TabsContent value="details" className="pt-3">
            <div className="grid gap-3 md:grid-cols-2">
              <Input
                value={draft.code}
                disabled={!canUpdate}
                placeholder="Partner code"
                onChange={(event) => setDraft({ ...draft, code: event.target.value })}
              />
              <Input
                value={draft.name}
                disabled={!canUpdate}
                placeholder="Partner name"
                onChange={(event) => setDraft({ ...draft, name: event.target.value })}
              />
              <Input
                value={draft.contactName ?? ""}
                disabled={!canUpdate}
                placeholder="Contact name"
                onChange={(event) => setDraft({ ...draft, contactName: event.target.value })}
              />
              <Input
                value={draft.contactEmail ?? ""}
                disabled={!canUpdate}
                placeholder="Contact email"
                onChange={(event) => setDraft({ ...draft, contactEmail: event.target.value })}
              />
              <Input
                value={draft.contactPhone ?? ""}
                disabled={!canUpdate}
                placeholder="Contact phone"
                onChange={(event) => setDraft({ ...draft, contactPhone: event.target.value })}
              />
              <div className="flex items-center justify-between gap-2 rounded-md border p-2 text-sm">
                Inbound enabled
                <Switch
                  checked={draft.enabledForInbound}
                  disabled={!canUpdate}
                  onCheckedChange={(checked) => setDraft({ ...draft, enabledForInbound: checked })}
                />
              </div>
              <div className="flex items-center justify-between gap-2 rounded-md border p-2 text-sm">
                Outbound enabled
                <Switch
                  checked={draft.enabledForOutbound}
                  disabled={!canUpdate}
                  onCheckedChange={(checked) => setDraft({ ...draft, enabledForOutbound: checked })}
                />
              </div>
            </div>
          </TabsContent>
          <TabsContent value="mappings" className="pt-3">
            <MappingProfilePanel partnerId={draft.id} canUpdate={canUpdate} />
          </TabsContent>
        </Tabs>
      )}
    </DataTablePanelContainer>
  );
}

function MappingProfilePanel({ partnerId, canUpdate }: { partnerId: string; canUpdate: boolean }) {
  const queryClient = useQueryClient();
  const { data } = useQuery(queries.edi.mappingProfile(partnerId));
  const [draft, setDraft] = useState<EDIMappingProfileItem>({
    entityType: "Customer",
    sourceId: "",
    sourceLabel: "",
    targetId: "",
    targetLabel: "",
  });
  const saveMutation = useMutation({
    mutationFn: (item: EDIMappingProfileItem) => apiService.ediService.saveMappingProfile(partnerId, [item]),
    onSuccess: async () => {
      toast.success("Mapping saved");
      setDraft((current) => ({ ...current, sourceId: "", sourceLabel: "", targetId: "", targetLabel: "" }));
      await queryClient.invalidateQueries({ queryKey: queries.edi.mappingProfile(partnerId).queryKey });
    },
    onError: () => toast.error("Failed to save mapping"),
  });
  const deleteMutation = useMutation({
    mutationFn: (itemId: string) => apiService.ediService.deleteMappingItem(partnerId, itemId),
    onSuccess: async () => {
      toast.success("Mapping deleted");
      await queryClient.invalidateQueries({ queryKey: queries.edi.mappingProfile(partnerId).queryKey });
    },
    onError: () => toast.error("Failed to delete mapping"),
  });

  return (
    <Tabs defaultValue="Customer" className="gap-3">
      <TabsList className="flex-wrap">
        {mappingEntityTypes.map((entityType) => (
          <TabsTrigger key={entityType} value={entityType}>{entityType}</TabsTrigger>
        ))}
      </TabsList>
      {mappingEntityTypes.map((entityType) => {
        const entries = (data?.entries ?? []).filter((entry) => entry.entityType === entityType);
        return (
          <TabsContent key={entityType} value={entityType} className="flex flex-col gap-3">
            {canUpdate && (
              <div className="grid gap-2 md:grid-cols-5">
                <Input placeholder="Source ID" value={draft.entityType === entityType ? draft.sourceId : ""} onChange={(event) => setDraft({ ...draft, entityType, sourceId: event.target.value })} />
                <Input placeholder="Source label" value={draft.entityType === entityType ? draft.sourceLabel ?? "" : ""} onChange={(event) => setDraft({ ...draft, entityType, sourceLabel: event.target.value })} />
                <TargetLookup entityType={entityType} value={draft.entityType === entityType ? draft.targetId : ""} onChange={(option) => setDraft({ ...draft, entityType, targetId: option?.id ?? "", targetLabel: getOptionLabel(option) })} />
                <Input placeholder="Target label" value={draft.entityType === entityType ? draft.targetLabel ?? "" : ""} onChange={(event) => setDraft({ ...draft, entityType, targetLabel: event.target.value })} />
                <Button disabled={!draft.sourceId || !draft.targetId || draft.entityType !== entityType} onClick={() => saveMutation.mutate(draft)}>
                  <CheckIcon data-icon="inline-start" />
                  Save
                </Button>
              </div>
            )}
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Source</TableHead>
                    <TableHead>Target</TableHead>
                    <TableHead />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {entries.map((entry) => (
                    <TableRow key={entry.id ?? `${entry.entityType}-${entry.sourceId}`}>
                      <TableCell>{entry.sourceLabel || entry.sourceId}</TableCell>
                      <TableCell>{entry.targetLabel || entry.targetId}</TableCell>
                      <TableCell className="text-right">
                        {canUpdate && entry.id && (
                          <Button variant="ghost" size="icon-sm" onClick={() => deleteMutation.mutate(entry.id!)}>
                            <Trash2Icon />
                          </Button>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                  {entries.length === 0 && (
                    <TableRow>
                      <TableCell colSpan={3} className="h-16 text-center text-muted-foreground">
                        No mappings saved for {entityType}.
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </div>
          </TabsContent>
        );
      })}
    </Tabs>
  );
}

function TransfersWorkspace({ direction }: { direction: "inbound" | "outbound" }) {
  const [selectedTransfer, setSelectedTransfer] = useState<EDITransfer | null>(null);
  const { data, isLoading } = useQuery({
    queryKey: ["edi", "transfers", direction, "?limit=100"],
    queryFn: () =>
      direction === "inbound"
        ? apiService.ediService.listInboundTransfers("?limit=100")
        : apiService.ediService.listOutboundTransfers("?limit=100"),
  });

  return (
    <div className="flex flex-col gap-3">
      <div className="rounded-md border bg-background">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Status</TableHead>
              <TableHead>Partner</TableHead>
              <TableHead>Source Shipment</TableHead>
              <TableHead>Submitted</TableHead>
              <TableHead>Target Shipment</TableHead>
              <TableHead />
            </TableRow>
          </TableHeader>
          <TableBody>
            {(data?.results ?? []).map((transfer) => (
              <TableRow key={transfer.id}>
                <TableCell><TransferStatusBadge status={transfer.status} /></TableCell>
                <TableCell>{direction === "inbound" ? transfer.sourcePartner?.name : transfer.targetPartner?.name}</TableCell>
                <TableCell>{transfer.sourceShipmentId}</TableCell>
                <TableCell>{formatUnix(transfer.submittedAt)}</TableCell>
                <TableCell>
                  {transfer.targetShipmentId ? (
                    <Link className="inline-flex items-center gap-1 text-primary underline-offset-4 hover:underline" to={`/shipment-management/shipments?item=${transfer.targetShipmentId}`}>
                      <LinkIcon className="size-3.5" />
                      {transfer.targetShipmentId}
                    </Link>
                  ) : "—"}
                </TableCell>
                <TableCell className="text-right">
                  <Button variant="ghost" size="sm" onClick={() => setSelectedTransfer(transfer)}>
                    <EyeIcon data-icon="inline-start" />
                    Details
                  </Button>
                </TableCell>
              </TableRow>
            ))}
            {!isLoading && (data?.results?.length ?? 0) === 0 && (
              <TableRow>
                <TableCell colSpan={6} className="h-24 text-center text-muted-foreground">
                  No {direction} EDI transfers yet.
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      <TransferSheet
        transfer={selectedTransfer}
        direction={direction}
        onOpenChange={(open) => {
          if (!open) setSelectedTransfer(null);
        }}
      />
    </div>
  );
}

function TransferSheet({
  transfer,
  direction,
  onOpenChange,
}: {
  transfer: EDITransfer | null;
  direction: "inbound" | "outbound";
  onOpenChange: (open: boolean) => void;
}) {
  const queryClient = useQueryClient();
  const [rejectReason, setRejectReason] = useState("");
  const [inlineMappings, setInlineMappings] = useState<Record<string, EDIMappingProfileItem>>({});
  const canUpdate = usePermissionStore((state) => state.hasPermission(Resource.EDI, Operation.Update));
  const { data: preview } = useQuery({
    ...queries.edi.mappingPreview(transfer?.id ?? ""),
    enabled: !!transfer && direction === "inbound" && isTransferActionable(transfer.status),
  });
  const approveMutation = useMutation({
    mutationFn: () =>
      apiService.ediService.approveTransfer(transfer!.id, { mappings: Object.values(inlineMappings) }),
    onSuccess: async () => {
      toast.success("EDI transfer approval started.");
      await invalidateTransfers(queryClient);
      onOpenChange(false);
    },
    onError: () => toast.error("Failed to approve transfer"),
  });
  const rejectMutation = useMutation({
    mutationFn: () => apiService.ediService.rejectTransfer(transfer!.id, { reason: rejectReason }),
    onSuccess: async () => {
      toast.success("EDI transfer rejected");
      await invalidateTransfers(queryClient);
      onOpenChange(false);
    },
    onError: () => toast.error("Failed to reject transfer"),
  });
  const cancelMutation = useMutation({
    mutationFn: () => apiService.ediService.cancelTransfer(transfer!.id),
    onSuccess: async () => {
      toast.success("EDI transfer canceled");
      await invalidateTransfers(queryClient);
      onOpenChange(false);
    },
    onError: () => toast.error("Failed to cancel transfer"),
  });
  const unresolved = preview?.unresolved ?? [];
  const approvalReady = unresolved.every((row) => inlineMappings[mappingKey(row.entityType, row.sourceId)]?.targetId);
  const isActionable = transfer?.status ? isTransferActionable(transfer.status) : false;

  return (
    <Sheet open={!!transfer} onOpenChange={onOpenChange}>
      <SheetContent className="sm:max-w-3xl">
        <SheetHeader>
          <SheetTitle>EDI Transfer Details</SheetTitle>
          <SheetDescription>{transfer?.sourceShipmentId ?? ""}</SheetDescription>
        </SheetHeader>
        {transfer && (
          <div className="flex min-h-0 flex-col gap-4 overflow-auto px-4">
            <div className="grid gap-2 md:grid-cols-3">
              <InfoTile label="Status" value={<TransferStatusBadge status={transfer.status} />} />
              <InfoTile label="Submitted" value={formatUnix(transfer.submittedAt)} />
              <InfoTile label="Target Shipment" value={transfer.targetShipmentId ? <Link to={`/shipment-management/shipments?item=${transfer.targetShipmentId}`}>{transfer.targetShipmentId}</Link> : "—"} />
            </div>
            <div className="rounded-md border p-3">
              <div className="mb-2 font-medium">Tender Summary</div>
              <div className="grid gap-2 text-sm md:grid-cols-3">
                <span>BOL: {transfer.tenderPayload.bol || "—"}</span>
                <span>Pieces: {transfer.tenderPayload.pieces ?? "—"}</span>
                <span>Weight: {transfer.tenderPayload.weight ?? "—"}</span>
                <span>Stops: {transfer.tenderPayload.moves.length}</span>
                <span>Commodities: {transfer.tenderPayload.commodities.length}</span>
                <span>Accessorials: {transfer.tenderPayload.additionalCharges.length}</span>
              </div>
            </div>
            {(transfer.rejectionReason || transfer.failureReason) && (
              <div className="rounded-md border border-destructive/30 p-3 text-sm">
                {transfer.rejectionReason || transfer.failureReason}
              </div>
            )}
            {direction === "inbound" && isActionable && (
              <div className="flex flex-col gap-3 rounded-md border p-3">
                <div className="font-medium">Mapping Preview</div>
                {unresolved.length === 0 ? (
                  <div className="text-sm text-muted-foreground">No mapping required.</div>
                ) : (
                  unresolved.map((row) => (
                    <div key={mappingKey(row.entityType, row.sourceId)} className="grid gap-2 md:grid-cols-[1fr_1fr]">
                      <div className="text-sm">
                        <div className="font-medium">{row.entityType}</div>
                        <div className="text-muted-foreground">{row.sourceLabel || row.sourceId}</div>
                      </div>
                      <TargetLookup
                        entityType={row.entityType}
                        value={inlineMappings[mappingKey(row.entityType, row.sourceId)]?.targetId ?? ""}
                        onChange={(option) => {
                          const key = mappingKey(row.entityType, row.sourceId);
                          setInlineMappings((current) => ({
                            ...current,
                            [key]: {
                              entityType: row.entityType,
                              sourceId: row.sourceId,
                              sourceLabel: row.sourceLabel ?? "",
                              targetId: option?.id ?? "",
                              targetLabel: getOptionLabel(option),
                            },
                          }));
                        }}
                      />
                    </div>
                  ))
                )}
                <Input placeholder="Rejection reason" value={rejectReason} onChange={(event) => setRejectReason(event.target.value)} />
              </div>
            )}
          </div>
        )}
        <SheetFooter>
          {transfer && canUpdate && direction === "inbound" && isActionable && (
            <div className="flex gap-2">
              <Button variant="outline" disabled={!rejectReason.trim()} isLoading={rejectMutation.isPending} onClick={() => rejectMutation.mutate()}>
                <XIcon data-icon="inline-start" />
                Reject
              </Button>
              <Button disabled={!approvalReady} isLoading={approveMutation.isPending} onClick={() => approveMutation.mutate()}>
                <CheckIcon data-icon="inline-start" />
                Approve
              </Button>
            </div>
          )}
          {transfer && canUpdate && direction === "outbound" && isActionable && (
            <Button variant="outline" isLoading={cancelMutation.isPending} onClick={() => cancelMutation.mutate()}>
              Cancel Transfer
            </Button>
          )}
        </SheetFooter>
      </SheetContent>
    </Sheet>
  );
}

function TargetLookup({
  entityType,
  value,
  onChange,
}: {
  entityType: EDIMappingEntityType;
  value: string;
  onChange: (option: SelectOption | null) => void;
}) {
  const { data } = useQuery({
    queryKey: ["edi", "target-options", entityType],
    queryFn: async () =>
      api.get<GenericLimitOffsetResponse<SelectOption>>(`${mappingTargetEndpoints[entityType]}?limit=100`),
  });
  const selected = (data?.results ?? []).find((option) => option.id === value) ?? null;

  return (
    <select
      className="h-9 rounded-md border bg-background px-3 text-sm"
      value={value}
      onChange={(event) =>
        onChange((data?.results ?? []).find((option) => option.id === event.target.value) ?? null)
      }
    >
      <option value="">Select target</option>
      {selected && !data?.results.some((option) => option.id === selected.id) && (
        <option value={selected.id}>{getOptionLabel(selected)}</option>
      )}
      {(data?.results ?? []).map((option) => (
        <option key={option.id} value={option.id}>{getOptionLabel(option)}</option>
      ))}
    </select>
  );
}

function InfoTile({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="rounded-md border p-3">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="mt-1 text-sm font-medium">{value}</div>
    </div>
  );
}

function TransferStatusBadge({ status }: { status: string }) {
  const final = ["Approved", "Rejected", "Canceled", "Failed"].includes(status);
  return <Badge variant={final ? "outline" : "secondary"}>{status}</Badge>;
}

function isTransferActionable(status: string) {
  return !["Approved", "Rejected", "Canceled", "Failed", "Processing"].includes(status);
}

function mappingKey(entityType: string, sourceId: string) {
  return `${entityType}:${sourceId}`;
}

function getOptionLabel(option: SelectOption | null | undefined) {
  if (!option) return "";
  return option.name || option.code || option.description || option.id;
}

function formatUnix(value: number | null | undefined) {
  if (!value) return "—";
  return new Date(value * 1000).toLocaleString();
}

async function invalidateTransfers(queryClient: ReturnType<typeof useQueryClient>) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: queries.edi.inboundTransfers._def }),
    queryClient.invalidateQueries({ queryKey: queries.edi.outboundTransfers._def }),
  ]);
}
