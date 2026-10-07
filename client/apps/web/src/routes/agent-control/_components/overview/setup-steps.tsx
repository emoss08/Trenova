import { queries } from "@/lib/queries";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useQuery } from "@tanstack/react-query";
import { Mark } from "../kit/marks";

/** The presets the first step offers, in order; the editor offers every one. */
const SETUP_PRESETS = ["anthropic", "openai", "groq", "openrouter", "ollama", "vllm"];

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
  const presets = SETUP_PRESETS.flatMap((key) => {
    const preset = catalogQuery.data?.presets.find((candidate) => candidate.key === key);
    return preset ? [preset] : [];
  });

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
    <section className="setup">
      <ol className="steps">
        {steps.map((step, index) => (
          <li key={step.title} className={cn(step.current && "cur")}>
            <span className="st-n">{index + 1}</span>
            <div>
              <b>{step.title}</b>
              <p>{step.body}</p>
              {step.current && (
                <div className="pre">
                  {presets.map((preset) => (
                    <button
                      key={preset.key}
                      type="button"
                      className="pre-b"
                      onClick={() => onPickPreset(preset.key)}
                    >
                      <Mark provider={{ name: preset.label }} s={20} />
                      {preset.label.replace(/\s*\(self-hosted\)$/, "")}
                    </button>
                  ))}
                </div>
              )}
            </div>
          </li>
        ))}
      </ol>
    </section>
  );
}
