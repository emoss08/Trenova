import { Badge } from "@trenova/shared/components/ui/badge";
import type { DeskCaseOption } from "@/lib/graphql/desk-case";
import { AssistMark } from "@trenova/shared/components/ui/assist-mark";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { textTone } from "./case-tone";

/**
 * What could happen: the two or three ways this can go, one of them
 * recommended and one chosen. Each says what it leads to and what it costs.
 * Choosing another redrafts the plan below; once the plan is decided the
 * choice is closed.
 */
export function CaseOptions({
  options,
  disabled,
  pendingId,
  onChoose,
}: {
  options: DeskCaseOption[];
  disabled: boolean;
  pendingId: string | null;
  onChoose: (option: DeskCaseOption) => void;
}) {
  const t = useT();
  if (options.length === 0) {
    return null;
  }

  return (
    <section aria-labelledby="dk-cz-opt-h">
      <div className="dk-cz-sh">
        <h2 id="dk-cz-opt-h">{t("What could happen")}</h2>
        {!disabled && <span className="dk-cz-sh-more">{t("Press 1–{0}", options.length)}</span>}
      </div>
      <div className="dk-cz-opts" role="radiogroup" aria-labelledby="dk-cz-opt-h">
        {options.map((option) => {
          const checked = pendingId !== null ? pendingId === option.id : option.chosen;
          return (
            <button
              key={option.id}
              type="button"
              role="radio"
              aria-checked={checked}
              className="dk-cz-opt ui-focus-ring"
              disabled={disabled || pendingId !== null}
              onClick={() => onChoose(option)}
            >
              <span className="dk-cz-opt-rd" />
              <span className="dk-cz-opt-t">
                {option.label}
                {option.recommended && (
                  <Badge>
                    <AssistMark size={11} />
                    {t("Recommended")}
                  </Badge>
                )}
              </span>
              <span className="dk-cz-opt-f">
                {option.outcome !== "" && (
                  <b className={cn(textTone(option.tone))}>{option.outcome}</b>
                )}
                {option.outcomeNote !== "" && (
                  <span className={cn(textTone(option.tone))}>{option.outcomeNote}</span>
                )}
              </span>
              <span className="dk-cz-opt-d">
                {option.description}
                {option.cost !== "" && (
                  <>
                    {" "}
                    <span>{t("{0}.", option.cost)}</span>
                  </>
                )}
              </span>
            </button>
          );
        })}
      </div>
    </section>
  );
}
