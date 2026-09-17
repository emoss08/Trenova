import { coverageStanding } from "@/lib/carrier-intelligence";
import type { CarrierIntelProfile } from "@/lib/graphql/carrier-intelligence";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import {
  EmptyValue,
  Fact,
  FactGrid,
  MiniTable,
  Muted,
  SectionNote,
  SubHeading,
  currencyOrDash,
  dateOrDash,
  numberOrDash,
  textOrDash,
} from "../intel-facts";
import { StatusDot } from "../status-dot";
import { useCarrierIntelLabels } from "../use-carrier-intel-labels";
import { ProfileTab, type ProfileSectionProps } from "./profile-tab";

type InsuranceFiling = NonNullable<
  NonNullable<CarrierIntelProfile["insurance"]>["filings"]
>[number];

const FILING_STATUS_ACTIVE = "A";
const FILING_STATUS_HISTORICAL = "H";

function CoverageCell({ onFile, required }: { onFile: string | null; required: string | null }) {
  const standing = coverageStanding(onFile, required);
  return (
    <span className="inline-flex items-center justify-end gap-2" data-coverage={standing}>
      {standing === "short" || standing === "missing" ? <StatusDot tone="critical" /> : null}
      {currencyOrDash(onFile)}
    </span>
  );
}

function FilingStatus({ status }: { status: string | null }) {
  const t = useT();
  if (status === FILING_STATUS_ACTIVE) {
    return (
      <span className="inline-flex items-center gap-2">
        <StatusDot tone="success" />
        {t("Active")}
      </span>
    );
  }
  if (status === FILING_STATUS_HISTORICAL) {
    return <Muted>{t("Historical")}</Muted>;
  }
  return textOrDash(status);
}

function sortFilings(filings: readonly InsuranceFiling[]): InsuranceFiling[] {
  const rank = (filing: InsuranceFiling) => (filing.status === FILING_STATUS_ACTIVE ? 0 : 1);
  return filings
    .map((filing, index) => ({ filing, index }))
    .sort((a, b) => rank(a.filing) - rank(b.filing) || a.index - b.index)
    .map((entry) => entry.filing);
}

export function InsuranceSection({ profile, provider }: ProfileSectionProps) {
  const t = useT();
  const labels = useCarrierIntelLabels();
  const insurance = profile.insurance;

  return (
    <ProfileTab
      profile={profile}
      provider={provider}
      parts={[
        {
          section: "Insurance",
          hasData: insurance !== null,
          render: () => {
            if (!insurance) {
              return null;
            }
            const filings = sortFilings(insurance.filings ?? []);
            return (
              <div className="flex flex-col gap-5">
                {insurance.pendingCancelAt ? (
                  <p className="flex items-center gap-2 text-sm">
                    <StatusDot tone="critical" />
                    {t("A filing cancels on {0}", formatUnixDateMedium(insurance.pendingCancelAt))}
                  </p>
                ) : null}
                <MiniTable
                  columns={[
                    { id: "coverage", label: t("Coverage") },
                    { id: "onFile", label: t("On file"), align: "right" },
                    { id: "required", label: t("Required"), align: "right" },
                  ]}
                  rows={[
                    {
                      id: "bipd",
                      label: t("BIPD"),
                      onFile: insurance.bipdOnFile,
                      required: insurance.bipdRequired,
                    },
                    {
                      id: "cargo",
                      label: t("Cargo"),
                      onFile: insurance.cargoOnFile,
                      required: insurance.cargoRequired,
                    },
                    {
                      id: "bond",
                      label: t("Bond"),
                      onFile: insurance.bondOnFile,
                      required: insurance.bondRequired,
                    },
                  ].map((row) => ({
                    id: row.id,
                    cells: [
                      row.label,
                      <CoverageCell key="onFile" onFile={row.onFile} required={row.required} />,
                      currencyOrDash(row.required),
                    ],
                  }))}
                />
                <FactGrid>
                  <Fact label={t("Cancellations")}>{numberOrDash(insurance.cancelCount)}</Fact>
                  <Fact label={t("Last canceled")}>{dateOrDash(insurance.lastCanceledAt)}</Fact>
                </FactGrid>
                <div className="flex flex-col gap-1">
                  <SubHeading>{t("Filings")}</SubHeading>
                  {filings.length > 0 ? (
                    <MiniTable
                      columns={[
                        { id: "type", label: t("Type") },
                        { id: "status", label: t("Status") },
                        { id: "insurer", label: t("Insurer") },
                        { id: "coverage", label: t("Coverage"), align: "right" },
                        { id: "effective", label: t("Effective") },
                        { id: "cancels", label: t("Cancels") },
                      ]}
                      rows={filings.map((filing, index) => ({
                        id: `${filing.type}-${filing.policyNumber ?? ""}-${index}`,
                        cells: [
                          labels.filingType[filing.type],
                          <FilingStatus key="status" status={filing.status} />,
                          <span
                            key="insurer"
                            className="flex min-w-0 flex-col"
                            title={filing.policyNumber ?? undefined}
                          >
                            <span className="truncate">{textOrDash(filing.insurerName)}</span>
                            {filing.policyNumber ? (
                              <span className="text-muted-foreground truncate text-xs">
                                {filing.policyNumber}
                              </span>
                            ) : null}
                          </span>,
                          currencyOrDash(filing.coverage),
                          dateOrDash(filing.effectiveAt),
                          filing.cancelEffectiveAt ? (
                            <span key="cancels" className="inline-flex items-center gap-2">
                              {filing.status !== FILING_STATUS_HISTORICAL ? (
                                <StatusDot tone="critical" />
                              ) : null}
                              {formatUnixDateMedium(filing.cancelEffectiveAt)}
                              {filing.cancelMethod ? <Muted>{filing.cancelMethod}</Muted> : null}
                            </span>
                          ) : (
                            <EmptyValue key="cancels" />
                          ),
                        ],
                      }))}
                    />
                  ) : (
                    <SectionNote>{t("No filings on record.")}</SectionNote>
                  )}
                </div>
              </div>
            );
          },
        },
      ]}
    />
  );
}
