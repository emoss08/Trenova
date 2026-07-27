import { CalculationReceipt } from "@/routes/detention-desk/_components/calculation-receipt";
import { useDebounce } from "@/hooks/use-debounce";
import { apiService } from "@/services/api";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { formatDetentionMinutes } from "@trenova/shared/lib/detention";
import { cn, formatCurrency } from "@trenova/shared/lib/utils";
import type { DetentionPolicy, PreviewScenario } from "@trenova/shared/types/detention";
import { useQuery } from "@tanstack/react-query";
import { FlaskConicalIcon } from "lucide-react";
import { useFormContext, useWatch } from "react-hook-form";

/**
 * Worked scenarios the preview runs on every change. They are chosen to expose
 * the terms operators most often misconfigure: a stop that lands just inside
 * free time, one that runs long enough to reach a second rate tier, and a late
 * arrival that exercises the entitlement rule.
 */
const SCENARIO_BASE = 1_767_225_600;
const PREVIEW_DEBOUNCE_MS = 400;

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
};

function PreviewRow({ definition, policy, enabled }: PreviewRowProps) {
  const { data, isFetching, isError, error } = useQuery({
    queryKey: ["detention", "preview", definition.key, definition.scenario, policy],
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
    return <Skeleton className="h-24 w-full" />;
  }

  if (isError) {
    return (
      <div className="rounded-md border border-dashed p-3">
        <p className="text-sm font-medium">{definition.label}</p>
        <p className="text-muted-foreground text-xs">
          {error instanceof Error
            ? error.message
            : "Resolve the validation errors on the Terms tab to see this scenario."}
        </p>
      </div>
    );
  }

  if (!data) {
    return null;
  }

  const suppressed = data.suppressedByGate;

  return (
    <div className="rounded-lg border p-3 transition-colors">
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

      <div className="mt-3 border-t pt-3">
        <CalculationReceipt
          trace={data.calculationTrace}
          currency={data.policySnapshot.currency}
        />
      </div>
    </div>
  );
}

/**
 * Live preview panel. Edits on the Terms tab re-run the server's production
 * calculator (debounced), so the operator sees the money their terms produce
 * before the policy ever touches a shipment.
 */
export function DetentionPolicyPreview() {
  const { control } = useFormContext<DetentionPolicy>();
  const values = useWatch({ control });

  const policy = useDebounce(values, PREVIEW_DEBOUNCE_MS) as DetentionPolicy;
  const ready = Boolean(policy?.accessorialChargeId && policy?.code && policy?.name);

  return (
    <div className="flex flex-col gap-3">
      <div>
        <h3 className="text-sm font-medium">Worked examples</h3>
        <p className="text-muted-foreground text-xs">
          What these terms would charge, computed by the same engine that bills real
          shipments.
        </p>
      </div>

      {!ready ? (
        <div className="flex flex-col items-center gap-2 rounded-lg border border-dashed py-10 text-center">
          <FlaskConicalIcon className="text-muted-foreground size-5" />
          <p className="text-muted-foreground max-w-xs text-sm">
            Name the policy and choose an accessorial charge on the Terms tab to see worked
            examples.
          </p>
        </div>
      ) : (
        SCENARIOS.map((definition) => (
          <PreviewRow
            key={definition.key}
            definition={definition}
            policy={policy}
            enabled={ready}
          />
        ))
      )}
    </div>
  );
}
