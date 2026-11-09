import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { formatMinutesOfDay } from "@/lib/factoring";
import type { FactoringCompanyRecord } from "@/lib/graphql/factoring";
import type { FactoringCompanyOverview } from "@/lib/graphql/factoring-company-page";
import { factoringWorkspacePath } from "@/routes/factoring/use-factoring-state";
import {
  ClockIcon,
  Copy01Icon,
  DotsHorizontalIcon,
  LinkExternal01Icon,
  Mail01Icon,
  MarkerPin01Icon,
  Phone01Icon,
  RefreshCw01Icon,
  Send01Icon,
  Upload01Icon,
} from "@trenova/shared/components/icons";
import { StatusBadge } from "@trenova/shared/components/status-badge";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@trenova/shared/components/ui/dropdown-menu";
import { Tabs, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { useNow } from "@trenova/shared/hooks/use-now";
import { formatRelativeTime } from "@trenova/shared/i18n/format";
import { useT } from "@trenova/shared/i18n/use-t";
import type { ReactNode } from "react";
import { Link, useNavigate } from "react-router";
import { factoringCompanyPath, type FactoringCompanyTab } from "../use-factoring-company-state";

/** Up to two letters for the factor's mark: the first of its first two words. */
function initialsOf(name: string): string {
  const letters = name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((word) => word[0]?.toUpperCase() ?? "");
  return letters.join("") || "?";
}

function MetaItem({ icon, children }: { icon: ReactNode; children: ReactNode }) {
  return (
    <span className="inline-flex min-w-0 items-center gap-1.5 [&>svg]:size-3 [&>svg]:shrink-0">
      {icon}
      <span className="truncate">{children}</span>
    </span>
  );
}

/** When the overview was read, moving on the minute without redrawing the rest of the page. */
function UpdatedAgo({ at, onRefresh, busy }: { at: number; onRefresh: () => void; busy: boolean }) {
  const t = useT();
  const now = useNow("minute");
  return (
    <button
      type="button"
      onClick={onRefresh}
      disabled={busy}
      title={t("Read the figures again")}
      className="text-foreground-subtle hover:text-foreground ui-focus-ring inline-flex items-center gap-1.5 rounded-sm text-xs transition-colors disabled:opacity-60"
    >
      <RefreshCw01Icon className={busy ? "size-3 animate-spin" : "size-3"} />
      {t("Updated {0}", formatRelativeTime(Math.min(Math.round(at / 1000) - now, 0)))}
    </button>
  );
}

export type FactoringCompanyHeaderProps = {
  company: FactoringCompanyRecord;
  overview: FactoringCompanyOverview | undefined;
  overviewUpdatedAt: number;
  overviewFetching: boolean;
  onRefresh: () => void;
  /** The agreement in force, numbered by start date among all of the factor's agreements. */
  agreementNumber: number | null;
  tab: FactoringCompanyTab;
  visible: Record<FactoringCompanyTab, boolean>;
  onTab: (tab: FactoringCompanyTab) => void;
  canImport: boolean;
  canSubmit: boolean;
  onImport: () => void;
};

/**
 * The factor's record header: its mark, name, code, status and recourse, how to reach it and when
 * today's schedule closes, the two things done from here most (import a remittance, submit
 * invoices), and the page's tabs with what each holds.
 */
export function FactoringCompanyHeader({
  company,
  overview,
  overviewUpdatedAt,
  overviewFetching,
  onRefresh,
  agreementNumber,
  tab,
  visible,
  onTab,
  canImport,
  canSubmit,
  onImport,
}: FactoringCompanyHeaderProps) {
  const t = useT();
  const navigate = useNavigate();
  const { copy } = useCopyToClipboard();
  const agreement = overview?.agreement ?? null;
  const nonRecourse = agreement ? agreement.recourse === "NonRecourse" : company.nonRecourse;
  const address = [
    company.addressLine1,
    company.addressLine2,
    [company.city, [company.state?.abbreviation, company.postalCode].filter(Boolean).join(" ")]
      .filter(Boolean)
      .join(", "),
  ]
    .filter(Boolean)
    .join(", ");
  const deliveryAlert =
    overview?.attention.some(
      (item) => item.kind === "ScheduleFailed" || item.kind === "DeliveryFailing",
    ) ?? false;
  const ready = overview?.ready.count ?? 0;

  return (
    <header className="bg-card flex flex-col gap-2.5 border-b px-6 pt-3.5">
      <div className="flex flex-wrap items-start gap-x-3.5 gap-y-3">
        <span
          aria-hidden
          className="bg-sunken grid size-10 shrink-0 place-items-center rounded-md border font-mono text-sm font-semibold tracking-wide"
        >
          {initialsOf(company.name)}
        </span>
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <div className="flex flex-wrap items-center gap-2">
            <h2 className="text-lg leading-6 font-semibold">{company.name}</h2>
            <Badge variant="neutral" className="font-mono">
              {company.code}
            </Badge>
            <StatusBadge status={company.status} />
            <Badge variant="neutral">{nonRecourse ? t("Non-recourse") : t("Recourse")}</Badge>
          </div>
          <p className="text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-xs">
            <MetaItem icon={<MarkerPin01Icon />}>{address}</MetaItem>
            {company.email ? <MetaItem icon={<Mail01Icon />}>{company.email}</MetaItem> : null}
            {company.phoneNumber ? (
              <MetaItem icon={<Phone01Icon />}>{company.phoneNumber}</MetaItem>
            ) : null}
            {agreement ? (
              <MetaItem icon={<ClockIcon />}>
                {t("Today's schedule closes at {0}", formatMinutesOfDay(agreement.submissionTime))}
              </MetaItem>
            ) : null}
          </p>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {canImport ? (
            <Button type="button" size="sm" variant="outline" onClick={onImport}>
              <Upload01Icon className="size-3.5" />
              {t("Import remittance")}
            </Button>
          ) : null}
          {canSubmit ? (
            <Button
              size="sm"
              nativeButton={false}
              render={<Link to={factoringWorkspacePath("ready", company.id)} />}
              title={t("Opens the factoring workspace with this factor's ready invoices")}
            >
              <Send01Icon className="size-3.5" />
              {t("Submit invoices")}
              {ready > 0 ? (
                <span className="bg-background/15 grid h-4.5 min-w-4.5 place-items-center rounded-sm px-1 text-2xs font-medium tabular-nums">
                  {ready}
                </span>
              ) : null}
            </Button>
          ) : null}
          <DropdownMenu>
            <DropdownMenuTrigger
              render={<Button variant="outline" size="icon-sm" aria-label={t("More actions")} />}
            >
              <DotsHorizontalIcon className="size-4" />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-60">
              <DropdownMenuItem
                title={t("Open in the factoring workspace")}
                startContent={<LinkExternal01Icon className="size-3.5" />}
                onClick={() => void navigate(factoringWorkspacePath("ready", company.id))}
              />
              <DropdownMenuItem
                title={t("Copy link")}
                startContent={<Copy01Icon className="size-3.5" />}
                onClick={() =>
                  void copy(`${window.location.origin}${factoringCompanyPath(company.id)}`, {
                    withToast: true,
                  })
                }
              />
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <Tabs value={tab} onValueChange={(value) => onTab(value as FactoringCompanyTab)}>
        <div className="-mb-px flex items-center gap-3">
          <TabsList variant="underline" className="-ml-2.5 max-w-full min-w-0 overflow-x-auto">
            <TabsTab value="overview">{t("Overview")}</TabsTab>
            <TabsTab value="details">{t("Details")}</TabsTab>
            {visible.agreement ? (
              <TabsTab value="agreement">
                {t("Agreement")}
                {agreementNumber !== null ? (
                  <span className="text-foreground-subtle font-normal tabular-nums">
                    {t("v{0}", agreementNumber)}
                  </span>
                ) : null}
              </TabsTab>
            ) : null}
            {visible.customers ? (
              <TabsTab value="customers">
                {t("Customers")}
                {overview && overview.notices.assigned > 0 ? (
                  <span className="text-foreground-subtle font-normal tabular-nums">
                    {overview.notices.assigned}
                  </span>
                ) : null}
              </TabsTab>
            ) : null}
            <TabsTab value="delivery">
              {t("Delivery")}
              {deliveryAlert ? (
                <span
                  role="img"
                  aria-label={t("Delivery needs attention")}
                  className="bg-danger size-1.5 rounded-full"
                />
              ) : null}
            </TabsTab>
            {visible.ledger ? <TabsTab value="ledger">{t("Ledger")}</TabsTab> : null}
            {visible.activity ? <TabsTab value="activity">{t("Activity")}</TabsTab> : null}
          </TabsList>
          {overviewUpdatedAt > 0 ? (
            <span className="ml-auto hidden shrink-0 sm:inline-flex">
              <UpdatedAgo at={overviewUpdatedAt} onRefresh={onRefresh} busy={overviewFetching} />
            </span>
          ) : null}
        </div>
      </Tabs>
    </header>
  );
}
