import type { FactoringCompanyOverview } from "@/lib/graphql/factoring-company-page";
import { CheckIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { FactoringCompanyTab } from "../use-factoring-company-state";

type Step = {
  key: string;
  done: boolean;
  title: string;
  detail: string;
  action: string;
  tab: FactoringCompanyTab;
};

/**
 * What stands between a factor and its first schedule, in the order it has to happen. Shown in
 * place of the money path until the factor has funded anything.
 */
export function FactoringSetupSteps({
  overview,
  onTab,
}: {
  overview: FactoringCompanyOverview;
  onTab: (tab: FactoringCompanyTab) => void;
}) {
  const t = useT();
  const { setup, notices } = overview;
  const steps: Step[] = [
    {
      key: "agreement",
      done: setup.agreement,
      title: t("Agreement"),
      detail: setup.agreement
        ? t("An agreement is in force.")
        : t(
            "The advance, reserve, fee tiers and daily schedule time. Funding can't be worked out without one.",
          ),
      action: t("Open agreement"),
      tab: "agreement",
    },
    {
      key: "delivery",
      done: setup.delivery,
      title: t("Schedule delivery"),
      detail: setup.delivery
        ? t("Schedules go out by the method set up, and its last test passed.")
        : overview.delivery.configured
          ? t("Set up, but its last test failed. Schedules may not reach the factor.")
          : t("Email or SFTP, tested once. Schedules have nowhere to go until then."),
      action: overview.delivery.configured ? t("Open delivery") : t("Set up delivery"),
      tab: "delivery",
    },
    {
      key: "customers",
      done: setup.customers,
      title: t("Customers"),
      detail: setup.customers
        ? t("{0, plural, one {# customer is} other {# customers are}} assigned.", notices.assigned)
        : t("Assign customers to this factor in their billing profiles."),
      action: t("Open customers"),
      tab: "customers",
    },
    {
      key: "notices",
      done: setup.notices,
      title: t("Notices of assignment"),
      detail: setup.notices
        ? t("Every assigned customer has acknowledged a notice.")
        : t("Each assigned customer acknowledges a notice telling it to pay the factor."),
      action: t("Open customers"),
      tab: "customers",
    },
  ];
  const next = steps.find((step) => !step.done);

  return (
    <section aria-label={t("Setup")} className="rounded-lg border bg-card">
      <header className="flex min-h-9 items-center justify-between gap-2 border-b px-3 py-1.5">
        <h3 className="text-sm font-semibold">{t("This factor can't take invoices yet")}</h3>
        <span className="text-xs text-muted-foreground tabular-nums">
          {t("{0} of {1} done", setup.done, steps.length)}
        </span>
      </header>
      <ol className="divide-y">
        {steps.map((step, index) => (
          <li
            key={step.key}
            className="grid grid-cols-[1.25rem_minmax(0,1fr)_auto] items-start gap-3 px-3 py-3"
          >
            <span
              aria-hidden
              className={cn(
                "grid size-5 place-items-center rounded-full border text-2xs font-medium",
                step.done
                  ? "border-success bg-success text-foreground-on-solid"
                  : step === next
                    ? "border-foreground text-foreground"
                    : "border-border-strong text-foreground-faint",
              )}
            >
              {step.done ? <CheckIcon className="size-3" /> : index + 1}
            </span>
            <span className="flex min-w-0 flex-col gap-0.5">
              <span className="text-sm font-medium">
                {step.title}
                <span className="sr-only">{step.done ? t("Done") : t("Not done")}</span>
              </span>
              <span className="text-xs text-muted-foreground">{step.detail}</span>
            </span>
            {step.done ? null : (
              <Button
                type="button"
                size="sm"
                variant={step === next ? "default" : "outline"}
                onClick={() => onTab(step.tab)}
              >
                {step.action}
              </Button>
            )}
          </li>
        ))}
      </ol>
    </section>
  );
}
