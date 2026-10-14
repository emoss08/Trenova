import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import { useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ControlledEDITransferAutocompleteField } from "../autocomplete-fields";

const CURSOR = "eyJpZCI6ImVkaWx0XzEifQ";

type TransferOption = {
  id: string;
  label: string;
  description: string;
  meta: Record<string, string>;
};

// Shaped like ediTransferSelectOption on the server: the label is the bill of
// lading, and meta names the partner the reading organization keeps for the
// other side, under `partner`, whichever way the tender went.
const FROM_LONE_STAR: TransferOption = {
  id: "edilt_1",
  label: "BOL-55120",
  description: "",
  meta: {
    status: "PendingApproval",
    bol: "BOL-55120",
    customerLabel: "",
    partner: "Lone Star Logistics",
  },
};

const FOR_ACME: TransferOption = {
  id: "edilt_2",
  label: "LT-88241",
  description: "Acme Foods",
  meta: {
    status: "MappingRequired",
    bol: "LT-88241",
    customerLabel: "Acme Foods",
    partner: "Acme",
  },
};

const UNNAMED: TransferOption = {
  id: "edilt_3",
  label: "Load tender edilt_3",
  description: "",
  meta: { status: "Submitted", bol: "", customerLabel: "" },
};

function selectOptionsResponse(nodes: TransferOption[]) {
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
  return <ControlledEDITransferAutocompleteField value={value} onValueChange={setValue} />;
}

describe("ControlledEDITransferAutocompleteField", () => {
  afterEach(() => {
    clearCsrfToken();
    cleanup();
    vi.unstubAllGlobals();
  });

  it("names the customer under each tender, else the partner, else its status", async () => {
    setCsrfToken("graphql-token");
    const user = userEvent.setup();
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => selectOptionsResponse([FROM_LONE_STAR, FOR_ACME, UNNAMED])),
    );

    render(
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <Harness />
      </QueryClientProvider>,
    );
    await user.click(screen.getByRole("combobox"));

    const detail = (label: string) => screen.getByText(label).nextElementSibling?.textContent;
    await screen.findByText("BOL-55120");
    expect(detail("BOL-55120")).toBe("Lone Star Logistics");
    expect(detail("LT-88241")).toBe("Acme Foods");
    expect(detail("Load tender edilt_3")).toBe("Submitted");
  });
});
