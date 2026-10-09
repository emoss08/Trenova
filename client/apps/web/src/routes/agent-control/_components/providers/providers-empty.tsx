import type { AIProviderPreset } from "@/types/ai-provider";
import { useT } from "@trenova/shared/i18n/use-t";
import { Ic } from "../kit/ic";
import { Mark } from "../kit/marks";
import { presetDisplayName, presetHint } from "./preset-options";

type ProvidersEmptyProps = {
  presets: readonly AIProviderPreset[];
  canCreate: boolean;
  onPick: (preset: AIProviderPreset) => void;
};

/** No provider yet: what one does, where to start, and what Trenova does until then. */
export function ProvidersEmpty({ presets, canCreate, onPick }: ProvidersEmptyProps) {
  const t = useT();
  const hints = {
    local: t("Your network · no key"),
    key: t("Hosted · API key"),
    none: t("Hosted · no key"),
  };

  return (
    <div className="tabp">
      <div className="ep">
        <span className="ep-k mono">{t("No providers yet")}</span>
        <h2>{t("Connect a model to wake your agents")}</h2>
        <p>
          {t(
            "Agents, the assistant and document reading all route through a provider. Start with a hosted model, or point Trenova at one on your own network.",
          )}
        </p>
        {canCreate && (
          <div className="ep-g">
            {presets.map((preset) => {
              const name = presetDisplayName(preset);
              return (
                <button
                  key={preset.key}
                  type="button"
                  className="ep-b"
                  onClick={() => onPick(preset)}
                >
                  <Mark provider={{ name }} preset={preset} s={30} />
                  <span>
                    <b>{name}</b>
                    <em>{hints[presetHint(preset)]}</em>
                  </span>
                  <Ic n="arrowR" s={13} />
                </button>
              );
            })}
          </div>
        )}
        <div className="ep-f">
          <b>{t("Until then")}</b>
          <span>{t("Search uses words only")}</span>
          <span>{t("Scope checks use built-in rules")}</span>
          <span>{t("Documents wait for manual entry")}</span>
        </div>
      </div>
    </div>
  );
}
