import { SectionPanel } from "@/components/section-panel";
import type { AgentControl } from "@/lib/graphql/agent-control";
import {
  Award01Icon,
  Database01Icon,
  Dataflow03Icon,
  Edit02Icon,
  GraduationHat01Icon,
  Speedometer03Icon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";

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

  const rows: { icon: IconComponent; label: string; value: string; on: boolean }[] = [
    {
      icon: Award01Icon,
      label: t("Earned autonomy"),
      value: control.earnedAutonomy
        ? t("On · after {0} clean", control.promotionThreshold)
        : t("Off"),
      on: control.earnedAutonomy,
    },
    {
      icon: GraduationHat01Icon,
      label: t("Learn from their work"),
      value: control.learningOff ? t("Off") : t("On"),
      on: !control.learningOff,
    },
    {
      icon: Speedometer03Icon,
      label: t("Monthly allowance"),
      value:
        control.personMonthlyMessages > 0
          ? t("{0} per person", control.personMonthlyMessages.toLocaleString())
          : t("Unlimited"),
      on: control.personMonthlyMessages > 0,
    },
    {
      icon: Database01Icon,
      label: t("Share corrections"),
      value: control.aiTrainingConsent ? t("On") : t("Off"),
      on: control.aiTrainingConsent,
    },
  ];

  return (
    <SectionPanel
      title={t("Organization-wide")}
      action={
        canEdit ? (
          <Button size="xs" variant="outline" onClick={onEdit}>
            <Edit02Icon className="size-3" />
            {t("Edit")}
          </Button>
        ) : undefined
      }
    >
      <ul className="flex flex-col p-1.5">
        {rows.map((row) => (
          <li key={row.label}>
            <SettingRow {...row} onClick={canEdit ? onEdit : undefined} />
          </li>
        ))}
        {routing && (
          <li>
            <SettingRow
              icon={Dataflow03Icon}
              label={t("Routing")}
              value={t("{0} of {1} covered", routing.covered, routing.total)}
              on={routing.covered === routing.total}
              warn={routing.covered < routing.total}
              onClick={onOpenRouting}
            />
          </li>
        )}
      </ul>
    </SectionPanel>
  );
}

function SettingRow({
  icon: Icon,
  label,
  value,
  on,
  warn = false,
  onClick,
}: {
  icon: IconComponent;
  label: string;
  value: string;
  on: boolean;
  warn?: boolean;
  onClick?: () => void;
}) {
  return (
    <button
      type="button"
      disabled={!onClick}
      onClick={onClick}
      className="ui-focus-ring flex w-full items-center gap-2.5 rounded-control px-2 py-1.5 text-left text-sm enabled:hover:bg-muted"
    >
      <Icon className="size-3.5 text-muted-foreground" />
      <span className="flex-1 truncate">{label}</span>
      <em
        className={cn(
          "text-xs not-italic",
          warn ? "text-warning" : on ? "text-foreground" : "text-muted-foreground",
        )}
      >
        {value}
      </em>
    </button>
  );
}
