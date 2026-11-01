import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { useForm } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CustomFieldsSection } from "../custom-fields-section";

const { formFields } = vi.hoisted(() => ({ formFields: vi.fn() }));

vi.mock("@trenova/shared/lib/graphql", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@trenova/shared/lib/graphql")>();
  return {
    ...actual,
    requestGraphQL: (request: Parameters<typeof actual.requestGraphQL>[0]) =>
      request.operationName === "CustomFieldFormFields"
        ? Promise.resolve({
            customFieldFormFields: formFields(
              (request.variables as { resourceType: string }).resourceType,
            ),
          })
        : actual.requestGraphQL(request),
  };
});

function field(id: string, label: string, displayOrder: number) {
  return {
    id,
    name: label.toLowerCase().replaceAll(" ", "_"),
    label,
    description: null,
    fieldType: "text",
    isRequired: false,
    displayOrder,
    color: null,
    options: [],
    validationRules: null,
    defaultValue: null,
    uiAttributes: null,
  };
}

function Wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

function WorkerFields() {
  const { control } = useForm();
  return <CustomFieldsSection resourceType="worker" control={control} />;
}

describe("CustomFieldsSection", () => {
  afterEach(cleanup);

  it("draws the form fields the server returns for the record's resource type, in its order", async () => {
    formFields.mockReturnValue([
      field("cfd_2", "Badge number", 1),
      field("cfd_1", "Shirt size", 2),
    ]);

    render(<WorkerFields />, { wrapper: Wrapper });

    expect(await screen.findByText("Custom fields")).toBeInTheDocument();
    const labels = screen.getAllByText(/Badge number|Shirt size/).map((node) => node.textContent);
    expect(labels).toEqual(["Badge number", "Shirt size"]);
    expect(formFields).toHaveBeenCalledWith("worker");
  });

  it("draws nothing when the resource type has no custom fields", async () => {
    formFields.mockReturnValue([]);

    const { container } = render(<WorkerFields />, { wrapper: Wrapper });

    await vi.waitFor(() => expect(formFields).toHaveBeenCalled());
    expect(container).toBeEmptyDOMElement();
  });
});
