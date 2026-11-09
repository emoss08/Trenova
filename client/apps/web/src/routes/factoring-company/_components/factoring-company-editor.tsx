import type { FactoringCompany } from "@/types/factoring-company";
import { useState, type ReactNode } from "react";
import type { Control } from "react-hook-form";
import { FactoringCompanyForm, type InvoiceAttention } from "./factoring-company-form";
import {
  FactoringInvoicePreview,
  type FactorState,
  type InvoiceRegion,
} from "./factoring-invoice-preview";

/**
 * The factor's details beside the invoice they print. Pointing at or editing a section lights the
 * part of the invoice it sets, so a person sees where each field lands before saving. The create
 * page and the factor's own page share it; the create page brings a footer.
 */
export function FactoringCompanyEditor({
  control,
  state,
  onState,
  readOnly = false,
  recourseFromAgreement = false,
  onOpenAgreement,
  remitNotices,
  onOpenCustomers,
  footer,
}: {
  control: Control<FactoringCompany>;
  state: FactorState;
  onState: (state: FactorState) => void;
  readOnly?: boolean;
  recourseFromAgreement?: boolean;
  onOpenAgreement?: () => void;
  remitNotices?: number;
  onOpenCustomers?: () => void;
  footer?: ReactNode;
}) {
  const [hovered, setHovered] = useState<readonly InvoiceRegion[] | null>(null);
  const [focused, setFocused] = useState<readonly InvoiceRegion[] | null>(null);
  const active = new Set<InvoiceRegion>(hovered ?? focused ?? []);
  const attend: InvoiceAttention = (kind, regions) =>
    kind === "hover" ? setHovered(regions) : setFocused(regions);

  return (
    <div className="flex min-w-0 flex-1 flex-col">
      <div className="grid flex-1 items-start gap-6 xl:grid-cols-[minmax(0,40rem)_minmax(0,1fr)] xl:items-stretch">
        <fieldset disabled={readOnly} className="min-w-0">
          <FactoringCompanyForm
            recourseFromAgreement={recourseFromAgreement}
            onOpenAgreement={onOpenAgreement}
            onState={onState}
            onAttention={attend}
            remitNotices={remitNotices}
            onOpenCustomers={onOpenCustomers}
          />
        </fieldset>
        <div className="flex min-w-0 flex-col">
          <FactoringInvoicePreview control={control} state={state} active={active} />
        </div>
      </div>
      {footer}
    </div>
  );
}
