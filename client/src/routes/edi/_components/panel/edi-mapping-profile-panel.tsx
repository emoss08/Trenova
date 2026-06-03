import {
  EDIPartnerAutocompleteField,
  ServiceFailureReasonCodeAutocompleteField,
} from "@/components/autocomplete-fields";
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
import type { EDIMappingProfileItem, EDIPartner } from "@/types/edi";
import { Operation, Resource } from "@/types/permission";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { CheckIcon, Trash2Icon } from "lucide-react";
import { useEffect, useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { mappingEntityTypes } from "../edi-schemas";
import { TargetLookup } from "../edi-target-lookup";
import { EDIEmptyState } from "./edi-empty-state";

export function MappingProfilePanel({
  partnerId,
  canUpdate,
}: {
  partnerId: string;
  canUpdate: boolean;
}) {
  const queryClient = useQueryClient();
  const { data } = useQuery(queries.edi.mappingProfile(partnerId));
  const [draft, setDraft] = useState<EDIMappingProfileItem>({
    entityType: "Customer",
    sourceId: "",
    sourceLabel: "",
    targetId: "",
    targetLabel: "",
  });
  const saveMutation = useApiMutation({
    mutationFn: (item: EDIMappingProfileItem) =>
      data?.id
        ? apiService.ediService.saveMappingProfileItems(data.id, [item])
        : apiService.ediService.saveMappingProfile(partnerId, [item]),
    onSuccess: async () => {
      toast.success("Mapping saved");
      setDraft((current) => ({
        ...current,
        sourceId: "",
        sourceLabel: "",
        targetId: "",
        targetLabel: "",
      }));
      await queryClient.invalidateQueries({
        queryKey: queries.edi.mappingProfile(partnerId).queryKey,
      });
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
      await queryClient.invalidateQueries({
        queryKey: queries.edi.mappingProfile(partnerId).queryKey,
      });
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
  const { control, setValue } = useForm<{ sourceId: string }>({
    defaultValues: { sourceId: value },
  });

  useEffect(() => {
    setValue("sourceId", value);
  }, [setValue, value]);

  if (entityType === "ServiceFailureReasonCode") {
    return (
      <ServiceFailureReasonCodeAutocompleteField
        control={control}
        name="sourceId"
        placeholder="Service failure reason"
        clearable
        onOptionChange={(option) =>
          onChange({
            sourceId: option?.id ?? "",
            sourceLabel: option ? `${option.code} - ${option.label}` : "",
          })
        }
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

export function MappingProfilesWorkspace() {
  const canUpdate = usePermissionStore((state) =>
    state.hasPermission(Resource.EDI, Operation.Update),
  );
  const [selectedPartner, setSelectedPartner] = useState<EDIPartner | null>(null);
  const { control } = useForm<{ partnerId: string }>({
    defaultValues: { partnerId: "" },
  });
  const selectedPartnerId = useWatch({ control, name: "partnerId" });

  return (
    <div className="grid min-h-0 gap-4 p-3 lg:grid-cols-[18rem_1fr]">
      <div className="rounded-md border bg-background">
        <div className="border-b px-3 py-2">
          <div className="text-sm font-medium">Partner</div>
          <div className="text-xs text-muted-foreground">
            Choose which partner source values should map into local records.
          </div>
        </div>
        <div className="p-3">
          <EDIPartnerAutocompleteField
            control={control}
            name="partnerId"
            placeholder="Select partner"
            clearable
            onOptionChange={setSelectedPartner}
          />
          {selectedPartner && (
            <div className="mt-3 rounded-md border bg-muted/20 p-3 text-sm">
              <div className="font-medium">{selectedPartner.name}</div>
              <div className="text-xs text-muted-foreground">{selectedPartner.code}</div>
            </div>
          )}
        </div>
      </div>
      <div className="min-w-0 rounded-md border bg-background">
        {selectedPartnerId ? (
          <MappingProfilePanel partnerId={selectedPartnerId} canUpdate={canUpdate} />
        ) : (
          <EDIEmptyState message="Select a partner to manage mapping records." />
        )}
      </div>
    </div>
  );
}
