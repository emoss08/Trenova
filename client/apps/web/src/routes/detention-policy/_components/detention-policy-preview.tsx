import { CalculationReceipt } from "@/routes/detention-desk/_components/calculation-receipt";
import { apiService } from "@/services/api";
import { Badge } from "@trenova/shared/components/ui/badge";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@trenova/shared/components/ui/card";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatDetentionMinutes } from "@trenova/shared/lib/detention";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type {
  DetentionPolicy,
  PreviewScenario,
} from "@trenova/shared/types/detention";
import { useQuery } from "@tanstack/react-query";
import { useMemo } from "react";
import { useWatch, type Control } from "react-hook-form";

/**
 * Worked scenarios the preview runs on every change. They are chosen to expose
 * the terms operators most often misconfigure: a stop that lands just inside
 * free time, one that runs long enough to reach a second rate tier, and a late
 * arrival that exercises the entitlement rule.
 */
const SCENARIO_BASE = 1_767_225_600;

type ScenarioDefinition = {
  key: string;
  label: string;
  description: string;
  scenario: PreviewScenario;
};

const SCENARIOS: ScenarioDefinition[] = [
  {
    key: "just-over",
    label: "Just over free time",
    description: "On-time arrival, departs 2h10m later",
    scenario: {
      arrivedAt: SCENARIO_BASE,
      departedAt: SCENARIO_BASE + 130 * 60,
      appointmentStart: SCENARIO_BASE,
      appointmentEnd: SCENARIO_BASE + 60 * 60,
      stopType: "Delivery",
      scheduleType: "Appointment",
      driverPayRate: "25.00",
    },
  },
  {
    key: "long-hold",
    label: "Long hold",
    description: "On-time arrival, departs 7h later",
    scenario: {
      arrivedAt: SCENARIO_BASE,
      departedAt: SCENARIO_BASE + 7 * 3600,
      appointmentStart: SCENARIO_BASE,
      appointmentEnd: SCENARIO_BASE + 60 * 60,
      stopType: "Delivery",
      scheduleType: "Appointment",
      driverPayRate: "25.00",
    },
  },
  {
    key: "late-arrival",
    label: "Late arrival",
    description: "Arrives 90m after the window, departs 5h later",
    scenario: {
      arrivedAt: SCENARIO_BASE + 150 * 60,
      departedAt: SCENARIO_BASE + 150 * 60 + 5 * 3600,
      appointmentStart: SCENARIO_BASE,
      appointmentEnd: SCENARIO_BASE + 60 * 60,
      stopType: "Delivery",
      scheduleType: "Appointment",
      driverPayRate: "25.00",
    },
  },
];

type PreviewRowProps = {
  definition: ScenarioDefinition;
  policy: DetentionPolicy;
  enabled: boolean;
  expanded: boolean;
};

function PreviewRow({ definition, policy, enabled, expanded }: PreviewRowProps) {
  const { data, isFetching, isError, error } = useQuery({
    queryKey: ["detentionPreview", definition.key, definition.scenario, policy],
    queryFn: async () =>
      apiService.detentionPolicyService.preview(policy, definition.scenario),
    enabled,
    retry: false,
    staleTime: 0,
  });

  if (!enabled) {
    return null;
  }

  if (isFetching && !data) {
    return <Skeleton className="h-20 w-full" />;
  }

  if (isError) {
    return (
      <div className="rounded-md border border-dashed p-3">
        <p className="text-sm font-medium">{definition.label}</p>
        <p className="text-muted-foreground text-xs">
          {error instanceof Error
            ? error.message
            : "Resolve the validation errors above to see this scenario."}
        </p>
      </div>
    );
  }

  if (!data) {
    return null;
  }

  const suppressed = data.suppressedByGate;

  return (
    <div className="rounded-md border p-3">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="text-sm font-medium">{definition.label}</p>
          <p className="text-muted-foreground text-xs">{definition.description}</p>
        </div>
        <div className="shrink-0 text-right">
          <p
            className={cn(
              "text-lg font-semibold tabular-nums",
              suppressed && "text-muted-foreground line-through",
            )}
          >
            {formatCurrency(data.billableAmount, data.policySnapshot.currency)}
          </p>
          <p className="text-muted-foreground text-xs tabular-nums">
            {formatDetentionMinutes(data.roundedMinutes)} billable of{" "}
            {formatDetentionMinutes(data.rawDwellMinutes)}
          </p>
        </div>
      </div>

      <div className="mt-2 flex flex-wrap gap-1.5">
        {data.arrivedLate && (
          <Badge variant="outline" className="text-[10px]">
            Arrived late
          </Badge>
        )}
        {data.capApplied !== "None" && (
          <Badge variant="outline" className="text-[10px]">
            {data.capApplied}
          </Badge>
        )}
        {suppressed && (
          <Badge className="border-none bg-red-500/15 text-[10px] text-red-700 dark:text-red-400">
            Suppressed: notice missed
          </Badge>
        )}
        {data.netMargin < 0 && (
          <Badge className="border-none bg-amber-500/15 text-[10px] text-amber-700 dark:text-amber-400">
            Negative margin {formatCurrency(data.netMargin)}
          </Badge>
        )}
        {data.billableAmount === 0 && !suppressed && (
          <Badge variant="outline" className="text-[10px]">
            Nothing billable
          </Badge>
        )}
      </div>

      {expanded && (
        <div className="mt-3 border-t pt-3">
          <CalculationReceipt
            trace={data.calculationTrace}
            currency={data.policySnapshot.currency}
          />
        </div>
      )}
    </div>
  );
}

type DetentionPolicyPreviewProps = {
  control: Control<DetentionPolicy>;
  expanded?: boolean;
};

/**
 * Live preview panel. Every keystroke re-runs the server's production
 * calculator, so the operator sees the money their terms produce before the
 * policy ever touches a shipment.
 */
export function DetentionPolicyPreview({
  control,
  expanded = false,
}: DetentionPolicyPreviewProps) {
  const values = useWatch({ control });

  const policy = values as DetentionPolicy;
  const ready = useMemo(
    () => Boolean(policy?.accessorialChargeId && policy?.code && policy?.name),
    [policy?.accessorialChargeId, policy?.code, policy?.name],
  );

  return (
    <Card className="sticky top-4">
      <CardHeader>
        <CardTitle className="text-base">Live preview</CardTitle>
        <CardDescription>
          What these terms would charge, computed by the same engine that bills
          real shipments.
        </CardDescription>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        {!ready ? (
          <p className="text-muted-foreground text-sm">
            Choose an accessorial charge and name the policy to see worked
            examples.
          </p>
        ) : (
          SCENARIOS.map((definition) => (
            <PreviewRow
              key={definition.key}
              definition={definition}
              policy={policy}
              enabled={ready}
              expanded={expanded}
            />
          ))
        )}
      </CardContent>
    </Card>
  );
}
