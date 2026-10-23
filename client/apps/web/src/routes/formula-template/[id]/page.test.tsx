import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { FormulaTemplate } from "@trenova/shared/types/formula-template";
import type { ReactNode } from "react";
import { useFormContext, useFormState } from "react-hook-form";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { FormulaStudioEditPage } from "./page";

const mocks = vi.hoisted(() => ({
  get: vi.fn(),
  update: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: {
    formulaTemplateService: { get: mocks.get, update: mocks.update },
  },
}));

// The studio stands in as the one field a save reads and the reset writes back, so the test
// sees exactly what the editor would be handed after a save, beside the source section's notice.
vi.mock("../_components/studio/formula-studio", async () => {
  const { NewerVersionNotice } = await import("@/components/tariff-studio/newer-version");
  return {
    FormulaStudio: ({ onSave }: { onSave: () => void }) => {
      const { register } = useFormContext();
      const { isDirty } = useFormState();
      return (
        <>
          <NewerVersionNotice />
          <textarea aria-label="Tariff source" {...register("source")} />
          <output aria-label="Save state">{isDirty ? "Unsaved" : "Saved"}</output>
          <button type="button" onClick={onSave}>
            Save
          </button>
        </>
      );
    },
  };
});

vi.mock("@/components/tariff-studio/source-diff", () => ({
  SourceDiff: ({ before, after }: { before: string; after: string }) => (
    <div>
      <pre aria-label="Saved">{before}</pre>
      <pre aria-label="Yours">{after}</pre>
    </div>
  ),
}));

vi.mock("../_components/studio/dialogs/save-demote-dialog", () => ({
  SaveDemoteDialog: () => null,
}));

const ORIGINAL_SOURCE = 'tariff "Linehaul"';
const EDITED_SOURCE = 'tariff "Linehaul"\ncharge "Fuel" = 25';
const ELSEWHERE_SOURCE = 'tariff "Saved elsewhere"';
const NOTICE =
  "This was saved again after you started editing. Compare it with your edits before you save.";

function template(overrides: Partial<FormulaTemplate> = {}): FormulaTemplate {
  return {
    id: "ftmp_01",
    name: "Linehaul",
    description: "",
    source: ORIGINAL_SOURCE,
    status: "Draft",
    inputs: [],
    imports: [],
    tables: [],
    testCount: 0,
    version: 1,
    currentVersionNumber: 1,
    ...overrides,
  };
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}

function renderPage() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  function Wrapper({ children }: { children: ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={["/formula-templates/ftmp_01/edit"]}>
          <Routes>
            <Route path="/formula-templates/:id/edit" element={children} />
          </Routes>
        </MemoryRouter>
      </QueryClientProvider>
    );
  }
  render(<FormulaStudioEditPage />, { wrapper: Wrapper });
  return queryClient;
}

async function editSource(user: ReturnType<typeof userEvent.setup>) {
  const field = await screen.findByLabelText<HTMLTextAreaElement>("Tariff source");
  await user.clear(field);
  await user.type(field, EDITED_SOURCE);
  return field;
}

describe("FormulaStudioEditPage", () => {
  beforeEach(() => {
    mocks.get.mockReset();
    mocks.update.mockReset();
  });

  it("keeps the saved source while the refetch still holds the record from before the save", async () => {
    const refetch = deferred<FormulaTemplate>();
    mocks.get.mockResolvedValueOnce(template()).mockReturnValueOnce(refetch.promise);
    mocks.update.mockResolvedValue(template({ source: EDITED_SOURCE, version: 2 }));
    const user = userEvent.setup();

    const queryClient = renderPage();
    const field = await editSource(user);
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(mocks.get).toHaveBeenCalledTimes(2));
    expect(mocks.update).toHaveBeenCalledWith("ftmp_01", {
      description: "",
      source: EDITED_SOURCE,
      version: 1,
    });
    expect(field.value).toBe(EDITED_SOURCE);
    expect(screen.getByLabelText("Save state")).toHaveTextContent("Saved");

    refetch.resolve(template({ source: EDITED_SOURCE, version: 2 }));
    await waitFor(() => expect(queryClient.isFetching()).toBe(0));
    expect(field.value).toBe(EDITED_SOURCE);
    expect(screen.getByLabelText("Save state")).toHaveTextContent("Saved");
  });

  it("keeps what was typed while the save was in flight, as unsaved", async () => {
    const save = deferred<FormulaTemplate>();
    mocks.get
      .mockResolvedValueOnce(template())
      .mockResolvedValue(template({ source: EDITED_SOURCE, version: 2 }));
    mocks.update.mockReturnValue(save.promise);
    const user = userEvent.setup();

    const queryClient = renderPage();
    const field = await editSource(user);
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(mocks.update).toHaveBeenCalledTimes(1));
    await user.type(field, "!");

    save.resolve(template({ source: EDITED_SOURCE, version: 2 }));

    await waitFor(() => expect(mocks.get).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(queryClient.isFetching()).toBe(0));
    expect(field.value).toBe(`${EDITED_SOURCE}!`);
    expect(screen.getByLabelText("Save state")).toHaveTextContent("Unsaved");
  });

  it("sends the saved version with the next save", async () => {
    mocks.get
      .mockResolvedValueOnce(template())
      .mockReturnValue(new Promise<FormulaTemplate>(() => {}));
    mocks.update
      .mockResolvedValueOnce(template({ source: EDITED_SOURCE, version: 2 }))
      .mockResolvedValueOnce(template({ source: `${EDITED_SOURCE}!`, version: 3 }));
    const user = userEvent.setup();

    renderPage();
    const field = await editSource(user);
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() => expect(mocks.get).toHaveBeenCalledTimes(2));
    await user.type(field, "!");
    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(mocks.update).toHaveBeenCalledTimes(2));
    expect(mocks.update).toHaveBeenLastCalledWith(
      "ftmp_01",
      expect.objectContaining({ source: `${EDITED_SOURCE}!`, version: 2 }),
    );
  });

  it("reseats a form without edits when the server holds a newer version", async () => {
    mocks.get
      .mockResolvedValueOnce(template())
      .mockResolvedValueOnce(template({ source: ELSEWHERE_SOURCE, version: 3 }));

    const queryClient = renderPage();
    const field = await screen.findByLabelText<HTMLTextAreaElement>("Tariff source");
    expect(field.value).toBe(ORIGINAL_SOURCE);

    await queryClient.invalidateQueries();

    await waitFor(() => expect(field.value).toBe(ELSEWHERE_SOURCE));
    expect(screen.queryByText(NOTICE)).not.toBeInTheDocument();
  });

  it("holds a newer version back from unsaved edits and says so", async () => {
    mocks.get
      .mockResolvedValueOnce(template())
      .mockResolvedValue(template({ source: ELSEWHERE_SOURCE, version: 3 }));
    const user = userEvent.setup();

    const queryClient = renderPage();
    const field = await editSource(user);
    await queryClient.invalidateQueries();

    expect(await screen.findByText(NOTICE)).toBeInTheDocument();
    expect(field.value).toBe(EDITED_SOURCE);
    expect(screen.getByLabelText("Save state")).toHaveTextContent("Unsaved");
  });

  it("opens the comparison instead of saving over a held version", async () => {
    mocks.get
      .mockResolvedValueOnce(template())
      .mockResolvedValue(template({ source: ELSEWHERE_SOURCE, version: 3 }));
    const user = userEvent.setup();

    const queryClient = renderPage();
    await editSource(user);
    await queryClient.invalidateQueries();
    await screen.findByText(NOTICE);
    await user.click(screen.getByRole("button", { name: "Save" }));

    const dialog = await screen.findByRole("dialog", { name: "Compare with the saved version" });
    expect(mocks.update).not.toHaveBeenCalled();
    expect(within(dialog).getByLabelText("Saved")).toHaveTextContent(ELSEWHERE_SOURCE);
    expect(within(dialog).getByLabelText("Yours").textContent).toBe(EDITED_SOURCE);
  });

  it("loads the held version over the edits", async () => {
    mocks.get
      .mockResolvedValueOnce(template())
      .mockResolvedValue(template({ source: ELSEWHERE_SOURCE, version: 3 }));
    const user = userEvent.setup();

    const queryClient = renderPage();
    const field = await editSource(user);
    await queryClient.invalidateQueries();
    await user.click(await screen.findByRole("button", { name: "Compare" }));
    await user.click(await screen.findByRole("button", { name: "Load the saved version" }));

    await waitFor(() => expect(field.value).toBe(ELSEWHERE_SOURCE));
    expect(screen.getByLabelText("Save state")).toHaveTextContent("Saved");
    expect(screen.queryByText(NOTICE)).not.toBeInTheDocument();
  });

  it("keeps the edits over the held version and saves them against it", async () => {
    mocks.get
      .mockResolvedValueOnce(template())
      .mockResolvedValue(template({ name: "Linehaul 2026", source: ELSEWHERE_SOURCE, version: 3 }));
    mocks.update.mockResolvedValue(
      template({ name: "Linehaul 2026", source: EDITED_SOURCE, version: 4 }),
    );
    const user = userEvent.setup();

    const queryClient = renderPage();
    const field = await editSource(user);
    await queryClient.invalidateQueries();
    await user.click(await screen.findByRole("button", { name: "Compare" }));
    await user.click(await screen.findByRole("button", { name: "Keep my edits" }));

    await waitFor(() => expect(screen.queryByText(NOTICE)).not.toBeInTheDocument());
    expect(field.value).toBe(EDITED_SOURCE);
    expect(screen.getByLabelText("Save state")).toHaveTextContent("Unsaved");

    await user.click(screen.getByRole("button", { name: "Save" }));

    await waitFor(() => expect(mocks.update).toHaveBeenCalledTimes(1));
    expect(mocks.update).toHaveBeenCalledWith("ftmp_01", {
      description: "",
      source: EDITED_SOURCE,
      version: 3,
    });
  });
});
