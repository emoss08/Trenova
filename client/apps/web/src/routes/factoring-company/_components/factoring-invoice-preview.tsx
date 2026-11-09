import { LazyImage } from "@/components/image";
import { useCenteredStickyTop } from "@/hooks/use-centered-sticky-top";
import { usePermission } from "@/hooks/use-permission";
import { factorRemitLines, standardNoticeOfAssignment } from "@/lib/factoring";
import { organization } from "@/lib/queries/organization";
import { userOrganization } from "@/lib/queries/user";
import type { FactoringCompany } from "@/types/factoring-company";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixDate } from "@trenova/shared/lib/date";
import { cityStatePostal, mailingAddressLines } from "@trenova/shared/lib/mailing-address";
import { formatMinor } from "@trenova/shared/lib/money";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { useState, type ReactNode } from "react";
import { useWatch, type Control } from "react-hook-form";

/** The parts of the invoice this page's details print, so a section can point at its own. */
export type InvoiceRegion = "name" | "address" | "contact" | "instructions" | "notice";

export type FactorState = { abbreviation: string | null; countryName: string | null };

const SECONDS_PER_DAY = 86_400;
const NET_DAYS = 30;

const SAMPLE = {
  invoiceNumber: "INV-2026-1042",
  billTo: {
    name: "Halstead Grocery Group",
    lines: ["Accounts Payable", "900 Distribution Pkwy", "Fort Worth, TX 76106"],
  },
  shipper: { name: "Rio Grande Produce", lines: ["1200 Airport Rd", "Pharr, TX 78577"] },
  consignee: { name: "Halstead DC 14", lines: ["900 Distribution Pkwy", "Fort Worth, TX 76106"] },
  refs: [
    ["Pro Number", "TRV-884120"],
    ["BOL", "BOL-77401"],
    ["Customer PO", "PO-5591004"],
  ],
  charges: [
    { description: "Linehaul", quantity: "612 mi", rate: "2.85", amountMinor: 174_420 },
    { description: "Fuel surcharge", quantity: "612 mi", rate: "0.42", amountMinor: 25_704 },
    {
      description: "Detention — Halstead DC 14",
      quantity: "2.5 hr",
      rate: "65.00",
      amountMinor: 16_250,
    },
    { description: "Lumper", quantity: "1", rate: "175.00", amountMinor: 17_500 },
  ],
} as const;

const SAMPLE_TOTAL_MINOR = SAMPLE.charges.reduce((sum, charge) => sum + charge.amountMinor, 0);

function Tag({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div
      className={cn(
        "mb-1.5 font-mono text-3xs tracking-widest text-doc-ink-2 uppercase",
        className,
      )}
    >
      [ {children} ]
    </div>
  );
}

/** A part of the invoice this page fills in, lit while the section that sets it has the person's attention. */
function Marked({
  on,
  className,
  children,
}: {
  on: boolean;
  className?: string;
  children: ReactNode;
}) {
  return (
    <div
      data-marked={on || undefined}
      className={cn(
        "-m-1 rounded-xs p-1 outline-offset-2 transition-colors duration-150",
        on && "bg-brand-subtle outline-1 outline-brand outline-dashed",
        className,
      )}
    >
      {children}
    </div>
  );
}

function useIssuer() {
  const t = useT();
  const orgs = useQuery(userOrganization.all());
  const current = orgs.data?.find((org) => org.isCurrent);
  const { allowed: canReadOrganization } = usePermission(Resource.Organization, Operation.Read);
  const detail = useQuery({
    ...organization.detail(current?.id ?? ""),
    enabled: canReadOrganization && Boolean(current?.id),
  });
  const logoUrl = current?.logoUrl ?? "";
  const absoluteLogo = /^https?:\/\//.test(logoUrl);
  const resolvedLogo = useQuery({
    ...organization.logo(current?.id ?? ""),
    enabled: Boolean(current?.id) && Boolean(logoUrl) && !absoluteLogo,
    retry: false,
  });
  const org = detail.data;

  return {
    name: org?.name ?? current?.name ?? t("Your organization"),
    lines: org
      ? mailingAddressLines({
          line1: org.addressLine1,
          line2: org.addressLine2,
          city: org.city,
          state: org.state?.abbreviation,
          postalCode: org.postalCode,
          country: org.state?.countryName,
        })
      : current
        ? [cityStatePostal(current.city, current.state, null)].filter(Boolean)
        : [],
    logo: absoluteLogo ? logoUrl : (resolvedLogo.data ?? null),
  };
}

/**
 * The invoice a customer of this factor receives, laid out as the Ledger invoice template prints
 * it: your organization's masthead, the sample shipment and charges, and the factor's part drawn
 * from the form as it changes. The remit-to lines and the notice follow the server's own rules,
 * so the preview prints what the document will.
 */
export function FactoringInvoicePreview({
  control,
  state,
  active,
}: {
  control: Control<FactoringCompany>;
  state: FactorState;
  active: ReadonlySet<InvoiceRegion>;
}) {
  const t = useT();
  const issuer = useIssuer();
  const [name, line1, line2, city, postalCode, instructions, email, phoneNumber, notice] = useWatch(
    {
      control,
      name: [
        "name",
        "addressLine1",
        "addressLine2",
        "city",
        "postalCode",
        "remittanceInstructions",
        "email",
        "phoneNumber",
        "noticeOfAssignment",
      ],
    },
  );
  const payee = name?.trim() || t("The factor's legal name");
  const address = mailingAddressLines({
    line1,
    line2,
    city,
    state: state.abbreviation,
    postalCode,
    country: state.countryName,
  });
  const instructionLines = factorRemitLines({ remittanceInstructions: instructions });
  const contactLines = factorRemitLines({ email, phoneNumber });
  const remitLines = factorRemitLines({
    line1,
    line2,
    city,
    state: state.abbreviation,
    postalCode,
    country: state.countryName,
    remittanceInstructions: instructions,
    email,
    phoneNumber,
  });
  const printedNotice = notice?.trim() || standardNoticeOfAssignment(payee);
  const paperRef = useCenteredStickyTop<HTMLElement>();
  const [issued] = useState(() => Math.floor(Date.now() / 1000));
  const due = issued + NET_DAYS * SECONDS_PER_DAY;

  return (
    <aside aria-label={t("Invoice preview")} className="flex min-w-0 flex-1 flex-col gap-2.5">
      <div className="flex-1 rounded-lg border bg-sunken p-3 sm:p-5">
        <article
          ref={paperRef}
          aria-label={t("Sample invoice")}
          className="mx-auto flex w-full max-w-176 flex-col gap-4 border border-doc-rule bg-doc-paper px-6 py-7 font-sans text-xs text-doc-ink tabular-nums sm:px-9 sm:py-9 xl:sticky"
        >
          <header className="flex items-start justify-between gap-6 border-b border-doc-ink pb-4">
            <div className="flex min-w-0 items-center gap-3">
              {issuer.logo ? (
                <LazyImage
                  src={issuer.logo}
                  alt={t("{0} logo", issuer.name)}
                  className="block max-h-11 max-w-30 object-contain"
                />
              ) : null}
              <div className="min-w-0">
                <div className="text-base font-semibold tracking-tight">{issuer.name}</div>
                <div className="text-2xs text-doc-ink-2">{issuer.lines.join(" · ")}</div>
              </div>
            </div>
            <div className="grid shrink-0 justify-items-end gap-0.5 text-right">
              <Tag className="mb-0">{t("Invoice")}</Tag>
              <div className="text-lg font-semibold tracking-tight">{SAMPLE.invoiceNumber}</div>
              <dl className="mt-1 grid grid-cols-[auto_auto] gap-x-3 text-2xs">
                <dt className="text-doc-ink-2">{t("Issued")}</dt>
                <dd className="m-0 font-medium">{formatUnixDate(issued)}</dd>
                <dt className="text-doc-ink-2">{t("Due")}</dt>
                <dd className="m-0 font-medium">{formatUnixDate(due)}</dd>
              </dl>
              <div className="mt-1.5 flex items-baseline gap-2.5 border-t border-doc-rule pt-1.5">
                <Tag className="mb-0">{t("Amount due")}</Tag>
                <span className="text-sm font-semibold">{formatMinor(SAMPLE_TOTAL_MINOR)}</span>
              </div>
            </div>
          </header>

          <section className="grid grid-cols-2 gap-6">
            <div className="min-w-0">
              <Tag>{t("Bill To")}</Tag>
              <div className="font-semibold">{SAMPLE.billTo.name}</div>
              {SAMPLE.billTo.lines.map((line) => (
                <div key={line} className="text-doc-ink-2">
                  {line}
                </div>
              ))}
            </div>
            <div className="min-w-0">
              <Tag>{t("Remit To")}</Tag>
              <Marked on={active.has("name")}>
                <div className="font-semibold">{payee}</div>
              </Marked>
              <Marked on={active.has("address")}>
                {address.length > 0 ? (
                  address.map((line) => (
                    <div key={line} className="text-doc-ink-2">
                      {line}
                    </div>
                  ))
                ) : (
                  <div className="text-doc-ink-3">{t("The remit-to address")}</div>
                )}
              </Marked>
              {instructionLines.length > 0 ? (
                <Marked on={active.has("instructions")}>
                  {instructionLines.map((line, index) => (
                    <div key={`${index}-${line}`} className="break-words text-doc-ink-2">
                      {line}
                    </div>
                  ))}
                </Marked>
              ) : null}
              {contactLines.length > 0 ? (
                <Marked on={active.has("contact")}>
                  {contactLines.map((line) => (
                    <div key={line} className="break-words text-doc-ink-2">
                      {line}
                    </div>
                  ))}
                </Marked>
              ) : null}
            </div>
          </section>

          <Marked on={active.has("notice") || active.has("name")}>
            <section className="border-[1.5px] border-doc-ink px-3.5 py-2.5">
              <Tag>{t("Notice of assignment")}</Tag>
              <p className="m-0 font-medium whitespace-pre-line">{printedNotice}</p>
            </section>
          </Marked>

          <section className="grid grid-cols-2 gap-6 border border-doc-rule bg-doc-paper-warm px-4 py-3">
            {[
              { tag: t("Shipper"), party: SAMPLE.shipper },
              { tag: t("Consignee"), party: SAMPLE.consignee },
            ].map(({ tag, party }) => (
              <div key={party.name} className="min-w-0">
                <Tag>{tag}</Tag>
                <div className="font-semibold">{party.name}</div>
                {party.lines.map((line) => (
                  <div key={line} className="text-doc-ink-2">
                    {line}
                  </div>
                ))}
              </div>
            ))}
          </section>

          <div className="-mt-1.5 flex flex-wrap gap-x-4 gap-y-1 font-mono text-2xs text-doc-ink-2">
            {SAMPLE.refs.map(([label, value]) => (
              <span key={label}>
                {label} <span className="font-medium text-doc-ink">{value}</span>
              </span>
            ))}
          </div>

          <section>
            <Tag>{t("Charges")}</Tag>
            <table className="w-full border-collapse text-2xs">
              <thead>
                <tr className="border-b border-doc-ink font-mono text-3xs tracking-widest text-doc-ink-2 uppercase">
                  <th className="w-5 py-1.5 pr-2 text-left font-medium">#</th>
                  <th className="px-2 py-1.5 text-left font-medium">{t("Description")}</th>
                  <th className="px-2 py-1.5 text-right font-medium">{t("Quantity")}</th>
                  <th className="px-2 py-1.5 text-right font-medium">{t("Rate")}</th>
                  <th className="py-1.5 pl-2 text-right font-medium">{t("Amount")}</th>
                </tr>
              </thead>
              <tbody>
                {SAMPLE.charges.map((charge, index) => (
                  <tr key={charge.description} className="border-b border-doc-rule-soft">
                    <td className="py-1.5 pr-2 align-top text-doc-ink-3">{index + 1}</td>
                    <td className="px-2 py-1.5 align-top">{charge.description}</td>
                    <td className="px-2 py-1.5 text-right align-top whitespace-nowrap">
                      {charge.quantity}
                    </td>
                    <td className="px-2 py-1.5 text-right align-top">{charge.rate}</td>
                    <td className="py-1.5 pl-2 text-right align-top whitespace-nowrap">
                      {formatMinor(charge.amountMinor)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>

          <section className="grid grid-cols-[minmax(0,1fr)_11rem] items-start gap-6">
            <Marked
              on={
                active.has("name") ||
                active.has("address") ||
                active.has("instructions") ||
                active.has("contact")
              }
            >
              <div className="border border-doc-rule px-3.5 py-3">
                <Tag>{t("How to pay")}</Tag>
                <div className="font-semibold">{payee}</div>
                <div className="break-words text-doc-ink-2">{remitLines.join(", ")}</div>
                <p className="m-0 mt-1.5 text-2xs text-doc-ink-2">
                  {t(
                    "Write {0} on your payment so it is applied to this invoice.",
                    SAMPLE.invoiceNumber,
                  )}
                </p>
              </div>
            </Marked>
            <dl className="m-0 grid grid-cols-[1fr_auto] gap-x-4 gap-y-1">
              <dt className="text-doc-ink-2">{t("Subtotal")}</dt>
              <dd className="m-0 text-right">{formatMinor(SAMPLE_TOTAL_MINOR)}</dd>
              <dt className="mt-1 border-t border-doc-ink pt-2 text-sm font-semibold">
                {t("Total")}
              </dt>
              <dd className="m-0 mt-1 border-t border-doc-ink pt-2 text-right text-sm font-semibold">
                {formatMinor(SAMPLE_TOTAL_MINOR)}
              </dd>
              <dt className="font-semibold">{t("Balance due")}</dt>
              <dd className="m-0 text-right font-semibold">{formatMinor(SAMPLE_TOTAL_MINOR)}</dd>
            </dl>
          </section>

          <footer className="flex justify-between gap-4 border-t border-doc-rule-soft pt-2 font-mono text-3xs tracking-wide text-doc-ink-3">
            <span className="truncate">{issuer.name}</span>
            <span>{SAMPLE.invoiceNumber}</span>
          </footer>
        </article>
      </div>
    </aside>
  );
}
