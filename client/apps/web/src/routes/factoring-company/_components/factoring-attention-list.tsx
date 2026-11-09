import { HoverGlideGroup } from "@/components/hover-glide";
import { SectionPanel, SectionPanelQuiet } from "@/components/section-panel";
import { recordPath } from "@/config/record-links";
import type {
  FactoringAttention,
  FactoringCompanyOverview,
} from "@/lib/graphql/factoring-company-page";
import { factoringWorkspacePath } from "@/routes/factoring/use-factoring-state";
import {
  AlertTriangleIcon,
  CoinsStacked01Icon,
  FileCheck02Icon,
  FileSearch02Icon,
  Hourglass01Icon,
  Mail01Icon,
  Server01Icon,
  type IconComponent,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { formatMinor } from "@trenova/shared/lib/money";
import { cn } from "@trenova/shared/lib/utils";
import type { FactoringAttentionKind } from "@trenova/graphql/generated/graphql";
import { Link } from "react-router";
import type { FactoringCompanyTab } from "../use-factoring-company-state";

const KIND_ICONS: Record<FactoringAttentionKind, IconComponent> = {
  ScheduleFailed: Server01Icon,
  DeliveryFailing: Server01Icon,
  NoDelivery: Server01Icon,
  NoAgreement: FileCheck02Icon,
  ChargebacksDue: Hourglass01Icon,
  FundingOverdue: CoinsStacked01Icon,
  RemittanceWaiting: FileSearch02Icon,
  NoticesMissing: Mail01Icon,
  NoticesAwaiting: Mail01Icon,
  ReadyWaiting: AlertTriangleIcon,
};

const WELL_TONES: Record<string, string> = {
  danger: "bg-danger-subtle border-danger-border text-danger-subtle-foreground",
  warning: "bg-warning-subtle border-warning-border text-warning-subtle-foreground",
  neutral: "bg-neutral-subtle border-neutral-border text-neutral-subtle-foreground",
};

type Action = { label: string; to?: string; tab?: FactoringCompanyTab };

type Worded = { title: string; detail: string; action: Action };

function word(
  item: FactoringAttention,
  t: TranslateFn,
  money: (minor: number) => string,
  amounts: boolean,
  factoringCompanyId: string,
): Worded {
  const date = (at: number | null | undefined) => (at ? formatUnixDate(at) : "");
  switch (item.kind) {
    case "ScheduleFailed":
      return {
        title:
          item.count === 1
            ? t("Schedule {0} didn't reach the factor", item.number ?? "")
            : t("{0, plural, other {# schedules}} didn't reach the factor", item.count),
        detail: item.detail
          ? t(
              "{0} {1, plural, one {# invoice waits} other {# invoices wait}} to go again.",
              item.detail,
              item.lineCount,
            )
          : t(
              "{0, plural, one {# invoice waits} other {# invoices wait}} to go again.",
              item.lineCount,
            ),
        action: { label: t("Fix delivery"), tab: "delivery" },
      };
    case "DeliveryFailing":
      return {
        title: t("The last delivery test failed"),
        detail: item.detail
          ? t("{0} The next schedule may not reach the factor until it passes.", item.detail)
          : t("The next schedule may not reach the factor until it passes."),
        action: { label: t("Fix delivery"), tab: "delivery" },
      };
    case "NoDelivery":
      return {
        title: t("Delivery isn't set up"),
        detail: t("Schedules have nowhere to go until email or SFTP is set up and tested."),
        action: { label: t("Set up delivery"), tab: "delivery" },
      };
    case "NoAgreement":
      return {
        title: t("No agreement is in force"),
        detail: t(
          "Invoices assigned to this factor aren't ready to factor until an agreement is active.",
        ),
        action: { label: t("Open agreement"), tab: "agreement" },
      };
    case "ChargebacksDue":
      return {
        title: t(
          "{0, plural, one {# recourse invoice reaches} other {# recourse invoices reach}} the chargeback window by {1}",
          item.count,
          date(item.at),
        ),
        detail:
          amounts && item.invoiceNumber
            ? t(
                "{0} comes back to you if the customers haven't paid the factor. The first is {1}, {2}.",
                money(item.amountMinor),
                item.invoiceNumber,
                item.customerName ?? "",
              )
            : amounts
              ? t(
                  "{0} comes back to you if the customers haven't paid the factor.",
                  money(item.amountMinor),
                )
              : t("They come back to you if the customers haven't paid the factor."),
        action: {
          label: t("Review invoices"),
          to: factoringWorkspacePath("funded", factoringCompanyId),
        },
      };
    case "FundingOverdue":
      return {
        title:
          item.count === 1
            ? t("Schedule {0} is past its funding date", item.number ?? "")
            : t("{0, plural, other {# schedules}} are past their funding date", item.count),
        detail: amounts
          ? t(
              "Funding was due {0}. {1} expected and not yet recorded.",
              date(item.at),
              money(item.amountMinor),
            )
          : t("Funding was due {0} and has not been recorded.", date(item.at)),
        action: {
          label: t("Open schedules"),
          to: factoringWorkspacePath("schedules", factoringCompanyId),
        },
      };
    case "RemittanceWaiting":
      return {
        title:
          item.count === 1
            ? t("The remittance from {0} is waiting to be applied", date(item.at))
            : t("{0, plural, other {# remittances}} are waiting to be applied", item.count),
        detail:
          item.exceptionCount > 0
            ? t(
                "{0, plural, one {# line} other {# lines}}, {1}. {2, plural, one {# line needs} other {# lines need}} resolving before it books.",
                item.lineCount,
                amounts ? money(item.amountMinor) : (item.fileName ?? ""),
                item.exceptionCount,
              )
            : t(
                "{0, plural, one {# line} other {# lines}}, {1}. Every line is matched.",
                item.lineCount,
                amounts ? money(item.amountMinor) : (item.fileName ?? ""),
              ),
        action: {
          label: t("Review remittance"),
          to: item.remittanceId ? recordPath("factoring_remittance", item.remittanceId) : undefined,
        },
      };
    case "NoticesMissing":
      return {
        title: t(
          "{0, plural, one {# customer has} other {# customers have}} no notice in force",
          item.count,
        ),
        detail: t(
          "Their invoices print the factor's remit-to, but they haven't acknowledged paying it.",
        ),
        action: { label: t("See customers"), tab: "customers" },
      };
    case "NoticesAwaiting":
      return {
        title: t(
          "{0, plural, one {# notice is} other {# notices are}} waiting on the customer",
          item.count,
        ),
        detail: t("Drafted or sent, and not yet acknowledged."),
        action: { label: t("See customers"), tab: "customers" },
      };
    case "ReadyWaiting":
      return {
        title: t(
          "{0, plural, one {# invoice has} other {# invoices have}} been ready for over a day",
          item.count,
        ),
        detail: amounts
          ? t("{0} that the factor could be funding.", money(item.amountMinor))
          : t("Ready to send on the next schedule."),
        action: {
          label: t("Submit invoices"),
          to: factoringWorkspacePath("ready", factoringCompanyId),
        },
      };
  }
}

/** What on the factor needs someone, worst first, each with the one step that moves it. */
export function FactoringAttentionList({
  overview,
  factoringCompanyId,
  onTab,
}: {
  overview: FactoringCompanyOverview;
  factoringCompanyId: string;
  onTab: (tab: FactoringCompanyTab) => void;
}) {
  const t = useT();
  const money = (minor: number) => formatMinor(minor, overview.currencyCode);

  return (
    <SectionPanel title={t("Needs attention")} count={overview.attention.length}>
      {overview.attention.length === 0 ? (
        <SectionPanelQuiet>
          {t("Nothing needs anyone. Schedules are going out and the factor is funding on time.")}
        </SectionPanelQuiet>
      ) : (
        <HoverGlideGroup className="bg-card p-1.5">
          <ul className="flex flex-col gap-px">
            {overview.attention.map((item, index) => {
              const worded = word(item, t, money, overview.amountsIncluded, factoringCompanyId);
              const Icon = KIND_ICONS[item.kind];
              const tab = worded.action.tab;
              return (
                <li
                  key={`${item.kind}-${index}`}
                  data-glide-target
                  className="relative grid grid-cols-[1.75rem_minmax(0,1fr)_auto] items-start gap-3 rounded-md px-2 py-2.5"
                >
                  <span
                    aria-hidden
                    className={cn(
                      "grid size-7 place-items-center rounded-md border",
                      WELL_TONES[item.tone] ?? WELL_TONES.neutral,
                    )}
                  >
                    <Icon className="size-3.5" />
                  </span>
                  <span className="flex min-w-0 flex-col gap-0.5">
                    <span className="text-sm font-medium">{worded.title}</span>
                    <span className="text-xs leading-relaxed text-muted-foreground">
                      {worded.detail}
                    </span>
                  </span>
                  {worded.action.to ? (
                    <Button
                      size="sm"
                      variant="outline"
                      nativeButton={false}
                      render={<Link to={worded.action.to} />}
                    >
                      {worded.action.label}
                    </Button>
                  ) : tab ? (
                    <Button type="button" size="sm" variant="outline" onClick={() => onTab(tab)}>
                      {worded.action.label}
                    </Button>
                  ) : null}
                </li>
              );
            })}
          </ul>
        </HoverGlideGroup>
      )}
    </SectionPanel>
  );
}
