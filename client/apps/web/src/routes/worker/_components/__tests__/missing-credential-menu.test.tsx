import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { workerTableGraphQLConfig } from "@/lib/graphql/worker-table";
import { MissingCredentialMenu } from "../missing-credential-menu";

const { fetchWorkerCredentialTypeChoices } = vi.hoisted(() => ({
  fetchWorkerCredentialTypeChoices: vi.fn(),
}));

vi.mock("@/lib/graphql/worker-credential", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/worker-credential")>()),
  fetchWorkerCredentialTypeChoices,
}));

const choices = [
  { id: "wct_road", code: "ROAD_TEST", name: "Road Test Certificate", isRequired: true },
  { id: "wct_fork", code: "FORKLIFT", name: "Forklift Certification", isRequired: false },
];

function renderMenu(value: string | null) {
  const onChange = vi.fn();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MissingCredentialMenu value={value} onChange={onChange} />
    </QueryClientProvider>,
  );

  return onChange;
}

afterEach(() => {
  cleanup();
  fetchWorkerCredentialTypeChoices.mockReset();
});

describe("MissingCredentialMenu", () => {
  it("offers every tracked credential type and narrows the roster to the one chosen", async () => {
    fetchWorkerCredentialTypeChoices.mockResolvedValue(choices);
    const onChange = renderMenu(null);

    fireEvent.click(screen.getByRole("button", { name: "Missing credential" }));
    const options = await screen.findAllByRole("radio");
    expect(options.map((option) => option.textContent)).toEqual([
      "Road Test CertificateRequired",
      "Forklift Certification",
    ]);

    fireEvent.click(screen.getByRole("radio", { name: /Road Test Certificate/ }));
    expect(onChange).toHaveBeenCalledWith("ROAD_TEST");
  });

  it("names the type chosen and clears it when chosen again", async () => {
    fetchWorkerCredentialTypeChoices.mockResolvedValue(choices);
    const onChange = renderMenu("ROAD_TEST");

    const trigger = await screen.findByRole("button", { name: "Missing credential" });
    expect(await screen.findByText("Missing Road Test Certificate")).toBeTruthy();
    expect(trigger.getAttribute("data-active")).toBe("true");

    fireEvent.click(trigger);
    const chosen = await screen.findByRole("radio", { name: /Road Test Certificate/ });
    expect(chosen.getAttribute("aria-checked")).toBe("true");
    fireEvent.click(chosen);
    expect(onChange).toHaveBeenCalledWith(null);
  });

  it("says when no credential types are tracked", async () => {
    fetchWorkerCredentialTypeChoices.mockResolvedValue([]);
    renderMenu(null);

    fireEvent.click(screen.getByRole("button", { name: "Missing credential" }));
    expect(await screen.findByText("No credential types are tracked yet.")).toBeTruthy();
  });
});

describe("workerTableGraphQLConfig", () => {
  it("sends the credential code beside the table input only when one is chosen", () => {
    expect(workerTableGraphQLConfig("ROAD_TEST").extraVariables).toEqual({
      withoutCredential: "ROAD_TEST",
    });
    expect(workerTableGraphQLConfig(null).extraVariables).toBeUndefined();
  });
});
