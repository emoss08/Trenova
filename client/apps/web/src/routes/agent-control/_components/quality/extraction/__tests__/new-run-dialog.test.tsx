import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const startRun = vi.fn();

vi.mock("@/lib/graphql/extraction-eval", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/graphql/extraction-eval")>();
  return { ...actual, startExtractionEvalRun: (input: unknown) => startRun(input) };
});

vi.mock("@/lib/graphql/ai-provider", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/graphql/ai-provider")>();
  return {
    ...actual,
    fetchAIProviders: () =>
      Promise.resolve([
        {
          id: "aip_frontier",
          name: "Frontier",
          model: "frontier-large",
          enabled: true,
          tasks: ["DocumentExtraction"],
        },
      ]),
  };
});

const { NewRunDialog } = await import("../new-run-dialog");

beforeEach(() => {
  startRun.mockReset();
  startRun.mockResolvedValue({ id: "eer_1" });
});

afterEach(() => cleanup());

function renderDialog() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <NewRunDialog open onOpenChange={() => undefined} activeCases={12} onStarted={vi.fn()} />
    </QueryClientProvider>,
  );
}

describe("NewRunDialog", () => {
  it("starts a run on the chosen provider over the case limit", async () => {
    renderDialog();
    const start = await screen.findByRole("button", { name: "Start run" });
    await waitFor(() => expect(start).toBeEnabled());

    await userEvent.click(start);

    await waitFor(() => expect(startRun).toHaveBeenCalledTimes(1));
    expect(startRun.mock.calls[0]).toStrictEqual([{ providerId: "aip_frontier", caseLimit: 50 }]);
  });
});
