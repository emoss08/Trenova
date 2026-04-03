import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { Operation, Resource } from "@/types/permission";
import { usePermission } from "@/hooks/use-permission";
import type { RuleVersion } from "@/types/document-parsing-rule";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  GitBranchIcon,
  LayersIcon,
  MapPinIcon,
  PlusIcon,
  TextIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { toast } from "sonner";
import { VersionDetail } from "../version-detail/version-detail";

const STATUS_BADGE_VARIANT = {
  Draft: "warning",
  Published: "active",
  Archived: "secondary",
} as const;

const PARSER_MODE_LABELS: Record<string, string> = {
  merge_with_base: "Merge",
  override_base: "Override",
};

const NEW_VERSION_TEMPLATE = {
  status: "Draft" as const,
  parserMode: "merge_with_base" as const,
  matchConfig: {
    providerFingerprints: [],
    fileNameContains: [],
    requiresAll: [],
    requiresAny: [],
    sectionAnchors: [],
  },
  ruleDocument: {
    sections: [],
    fields: [
      {
        key: "referenceNumber",
        label: "Reference Number",
        sectionNames: [],
        aliases: ["reference number"],
        patterns: [],
        normalizer: "",
        required: false,
        confidence: 0.8,
      },
    ],
    stops: [],
  },
};

export default function VersionsTab({ ruleSetId }: { ruleSetId: string }) {
  const [selectedVersionId, setSelectedVersionId] = useState<string | null>(
    null,
  );

  if (selectedVersionId) {
    return (
      <VersionDetail
        versionId={selectedVersionId}
        onBack={() => setSelectedVersionId(null)}
      />
    );
  }

  return (
    <VersionList
      ruleSetId={ruleSetId}
      onSelectVersion={setSelectedVersionId}
    />
  );
}

function VersionList({
  ruleSetId,
  onSelectVersion,
}: {
  ruleSetId: string;
  onSelectVersion: (id: string) => void;
}) {
  const queryClient = useQueryClient();
  const { allowed: canCreate } = usePermission(
    Resource.DocumentParsingRule,
    Operation.Create,
  );

  const { data: versions, isLoading } = useQuery({
    ...queries.documentParsingRule.versions(ruleSetId),
  });

  const sortedVersions = useMemo(
    () =>
      [...(versions ?? [])].sort(
        (a, b) => (b.versionNumber ?? 0) - (a.versionNumber ?? 0),
      ),
    [versions],
  );

  const createMutation = useMutation({
    mutationFn: () =>
      apiService.documentParsingRuleService.createVersion(
        ruleSetId,
        NEW_VERSION_TEMPLATE,
      ),
    onSuccess: (data) => {
      void queryClient.invalidateQueries({
        queryKey: queries.documentParsingRule.versions._def,
      });
      toast.success("Draft version created (auto-numbered by server)");
      if (data.id) onSelectVersion(data.id);
    },
    onError: (error) => {
      toast.error(
        error instanceof Error ? error.message : "Failed to create draft version",
      );
    },
  });

  if (isLoading) {
    return (
      <div className="space-y-2">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-16 w-full" />
        ))}
      </div>
    );
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-muted-foreground">
          {sortedVersions.length} version{sortedVersions.length !== 1 ? "s" : ""}
        </h3>
        {canCreate && (
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="gap-1"
            onClick={() => createMutation.mutate()}
            disabled={createMutation.isPending}
          >
            <PlusIcon className="size-3.5" />
            {createMutation.isPending ? "Creating..." : "New Draft"}
          </Button>
        )}
      </div>

      {sortedVersions.length === 0 && (
        <div className="flex flex-col items-center gap-3 rounded-lg border border-dashed py-12 text-center">
          <div className="flex size-10 items-center justify-center rounded-full bg-muted">
            <GitBranchIcon className="size-5 text-muted-foreground" />
          </div>
          <div className="space-y-1">
            <p className="text-sm font-medium">No versions yet</p>
            <p className="max-w-xs text-xs text-muted-foreground">
              Versions define how documents are parsed. Each version contains match
              criteria, field extraction rules, and stop definitions. Create a draft to
              get started.
            </p>
          </div>
        </div>
      )}

      <div className="space-y-1">
        {sortedVersions.map((v: RuleVersion) => {
          const fieldCount = v.ruleDocument?.fields?.length ?? 0;
          const stopCount = v.ruleDocument?.stops?.length ?? 0;
          const parserLabel = PARSER_MODE_LABELS[v.parserMode] ?? v.parserMode;

          return (
            <button
              key={v.id}
              type="button"
              onClick={() => onSelectVersion(v.id!)}
              className="flex w-full items-center justify-between rounded-md border p-3 text-left transition-colors hover:bg-muted/50"
            >
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium">
                    Version {v.versionNumber}
                  </span>
                  {v.label && (
                    <span className="text-xs text-muted-foreground">
                      {v.label}
                    </span>
                  )}
                </div>
                <div className="flex items-center gap-3 text-xs text-muted-foreground">
                  <span className="flex items-center gap-1">
                    <LayersIcon className="size-3" />
                    {parserLabel}
                  </span>
                  <span className="flex items-center gap-1">
                    <TextIcon className="size-3" />
                    {fieldCount} field{fieldCount !== 1 ? "s" : ""}
                  </span>
                  <span className="flex items-center gap-1">
                    <MapPinIcon className="size-3" />
                    {stopCount} stop{stopCount !== 1 ? "s" : ""}
                  </span>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <Badge
                  variant={
                    STATUS_BADGE_VARIANT[
                      v.status as keyof typeof STATUS_BADGE_VARIANT
                    ] ?? "secondary"
                  }
                >
                  {v.status}
                </Badge>
                {v.createdAt && (
                  <span className="text-xs text-muted-foreground">
                    {new Date(v.createdAt * 1000).toLocaleDateString()}
                  </span>
                )}
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}
