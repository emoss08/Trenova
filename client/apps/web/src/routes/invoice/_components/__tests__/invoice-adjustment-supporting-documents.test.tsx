import { cleanup, render, screen } from "@testing-library/react";
import { useForm } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { InvoiceAdjustmentSupportingDocumentsSection } from "../invoice-adjustment-dialog-sections";

const mocks = vi.hoisted(() => ({ documentField: vi.fn() }));

vi.mock("@/components/autocomplete-fields", () => ({
  DocumentMultiSelectAutocompleteField: (props: {
    label: string;
    description: string;
    rules?: { required?: boolean };
  }) => {
    mocks.documentField(props);
    return (
      <div>
        <span>{props.label}</span>
        <span>{props.description}</span>
      </div>
    );
  },
}));

vi.mock("@/components/document-upload-section", () => ({
  DocumentUploadSection: () => null,
}));

function Harness() {
  const form = useForm<{ reason: string; referencedDocumentIds: string[] }>({
    defaultValues: { reason: "", referencedDocumentIds: [] },
  });
  return (
    <InvoiceAdjustmentSupportingDocumentsSection
      control={form.control as never}
      shipmentId="shp_1"
      draft={null}
    />
  );
}

afterEach(() => {
  cleanup();
  mocks.documentField.mockReset();
});

describe("InvoiceAdjustmentSupportingDocumentsSection", () => {
  it("never requires supporting documents", () => {
    render(<Harness />);

    expect(screen.getByText("Supporting documents (optional)")).toBeInTheDocument();
    expect(screen.queryByText(/Required by policy/)).not.toBeInTheDocument();
    const props = mocks.documentField.mock.calls.at(-1)?.[0];
    expect(props?.rules?.required).toBeFalsy();
    expect(props?.rules).toBeUndefined();
  });
});
