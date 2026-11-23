import { Link } from "@trenova/shared/components/link";
import {
  MetaItem,
  RecordHeader,
  RecordMark,
  RecordTabs,
  TabCount,
  TabDot,
  UpdatedAgo,
} from "@/components/record-header";
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
import { TabsTab } from "@trenova/shared/components/ui/tabs";
import { useT } from "@trenova/shared/i18n/use-t";
import { useNavigate } from "@tanstack/react-router";
import { factoringCompanyPath, type FactoringCompanyTab } from "../use-factoring-company-state";

export type FactoringCompanyHeaderProps = {
  company: FactoringCompanyRecord;
  overview: FactoringCompanyOverview | undefined;
  overviewUpdatedAt: number;
  overviewFetching: boolean;
  onRefresh: () => void;
  /** The agreement in force, numbered by start date among all of the factor's agreements. */
  agreementNumber: number | null;
  /** The Details tab holds edits that are not saved, so the tab is marked from any other. */
  detailsUnsaved: boolean;
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
  detailsUnsaved,
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
    <RecordHeader
      mark={<RecordMark name={company.name} />}
      title={company.name}
      badges={
        <>
          <Badge variant="neutral" className="font-mono">
            {company.code}
          </Badge>
          <StatusBadge status={company.status} />
          <Badge variant="neutral">{nonRecourse ? t("Non-recourse") : t("Recourse")}</Badge>
        </>
      }
      meta={
        <>
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
        </>
      }
      actions={
        <>
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
                <span className="grid h-4.5 min-w-4.5 place-items-center rounded-sm bg-background/15 px-1 text-2xs font-medium tabular-nums">
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
                onClick={() => void navigate({ to: factoringWorkspacePath("ready", company.id) })}
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
        </>
      }
    >
      <RecordTabs<FactoringCompanyTab>
        value={tab}
        onValueChange={onTab}
        trailing={
          overviewUpdatedAt > 0 ? (
            <UpdatedAgo at={overviewUpdatedAt} onRefresh={onRefresh} busy={overviewFetching} />
          ) : null
        }
      >
        <TabsTab value="overview">{t("Overview")}</TabsTab>
        <TabsTab value="details">
          {t("Details")}
          {detailsUnsaved ? <TabDot tone="warning" label={t("Unsaved changes")} /> : null}
        </TabsTab>
        {visible.agreement ? (
          <TabsTab value="agreement">
            {t("Agreement")}
            {agreementNumber !== null ? <TabCount>{t("v{0}", agreementNumber)}</TabCount> : null}
          </TabsTab>
        ) : null}
        {visible.customers ? (
          <TabsTab value="customers">
            {t("Customers")}
            {overview && overview.notices.assigned > 0 ? (
              <TabCount>{overview.notices.assigned}</TabCount>
            ) : null}
          </TabsTab>
        ) : null}
        <TabsTab value="delivery">
          {t("Delivery")}
          {deliveryAlert ? <TabDot tone="danger" label={t("Delivery needs attention")} /> : null}
        </TabsTab>
        {visible.ledger ? <TabsTab value="ledger">{t("Ledger")}</TabsTab> : null}
        {visible.activity ? <TabsTab value="activity">{t("Activity")}</TabsTab> : null}
      </RecordTabs>
    </RecordHeader>
  );
}
