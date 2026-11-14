import { InvoiceTag, SampleInvoice } from "@/components/documents/sample-invoice";
import { INVOICE_LOGO_CLASS, SAMPLE_INVOICE } from "@/components/documents/sample-invoice-data";
import { LazyImage } from "@/components/image";
import { useCenteredStickyTop } from "@/hooks/use-centered-sticky-top";
import { useOrganizationBranding } from "@/hooks/use-organization-branding";
import { usePermission } from "@/hooks/use-permission";
import { factorRemitLines, standardNoticeOfAssignment } from "@/lib/factoring";
import { organization } from "@/lib/queries/organization";
import { userOrganization } from "@/lib/queries/user";
import type { FactoringCompany } from "@/types/factoring-company";
import { useQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cityStatePostal, mailingAddressLines } from "@trenova/shared/lib/mailing-address";
import { cn } from "@trenova/shared/lib/utils";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { ReactNode } from "react";
import { useWatch, type Control } from "react-hook-form";

/** The parts of the invoice this page's details print, so a section can point at its own. */
export type InvoiceRegion = "name" | "address" | "contact" | "instructions" | "notice";

export type FactorState = { abbreviation: string | null; countryName: string | null };

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
  const { branding, refresh: refreshBranding } = useOrganizationBranding(
    current?.id,
    Boolean(current?.logoUrl),
  );
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
    logo: branding?.logoUrl || null,
    refreshBranding,
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

  return (
    <aside aria-label={t("Invoice preview")} className="flex min-w-0 flex-1 flex-col gap-2.5">
      <div className="flex-1 rounded-lg border bg-sunken p-3 sm:p-5">
        <SampleInvoice
          ref={paperRef}
          className="mx-auto max-w-176 xl:sticky"
          issuer={{
            name: issuer.name,
            lines: [issuer.lines.join(" · ")],
            logo: issuer.logo ? (
              <LazyImage
                src={issuer.logo}
                alt={t("{0} logo", issuer.name)}
                className={INVOICE_LOGO_CLASS}
                onError={() => issuer.logo && issuer.refreshBranding(issuer.logo)}
              />
            ) : null,
          }}
          remitTo={
            <>
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
            </>
          }
          notice={
            <Marked on={active.has("notice") || active.has("name")}>
              <section className="border-[1.5px] border-doc-ink px-3.5 py-2.5">
                <InvoiceTag>{t("Notice of assignment")}</InvoiceTag>
                <p className="m-0 font-medium whitespace-pre-line">{printedNotice}</p>
              </section>
            </Marked>
          }
          howToPay={
            <Marked
              on={
                active.has("name") ||
                active.has("address") ||
                active.has("instructions") ||
                active.has("contact")
              }
            >
              <div className="border border-doc-rule px-3.5 py-3">
                <InvoiceTag>{t("How to pay")}</InvoiceTag>
                <div className="font-semibold">{payee}</div>
                <div className="break-words text-doc-ink-2">{remitLines.join(", ")}</div>
                <p className="m-0 mt-1.5 text-2xs text-doc-ink-2">
                  {t(
                    "Write {0} on your payment so it is applied to this invoice.",
                    SAMPLE_INVOICE.invoiceNumber,
                  )}
                </p>
              </div>
            </Marked>
          }
        />
      </div>
    </aside>
  );
}
