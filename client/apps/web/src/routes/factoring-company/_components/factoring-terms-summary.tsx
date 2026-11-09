import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { FactoringAgreementStatusBadge } from "@/components/factoring/factoring-badges";
import { formatBasisPoints, formatMinutesOfDay } from "@/lib/factoring";
import type { FactoringAgreement } from "@/lib/graphql/factoring";
import { ArrowNarrowRightIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatMinor } from "@trenova/shared/lib/money";
import type { ReactNode } from "react";
import type { FactoringCompanyTab } from "../use-factoring-company-state";

const strong = {
  b: (children: ReactNode) => <b className="text-foreground font-medium">{children}</b>,
};

/** The agreement in force said the way its contract would: advance, fees, risk and the daily cutoff. */
export function FactoringTermsSummary({
  agreement,
  onTab,
}: {
  agreement: FactoringAgreement | null | undefined;
  onTab: (tab: FactoringCompanyTab) => void;
}) {
  const t = useT();
  const rt = useRichT();

  if (!agreement) {
    return (
      <SectionPanel title={t("Terms in force")}>
        <SectionPanelQuiet>
          {t("No agreement is in force, so nothing assigned to this factor can be sent to it.")}
        </SectionPanelQuiet>
        <div className="px-3 pb-3">
          <Button type="button" size="sm" variant="outline" onClick={() => onTab("agreement")}>
            {t("Open agreement")}
          </Button>
        </div>
      </SectionPanel>
    );
  }

  const tiers = [...agreement.feeTiers].sort((a, b) => a.daysFrom - b.daysFrom);
  const first = tiers[0];
  const last = tiers.at(-1);

  return (
    <SectionPanel
      title={t("Terms in force")}
      action={<FactoringAgreementStatusBadge status={agreement.status} />}
    >
      <div className="flex flex-col gap-2 p-3">
        <p className="text-muted-foreground text-sm leading-relaxed text-pretty">
          {rt(
            "The factor advances <b>{0}</b> of each invoice within <b>{1, plural, one {# business day} other {# business days}}</b>.",
            strong,
            formatBasisPoints(agreement.advanceRateBp),
            agreement.fundingDays,
          )}{" "}
          {first && last && tiers.length > 1 && first.daysTo != null
            ? rt(
                "The fee is <b>{0}</b> with under {1, plural, one {# day} other {# days}} left on an invoice's terms, rising to <b>{2}</b> at {3, plural, one {# day} other {# days}} or more.",
                strong,
                formatBasisPoints(first.rateBp),
                first.daysTo,
                formatBasisPoints(last.rateBp),
                last.daysFrom,
              )
            : first
              ? rt("The fee is <b>{0}</b>.", strong, formatBasisPoints(first.rateBp))
              : t("The agreement has no fee tiers.")}{" "}
          {agreement.minimumFeeMinor > 0
            ? rt("No fee is less than <b>{0}</b>.", strong, formatMinor(agreement.minimumFeeMinor))
            : null}{" "}
          {agreement.recourse === "Recourse"
            ? rt(
                "It's recourse: an unpaid invoice comes back to you after <b>{0, plural, one {# day} other {# days}}</b>.",
                strong,
                agreement.chargebackWindowDays,
              )
            : rt("It's <b>non-recourse</b>: the factor carries the credit risk.", strong)}{" "}
          {rt(
            "Schedules close daily at <b>{0}</b>.",
            strong,
            formatMinutesOfDay(agreement.submissionTime),
          )}
        </p>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          className="self-start"
          onClick={() => onTab("agreement")}
        >
          {t("Try the terms on an invoice")}
          <ArrowNarrowRightIcon className="size-3.5" />
        </Button>
      </div>
    </SectionPanel>
  );
}
