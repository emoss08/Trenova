import type { AIProviderRow } from "@/lib/graphql/ai-provider";
import type { AIProviderPlatform } from "@/types/ai-provider";
import { Button } from "@trenova/shared/components/ui/button";
import { NovaBadge } from "@trenova/shared/components/ui/nova-badge";
import { useT } from "@trenova/shared/i18n/use-t";
import { Callout } from "../edit/callout";
import { SecH } from "../kit/layout";
import { Mark } from "../kit/marks";
import { canRunNova } from "./provider-model";

type NovaProviderProps = {
  providers: readonly AIProviderRow[];
  /** The model Nova runs on, from the catalog. */
  model: string;
  novaPlatforms: readonly AIProviderPlatform[];
  canChoose: boolean;
  canAdd: boolean;
  choosing: boolean;
  onChoose: (provider: AIProviderRow) => void;
  onAdd: () => void;
};

/**
 * Which provider Nova runs on. Nova always runs on its own model; an organization chooses
 * only whose key, limits and cap it uses, from its Anthropic providers.
 */
export function NovaProvider({
  providers,
  model,
  novaPlatforms,
  canChoose,
  canAdd,
  choosing,
  onChoose,
  onAdd,
}: NovaProviderProps) {
  const t = useT();
  const current = providers.find((provider) => provider.nova) ?? null;
  const eligible = providers.filter((provider) => canRunNova(provider, novaPlatforms));

  return (
    <section className="sec" id="nova" aria-label={t("Nova")}>
      <SecH
        t={t("Nova")}
        n={current ? t("Runs on {0} through {1}", model, current.name) : t("Not set up")}
      />
      <p className="lead">
        {t(
          "Nova reads paperwork, reviews the billing queue, sorts mail and writes briefings. It always runs on {0}, with the key, limits and cap of the provider chosen here.",
          model,
        )}
      </p>
      {current && !current.enabled && (
        <Callout tone="w">
          {t("{0} runs Nova but is off, so Nova is paused until it is turned on.", current.name)}
        </Callout>
      )}
      {!current && eligible.length === 0 && (
        <Callout
          tone="w"
          action={
            canAdd ? (
              <Button type="button" size="sm" variant="outline" onClick={onAdd}>
                {t("Add an Anthropic provider")}
              </Button>
            ) : undefined
          }
        >
          {t("Nova needs an Anthropic provider. Add one and Nova runs on its key.")}
        </Callout>
      )}
      {!current && eligible.length > 0 && (
        <Callout tone="w">{t("Nova is off until a provider is chosen for it.")}</Callout>
      )}
      {eligible.length > 0 && (
        <div
          className="flex flex-wrap items-center gap-2"
          role="group"
          aria-label={t("Nova's provider")}
        >
          {eligible.map((provider) => (
            <Button
              key={provider.id}
              type="button"
              size="sm"
              variant={provider.nova ? "default" : "outline"}
              aria-pressed={provider.nova}
              disabled={!canChoose || choosing || provider.nova}
              onClick={() => onChoose(provider)}
            >
              {provider.nova ? <NovaBadge size="xs" /> : <Mark provider={provider} s={14} />}
              {provider.name}
            </Button>
          ))}
        </div>
      )}
    </section>
  );
}
