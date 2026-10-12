import type { AgentControl } from "@/lib/graphql/agent-control";
import { useT } from "@trenova/shared/i18n/use-t";
import { Ic, type IcName } from "../kit/ic";
import { SecH } from "../kit/layout";
import { Button } from "@trenova/shared/components/ui/button";

type OrganizationWideProps = {
  control: AgentControl;
  /** Tasks some enabled provider takes, of all there are; absent with no provider. */
  routing: { covered: number; total: number } | null;
  canEdit: boolean;
  onEdit: () => void;
  onOpenRouting: () => void;
};

/** The settings that apply to every agent, one row each, all opening the policy editor. */
export function OrganizationWide({
  control,
  routing,
  canEdit,
  onEdit,
  onOpenRouting,
}: OrganizationWideProps) {
  const t = useT();

  const rows: { icon: IcName; label: string; value: string; on: boolean }[] = [
    {
      icon: "award",
      label: t("Earned autonomy"),
      value: control.earnedAutonomy
        ? t("On · after {0} clean", control.promotionThreshold)
        : t("Off"),
      on: control.earnedAutonomy,
    },
    {
      icon: "brain",
      label: t("Learn from their work"),
      value: control.learningOff ? t("Off") : t("On"),
      on: !control.learningOff,
    },
    {
      icon: "gauge",
      label: t("Monthly allowance"),
      value:
        control.personMonthlyMessages > 0
          ? t("{0} per person", control.personMonthlyMessages.toLocaleString())
          : t("Unlimited"),
      on: control.personMonthlyMessages > 0,
    },
    {
      icon: "database",
      label: t("Share corrections"),
      value: control.aiTrainingConsent ? t("On") : t("Off"),
      on: control.aiTrainingConsent,
    },
  ];

  return (
    <section className="sec">
      <SecH
        t={t("Organization-wide")}
        r={
          canEdit ? (
            <Button type="button" variant="outline" size="sm" onClick={onEdit}>
              <Ic n="edit" s={12} />
              {t("Edit")}
            </Button>
          ) : undefined
        }
      />
      <div className="os">
        {rows.map((row) => (
          <button
            key={row.label}
            type="button"
            className="os-r"
            disabled={!canEdit}
            onClick={onEdit}
          >
            <Ic n={row.icon} s={13} />
            <span>{row.label}</span>
            <em className={row.on ? "on" : undefined}>{row.value}</em>
          </button>
        ))}
        {routing && (
          <button type="button" className="os-r" onClick={onOpenRouting}>
            <Ic n="route" s={13} />
            <span>{t("Routing")}</span>
            <em className={routing.covered === routing.total ? "on" : "w"}>
              {t("{0} of {1} covered", routing.covered, routing.total)}
            </em>
          </button>
        )}
      </div>
    </section>
  );
}
