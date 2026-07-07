import { ControlledEDIPartnerAutocompleteField } from "@/components/autocomplete-fields";
import { DataTablePanelContainer } from "@/components/data-table/data-table-panel";
import { Autocomplete } from "@/components/fields/autocomplete/autocomplete";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { usePermissionStore } from "@/stores/permission-store";
import type { DataTablePanelProps } from "@/types/data-table";
import type { EDIMappingProfile, EDIMappingProfileItem } from "@/types/edi";
import { Operation, Resource } from "@/types/permission";
import type { ServiceFailureReasonCode } from "@/types/service-failure-reason-code";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckIcon, Trash2Icon } from "lucide-react";
import { useState } from "react";
import type { FieldValues } from "react-hook-form";
import { toast } from "sonner";
import { mappingEntityTypes } from "../edi-schemas";
import { TargetLookup } from "../edi-target-lookup";
import { EDIEmptyState } from "./edi-panel-primitives";

const emptyDraft: EDIMappingProfileItem = {
  entityType: "Customer",
  sourceId: "",
  sourceLabel: "",
  targetId: "",
  targetLabel: "",
};

export function MappingProfilePanel({
  partnerId,
  canUpdate,
}: {
  partnerId: string;
  canUpdate: boolean;
}) {
  const queryClient = useQueryClient();
  const { data } = useQuery(queries.edi.mappingProfile(partnerId));
  const [draft, setDraft] = useState<EDIMappingProfileItem>(emptyDraft);
  const saveMutation = useApiMutation({
    mutationFn: (item: EDIMappingProfileItem) =>
      data?.id
        ? apiService.ediService.saveMappingProfileItems(data.id, [item])
        : apiService.ediService.saveMappingProfile(partnerId, [item]),
    onSuccess: async () => {
      toast.success("Mapping saved");
      setDraft((current) => ({ ...emptyDraft, entityType: current.entityType }));
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.edi.mappingProfile(partnerId).queryKey,
        }),
        queryClient.invalidateQueries({ queryKey: ["edi-mapping-profile-list"] }),
      ]);
    },
    onError: () => toast.error("Failed to save mapping"),
  });
  const deleteMutation = useApiMutation({
    mutationFn: (itemId: string) =>
      data?.id
        ? apiService.ediService.deleteMappingProfileItem(data.id, itemId)
        : apiService.ediService.deleteMappingItem(partnerId, itemId),
    onSuccess: async () => {
      toast.success("Mapping deleted");
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.edi.mappingProfile(partnerId).queryKey,
        }),
        queryClient.invalidateQueries({ queryKey: ["edi-mapping-profile-list"] }),
      ]);
    },
    onError: () => toast.error("Failed to delete mapping"),
  });

  return (
    <Tabs defaultValue="Customer" className="gap-3">
      <TabsList
        className="w-full flex-wrap justify-start! border-b border-border px-1"
        variant="underline"
      >
        {mappingEntityTypes.map((entityType) => (
          <TabsTrigger key={entityType} value={entityType} className="max-w-30">
            {mappingEntityTabLabel(entityType)}
          </TabsTrigger>
        ))}
      </TabsList>
      {mappingEntityTypes.map((entityType) => {
        const entries = (data?.entries ?? []).filter((entry) => entry.entityType === entityType);
        return (
          <TabsContent
            key={entityType}
            value={entityType}
            className="flex flex-col gap-3 px-3 pb-3"
          >
            {canUpdate && (
              <div className="grid gap-2 md:grid-cols-5">
                <MappingSourceInput
                  entityType={entityType}
                  value={draft.entityType === entityType ? draft.sourceId : ""}
                  onChange={(source) =>
                    setDraft({
                      ...draft,
                      entityType,
                      sourceId: source.sourceId,
                      sourceLabel: source.sourceLabel,
                    })
                  }
                />
                <Input
                  placeholder="Source label"
                  value={draft.entityType === entityType ? (draft.sourceLabel ?? "") : ""}
                  onChange={(event) =>
                    setDraft({ ...draft, entityType, sourceLabel: event.target.value })
                  }
                />
                {entityType === "ServiceFailureReasonCode" ? (
                  <Input
                    placeholder="Partner X12 code"
                    value={draft.entityType === entityType ? draft.targetId : ""}
                    onChange={(event) =>
                      setDraft({
                        ...draft,
                        entityType,
                        targetId: event.target.value.trim().toUpperCase(),
                      })
                    }
                  />
                ) : (
                  <TargetLookup
                    entityType={entityType}
                    value={draft.entityType === entityType ? draft.targetId : ""}
                    onChange={(target) =>
                      setDraft({
                        ...draft,
                        entityType,
                        targetId: target.targetId,
                        targetLabel: target.targetLabel,
                      })
                    }
                  />
                )}
                <Input
                  placeholder="Target label"
                  value={draft.entityType === entityType ? (draft.targetLabel ?? "") : ""}
                  onChange={(event) =>
                    setDraft({ ...draft, entityType, targetLabel: event.target.value })
                  }
                />
                <Button
                  disabled={!draft.sourceId || !draft.targetId || draft.entityType !== entityType}
                  isLoading={saveMutation.isPending}
                  onClick={() => saveMutation.mutate(draft)}
                >
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
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            onClick={() => deleteMutation.mutate(entry.id!)}
                          >
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

function MappingSourceInput({
  entityType,
  value,
  onChange,
}: {
  entityType: EDIMappingProfileItem["entityType"];
  value: string;
  onChange: (source: { sourceId: string; sourceLabel: string }) => void;
}) {
  if (entityType === "ServiceFailureReasonCode") {
    return (
      <Autocomplete<ServiceFailureReasonCode, FieldValues>
        link="/service-failure-reason-codes/select-options/"
        selectedValueLink="/service-failure-reason-codes/"
        value={value}
        placeholder="Service failure reason"
        clearable
        onChange={(nextValue) => {
          if (!nextValue) {
            onChange({ sourceId: "", sourceLabel: "" });
          }
        }}
        onOptionChange={(option) =>
          onChange({
            sourceId: option?.id ?? "",
            sourceLabel: option ? `${option.code} - ${option.label}` : "",
          })
        }
        getOptionValue={(option) => option.id || ""}
        getDisplayValue={(option) => option.code || option.label || ""}
        renderOption={(option) => (
          <div className="flex size-full flex-col items-start">
            <span className="w-full truncate font-medium">{option.code}</span>
            <span className="w-full truncate text-2xs text-muted-foreground">{option.label}</span>
          </div>
        )}
      />
    );
  }

  return (
    <Input
      placeholder="Source value key"
      value={value}
      onChange={(event) =>
        onChange({
          sourceId: event.target.value,
          sourceLabel: "",
        })
      }
    />
  );
}

function mappingEntityTabLabel(entityType: EDIMappingProfileItem["entityType"]) {
  if (entityType === "ServiceFailureReasonCode") return "Failure Reason";
  return entityType;
}

export function MappingProfileTablePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<EDIMappingProfile>) {
  const canUpdate = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Update),
  );
  const [selectedPartnerId, setSelectedPartnerId] = useState("");
  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen) setSelectedPartnerId("");
    onOpenChange(nextOpen);
  };

  if (mode === "edit") {
    if (!row) return null;
    return (
      <DataTablePanelContainer
        open={open}
        onOpenChange={onOpenChange}
        title={row.name}
        description={
          row.partner
            ? `Source value mappings for ${row.partner.code} — ${row.partner.name}`
            : "Source value mappings for this trading partner"
        }
        size="xl"
      >
        <MappingProfilePanel partnerId={row.ediPartnerId} canUpdate={canUpdate} />
      </DataTablePanelContainer>
    );
  }

  return (
    <DataTablePanelContainer
      open={open}
      onOpenChange={handleOpenChange}
      title="New Mapping Profile"
      description="Choose which partner source values should map into local records."
      size="xl"
    >
      <div className="flex min-h-0 flex-col gap-4">
        <div className="max-w-md">
          <ControlledEDIPartnerAutocompleteField
            label="Partner"
            placeholder="Select partner"
            description="Saving the first mapping creates the partner's mapping profile."
            value={selectedPartnerId}
            onValueChange={setSelectedPartnerId}
          />
        </div>
        {selectedPartnerId ? (
          <MappingProfilePanel partnerId={selectedPartnerId} canUpdate={canUpdate} />
        ) : (
          <EDIEmptyState message="Select a partner to manage mapping records." />
        )}
      </div>
    </DataTablePanelContainer>
  );
}
