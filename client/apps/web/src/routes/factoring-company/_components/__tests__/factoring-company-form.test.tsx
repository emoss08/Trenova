import type { FactoringCompany } from "@/types/factoring-company";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { FormProvider, useForm } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { FactoringCompanyForm, type InvoiceAttention } from "../factoring-company-form";

vi.mock("@/components/autocomplete-fields", () => ({
  UsStateAutocompleteField: () => null,
}));

afterEach(cleanup);

const FACTOR: FactoringCompany = {
  status: "Active",
  code: "BLUE",
  name: "Bluewater Receivables Finance",
  addressLine1: "PO Box 29237",
  addressLine2: "",
  city: "Jacksonville",
  stateId: "us_fl",
  postalCode: "32202",
  email: "",
  phoneNumber: "",
  remittanceInstructions: "",
  noticeOfAssignment: "",
  notes: "",
  nonRecourse: false,
};

function Harness({
  remitNotices,
  onAttention,
}: {
  remitNotices: number;
  onAttention?: InvoiceAttention;
}) {
  const form = useForm<FactoringCompany>({ defaultValues: FACTOR });
  return (
    <FormProvider {...form}>
      <FactoringCompanyForm remitNotices={remitNotices} onAttention={onAttention} />
    </FormProvider>
  );
}

const WARNING = /a notice naming the current remit-to/;

describe("FactoringCompanyForm", () => {
  it("warns that customers hold a notice naming the remit-to once it changes", () => {
    render(<Harness remitNotices={18} />);
    expect(screen.queryByText(WARNING)).toBeNull();

    fireEvent.change(screen.getByLabelText("Address line 2"), {
      target: { value: "Lockbox 4471" },
    });

    expect(
      screen.getByText(
        "18 customers have a notice naming the current remit-to. Invoices print the new one once you save, so send them an updated notice.",
      ),
    ).toBeTruthy();
  });

  it("says nothing about notices when no customer holds one", () => {
    render(<Harness remitNotices={0} />);

    fireEvent.change(screen.getByLabelText("Address line 2"), {
      target: { value: "Lockbox 4471" },
    });

    expect(screen.queryByText(WARNING)).toBeNull();
  });

  it("does not warn for a change that never prints, like the notes", () => {
    render(<Harness remitNotices={18} />);

    fireEvent.change(screen.getByRole("textbox", { name: "Notes" }), {
      target: { value: "Call before the 15th" },
    });

    expect(screen.queryByText(WARNING)).toBeNull();
  });

  it("puts an info icon beside each section's title in place of a printed-where label", () => {
    render(<Harness remitNotices={0} />);

    for (const title of [
      "Identity",
      "Remit-to address",
      "Payment instructions and notice",
      "Contact",
      "Recourse",
      "Notes",
    ]) {
      expect(screen.getByRole("img", { name: `About ${title}` })).toBeTruthy();
    }
    expect(screen.queryByText("Prints as the payee")).toBeNull();
  });

  it("tells the preview which parts of the invoice a section sets while it has focus", () => {
    const onAttention = vi.fn();
    render(<Harness remitNotices={0} onAttention={onAttention} />);

    fireEvent.focus(screen.getByLabelText("Remittance instructions"));
    expect(onAttention).toHaveBeenLastCalledWith("focus", ["instructions", "notice"]);

    fireEvent.focus(screen.getByLabelText("Email"));
    expect(onAttention).toHaveBeenLastCalledWith("focus", ["contact"]);
  });
});
