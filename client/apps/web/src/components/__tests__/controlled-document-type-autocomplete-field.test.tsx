import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import { useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ControlledDocumentTypeAutocompleteField } from "../autocomplete-fields";

// The capture screens pick a document type outside a form. The options come
// from the DOCUMENT_TYPE select-option resource; the resolver
// (documentTypeSelectOptionItem) sends the type's code as the label and its
// name and color in meta.

const CURSOR = "eyJpZCI6ImR0XzEifQ";

const BILL_OF_LADING = {
  id: "dt_01bol",
  label: "BOL",
  description: "Signed at pickup",
  meta: { code: "BOL", color: "#2563eb", name: "Bill of lading" },
};
const PROOF_OF_DELIVERY = {
  id: "dt_01pod",
  label: "POD",
  description: null,
  meta: { code: "POD", color: null, name: "Proof of delivery" },
};

function selectOptionsResponse(nodes: (typeof BILL_OF_LADING | typeof PROOF_OF_DELIVERY)[]) {
  return new Response(
    JSON.stringify({
      data: {
        selectOptions: {
          edges: nodes.map((node) => ({ cursor: CURSOR, node })),
          pageInfo: { hasNextPage: false, endCursor: CURSOR },
          totalCount: nodes.length,
        },
      },
    }),
    { status: 200, headers: { "Content-Type": "application/json" } },
  );
}

function Harness() {
  const [value, setValue] = useState("");
  return (
    <>
      <ControlledDocumentTypeAutocompleteField
        label="Document type"
        placeholder="Optional"
        value={value}
        onValueChange={setValue}
      />
      <output aria-label="document type value">{value}</output>
    </>
  );
}

function renderField() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <Harness />
    </QueryClientProvider>,
  );
}

describe("ControlledDocumentTypeAutocompleteField", () => {
  afterEach(() => {
    clearCsrfToken();
    cleanup();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("lists the organization's document types and stores the chosen one's id", async () => {
    setCsrfToken("graphql-token");
    const user = userEvent.setup();
    const fetchMock = vi.fn(async () => selectOptionsResponse([BILL_OF_LADING, PROOF_OF_DELIVERY]));
    vi.stubGlobal("fetch", fetchMock);

    renderField();
    await user.click(screen.getByRole("combobox"));

    expect(await screen.findByText("Bill of lading")).toBeInTheDocument();
    expect(screen.getByText("Proof of delivery")).toBeInTheDocument();

    await user.click(screen.getByText("Proof of delivery"));
    await waitFor(() => {
      expect(screen.getByLabelText("document type value")).toHaveTextContent("dt_01pod");
    });
  });

  it("asks the DOCUMENT_TYPE select-option resource, never the REST route", async () => {
    setCsrfToken("graphql-token");
    const user = userEvent.setup();
    const fetchMock = vi.fn(async () => selectOptionsResponse([BILL_OF_LADING]));
    vi.stubGlobal("fetch", fetchMock);

    renderField();
    await user.click(screen.getByRole("combobox"));
    await screen.findByText("Bill of lading");

    expect(fetchMock).toHaveBeenCalled();
    for (const [input, init] of fetchMock.mock.calls as unknown as [RequestInfo, RequestInit][]) {
      expect(String(input)).not.toContain("/document-types/select-options/");
      expect(JSON.parse(init.body as string)).toMatchObject({
        operationName: "SelectOptions",
        variables: { input: { resource: "DOCUMENT_TYPE" } },
      });
    }
  });
});
