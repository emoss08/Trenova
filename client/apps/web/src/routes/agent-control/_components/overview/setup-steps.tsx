import { queries } from "@/lib/queries";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { ProviderMark } from "../providers/provider-mark";

/** How many presets the first step offers; the editor offers every one. */
const PRESETS_SHOWN = 6;

type SetupStepsProps = {
  agentCount: number;
  onPickPreset: (preset: string | null) => void;
};

/**
 * With no provider, the overview is the way in: connect one, choose what it handles, and
 * the agents already set up start on their own.
 */
export function SetupSteps({ agentCount, onPickPreset }: SetupStepsProps) {
  const t = useT();
  const catalogQuery = useQuery(queries.aiProvider.catalog());
  const presets = (catalogQuery.data?.presets ?? []).slice(0, PRESETS_SHOWN);

  const steps = [
    {
      title: t("Connect a model provider"),
      body: t("A hosted model with an API key, or one running on your own network."),
      current: true,
    },
    {
      title: t("Choose what it handles"),
      body: t("Each task goes to the first provider in order that takes it."),
      current: false,
    },
    {
      title: t("Agents start on their own"),
      body:
        agentCount === 1
          ? t("Your agent is set up already. Nothing else to switch on.")
          : t("All {0} are set up already. Nothing else to switch on.", agentCount),
      current: false,
    },
  ];

  return (
    <ol className="flex flex-col gap-4">
      {steps.map((step, index) => (
        <li key={step.title} className="flex gap-3">
          <span
            className={cn(
              "flex size-6 shrink-0 items-center justify-center rounded-full border text-xs font-medium tabular-nums",
              step.current
                ? "border-foreground bg-foreground text-background"
                : "border-border text-muted-foreground",
            )}
          >
            {index + 1}
          </span>
          <div className="flex min-w-0 flex-col gap-1">
            <b className={cn("text-sm font-semibold", !step.current && "text-muted-foreground")}>
              {step.title}
            </b>
            <p className="text-xs text-muted-foreground">{step.body}</p>
            {step.current && (
              <div className="mt-1.5 flex flex-wrap gap-1.5">
                {presets.map((preset) => (
                  <Button key={preset.key} size="sm" variant="outline" onClick={() => onPickPreset(preset.key)}>
                    <ProviderMark
                      provider={{ name: preset.label, kind: preset.kind, baseUrl: preset.baseUrl }}
                      presets={catalogQuery.data?.presets ?? []}
                      size={16}
                    />
                    {preset.label}
                  </Button>
                ))}
                <Button size="sm" variant="ghost" onClick={() => onPickPreset(null)}>
                  {t("Something else")}
                </Button>
              </div>
            )}
          </div>
        </li>
      ))}
    </ol>
  );
}
