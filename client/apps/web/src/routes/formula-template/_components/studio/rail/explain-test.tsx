import { NovaButton } from "@/components/tariff-studio/nova-button";
import { TickedText } from "@/components/tariff-studio/ticked-text";
import { describeApiError } from "@/lib/api-error-message";
import { apiService } from "@/services/api";
import { useMutation } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { useT } from "@trenova/shared/i18n/use-t";
import type { FormulaTestVerdict } from "@trenova/shared/types/formula-template";
import { useStudio } from "../studio-context";

const VERDICT_TONE = {
  bug: "danger",
  intended: "warning",
  unclear: "neutral",
} as const satisfies Record<FormulaTestVerdict, string>;

/**
 * Under a failing test: Nova reads the failure against the draft and says why it fails, and
 * whether the tariff has a bug or the test expects the wrong thing. The answer belongs to the
 * source it was asked about; once the source changes it is asked again.
 */
export function ExplainTest({ name }: { name: string }) {
  const t = useT();
  const { draftContent } = useStudio();
  const explain = useMutation({
    mutationFn: () => {
      const content = draftContent();
      return apiService.formulaTemplateService
        .explainTest(content, name)
        .then((answer) => ({ answer, source: content.source }));
    },
  });
  const current = explain.data && explain.data.source === draftContent().source;

  if (explain.data && current) {
    const { answer } = explain.data;
    return (
      <div className="bg-card flex flex-col gap-1.5 rounded-md border px-2.5 py-2">
        <div className="flex items-center gap-1.5">
          <Badge variant={VERDICT_TONE[answer.verdict]}>
            {answer.verdict === "bug"
              ? t("The tariff looks wrong")
              : answer.verdict === "intended"
                ? t("The test looks wrong")
                : t("Not clear which is wrong")}
          </Badge>
          {!answer.generated && (
            <span className="text-foreground-subtle text-2xs">{t("Read from the failure")}</span>
          )}
        </div>
        <p className="text-sm leading-relaxed">
          <TickedText text={answer.why} />
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col items-start gap-1">
      <NovaButton
        busy={explain.isPending}
        disabled={explain.isPending}
        onClick={() => explain.mutate()}
      >
        {t("Why does this fail?")}
      </NovaButton>
      {explain.isError && (
        <p className="text-danger text-xs">
          {describeApiError(explain.error, t("Nova could not explain the test."))}
        </p>
      )}
    </div>
  );
}
