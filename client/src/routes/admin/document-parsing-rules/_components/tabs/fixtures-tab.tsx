import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import { Operation, Resource } from "@/types/permission";
import { usePermission } from "@/hooks/use-permission";
import type { Fixture } from "@/types/document-parsing-rule";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { FlaskConicalIcon, PlusIcon } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import { FixtureDetail } from "../fixture-detail/fixture-detail";

const REVIEW_STATUS_VARIANT = {
  Ready: "active",
  NeedsReview: "warning",
  Unavailable: "inactive",
} as const;

const NEW_FIXTURE_TEMPLATE = {
  name: "New Fixture",
  textSnapshot: "Paste document text here",
  assertions: {
    expectedFields: {},
    fieldAssertions: {
      referenceNumber: [
        {
          operator: "not_empty" as const,
          value: "",
          values: [],
          pattern: "",
        },
      ],
    },
    requiredStopRoles: [],
    minimumStopCount: 0,
    reviewStatus: "NeedsReview" as const,
  },
};

export default function FixturesTab({ ruleSetId }: { ruleSetId: string }) {
  const [selectedFixtureId, setSelectedFixtureId] = useState<string | null>(
    null,
  );

  if (selectedFixtureId) {
    return (
      <FixtureDetail
        fixtureId={selectedFixtureId}
        ruleSetId={ruleSetId}
        onBack={() => setSelectedFixtureId(null)}
        onDeleted={() => setSelectedFixtureId(null)}
      />
    );
  }

  return (
    <FixtureList
      ruleSetId={ruleSetId}
      onSelectFixture={setSelectedFixtureId}
    />
  );
}

function FixtureList({
  ruleSetId,
  onSelectFixture,
}: {
  ruleSetId: string;
  onSelectFixture: (id: string) => void;
}) {
  const queryClient = useQueryClient();
  const { allowed: canCreate } = usePermission(
    Resource.DocumentParsingRule,
    Operation.Create,
  );

  const { data: fixtures, isLoading } = useQuery({
    ...queries.documentParsingRule.fixtures(ruleSetId),
  });

  const createMutation = useMutation({
    mutationFn: () =>
      apiService.documentParsingRuleService.createFixture(
        ruleSetId,
        NEW_FIXTURE_TEMPLATE,
      ),
    onSuccess: (data) => {
      void queryClient.invalidateQueries({
        queryKey: queries.documentParsingRule.fixtures._def,
      });
      toast.success("Fixture created");
      if (data.id) onSelectFixture(data.id);
    },
    onError: (error) => {
      toast.error(
        error instanceof Error ? error.message : "Failed to create fixture",
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
          {fixtures?.length ?? 0} fixture{(fixtures?.length ?? 0) !== 1 ? "s" : ""}
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
            {createMutation.isPending ? "Creating..." : "New Fixture"}
          </Button>
        )}
      </div>

      {(!fixtures || fixtures.length === 0) && (
        <div className="flex flex-col items-center gap-3 rounded-lg border border-dashed py-12 text-center">
          <div className="flex size-10 items-center justify-center rounded-full bg-muted">
            <FlaskConicalIcon className="size-5 text-muted-foreground" />
          </div>
          <div className="space-y-1">
            <p className="text-sm font-medium">No fixtures yet</p>
            <p className="max-w-xs text-xs text-muted-foreground">
              Fixtures are sample documents with expected extraction results. They
              let you validate that rules produce the correct fields and stops
              before publishing.
            </p>
          </div>
        </div>
      )}

      <div className="space-y-1">
        {fixtures?.map((f: Fixture) => {
          const assertionCount = countAssertions(f);

          return (
            <button
              key={f.id}
              type="button"
              onClick={() => onSelectFixture(f.id!)}
              className="flex w-full items-center justify-between rounded-md border p-3 text-left transition-colors hover:bg-muted/50"
            >
              <div className="space-y-1">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium">{f.name}</span>
                  {f.fileName && (
                    <span className="text-xs text-muted-foreground">
                      {f.fileName}
                    </span>
                  )}
                </div>
                {assertionCount > 0 && (
                  <p className="text-xs text-muted-foreground">
                    {assertionCount} assertion{assertionCount !== 1 ? "s" : ""}
                  </p>
                )}
              </div>
              <div className="flex items-center gap-2">
                {f.assertions?.reviewStatus && (
                  <Badge
                    variant={
                      REVIEW_STATUS_VARIANT[
                        f.assertions
                          .reviewStatus as keyof typeof REVIEW_STATUS_VARIANT
                      ] ?? "secondary"
                    }
                  >
                    {f.assertions.reviewStatus}
                  </Badge>
                )}
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}

function countAssertions(f: Fixture): number {
  if (!f.assertions) return 0;
  let count = Object.keys(f.assertions.expectedFields ?? {}).length;
  count += Object.values(f.assertions.fieldAssertions ?? {}).reduce(
    (total, assertions) => total + (assertions?.length ?? 0),
    0,
  );
  count += (f.assertions.requiredStopRoles ?? []).length;
  if (f.assertions.minimumStopCount && f.assertions.minimumStopCount > 0) count++;
  return count;
}
