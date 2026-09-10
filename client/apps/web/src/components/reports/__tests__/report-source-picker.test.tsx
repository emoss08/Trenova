import { ReportSourcePicker, type ReportSource } from "@/components/reports/report-source-picker";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  useReportDefinitionOptions: vi.fn(),
  useCannedReports: vi.fn(),
  useReportDefinition: vi.fn(),
}));

vi.mock("@/hooks/use-reports", () => ({
  useReportDefinitionOptions: mocks.useReportDefinitionOptions,
  useCannedReports: mocks.useCannedReports,
  useReportDefinition: mocks.useReportDefinition,
}));

type Definition = {
  id: string;
  name: string;
  description: string;
  category: string;
  status: string;
};

function definition(id: string, name: string): Definition {
  return { id, name, description: "", category: "Operations", status: "published" };
}

const fetchNextPage = vi.fn();

function stubDefinitions(
  rows: Definition[],
  overrides: { hasNextPage?: boolean; isLoading?: boolean } = {},
) {
  mocks.useReportDefinitionOptions.mockImplementation((_search: string, enabled = true) => ({
    data: enabled ? rows : undefined,
    isLoading: overrides.isLoading ?? false,
    error: null,
    hasNextPage: overrides.hasNextPage ?? false,
    isFetchingNextPage: false,
    fetchNextPage,
    refetch: vi.fn(),
  }));
}

function stubCanned(rows: { key: string; name: string; description: string; category: string }[]) {
  mocks.useCannedReports.mockImplementation((enabled = true) => ({
    data: enabled ? rows : undefined,
    isLoading: false,
    error: null,
    refetch: vi.fn(),
  }));
}

const NO_SOURCE: ReportSource = { definitionId: null, cannedKey: null };

function renderPicker(value: ReportSource = NO_SOURCE) {
  const onChange = vi.fn();
  render(<ReportSourcePicker value={value} onChange={onChange} />);
  return { onChange };
}

beforeEach(() => {
  vi.clearAllMocks();
  stubDefinitions([definition("rdef_1", "Aging Detail"), definition("rdef_2", "Backhaul Margin")]);
  stubCanned([
    {
      key: "on_time",
      name: "On-Time Service",
      description: "Service against target",
      category: "Operations",
    },
    {
      key: "ar_aging",
      name: "AR Aging",
      description: "Open receivables by bucket",
      category: "Finance",
    },
  ]);
  mocks.useReportDefinition.mockReturnValue({ data: undefined, isLoading: false });
});

describe("ReportSourcePicker", () => {
  it("hands what was typed to the query that reads the library", async () => {
    const user = userEvent.setup();
    renderPicker();

    await user.type(screen.getByLabelText("Search saved reports"), "margin");

    await waitFor(() =>
      expect(mocks.useReportDefinitionOptions).toHaveBeenCalledWith("margin", true),
    );
  });

  it("debounces so a five-letter word is not five round trips", async () => {
    const user = userEvent.setup();
    renderPicker();

    await user.type(screen.getByLabelText("Search saved reports"), "aging");

    const searched = mocks.useReportDefinitionOptions.mock.calls
      .map(([term]) => term as string)
      .filter((term) => term !== "");

    expect(searched).not.toContain("a");
    expect(searched).not.toContain("ag");
  });

  it("asks for the next page when the list is scrolled to the bottom", async () => {
    stubDefinitions([definition("rdef_1", "Aging Detail")], { hasNextPage: true });
    renderPicker();

    const list = screen.getByRole("listbox", { name: "Saved reports" });
    Object.defineProperty(list, "scrollHeight", { value: 400, configurable: true });
    Object.defineProperty(list, "clientHeight", { value: 200, configurable: true });
    list.scrollTop = 190;
    fireEvent.scroll(list);

    expect(fetchNextPage).toHaveBeenCalledTimes(1);
  });

  it("leaves the next page alone while the list is still near the top", () => {
    stubDefinitions([definition("rdef_1", "Aging Detail")], { hasNextPage: true });
    renderPicker();

    const list = screen.getByRole("listbox", { name: "Saved reports" });
    Object.defineProperty(list, "scrollHeight", { value: 400, configurable: true });
    Object.defineProperty(list, "clientHeight", { value: 200, configurable: true });
    list.scrollTop = 0;
    fireEvent.scroll(list);

    expect(fetchNextPage).not.toHaveBeenCalled();
  });

  // The library is paged, so the chosen report is usually not on the page in
  // front of the user. Without this the picker looks like nothing is chosen.
  it("shows a report chosen on an earlier page as chosen", async () => {
    mocks.useReportDefinition.mockReturnValue({
      data: definition("rdef_99", "Detention Recovery"),
      isLoading: false,
    });
    renderPicker({ definitionId: "rdef_99", cannedKey: null });

    const chosen = await screen.findByRole("option", { name: /Detention Recovery/ });
    expect(chosen).toHaveAttribute("aria-selected", "true");
    expect(mocks.useReportDefinition).toHaveBeenCalledWith("rdef_99");
  });

  // Reading the chosen report a second time when it is already in front of the
  // user is a request per open of the dialog, for nothing.
  it("does not re-read a chosen report the loaded page already carries", async () => {
    renderPicker({ definitionId: "rdef_1", cannedKey: null });

    expect(await screen.findByRole("option", { name: /Aging Detail/ })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(mocks.useReportDefinition).not.toHaveBeenCalledWith("rdef_1");
    expect(mocks.useReportDefinition).toHaveBeenCalledWith(undefined);
  });

  it("does not repeat the chosen report when the page already carries it", async () => {
    mocks.useReportDefinition.mockReturnValue({
      data: definition("rdef_1", "Aging Detail"),
      isLoading: false,
    });
    renderPicker({ definitionId: "rdef_1", cannedKey: null });

    expect(await screen.findAllByRole("option", { name: /Aging Detail/ })).toHaveLength(1);
  });

  // The server rejects a config carrying both a saved id and a gallery key, so
  // choosing one has to clear the other.
  it("clears the gallery key when a saved report is chosen", async () => {
    const user = userEvent.setup();
    const { onChange } = renderPicker({ definitionId: null, cannedKey: "on_time" });

    await user.click(screen.getByRole("radio", { name: "Saved reports" }));
    await user.click(await screen.findByRole("option", { name: /Aging Detail/ }));

    expect(onChange).toHaveBeenCalledWith({ definitionId: "rdef_1", cannedKey: null });
  });

  it("clears the saved id when a gallery report is chosen", async () => {
    const user = userEvent.setup();
    const { onChange } = renderPicker({ definitionId: "rdef_1", cannedKey: null });

    await user.click(screen.getByRole("radio", { name: "Report gallery" }));
    await user.click(await screen.findByRole("option", { name: /On-Time Service/ }));

    expect(onChange).toHaveBeenCalledWith({ definitionId: null, cannedKey: "on_time" });
  });

  it("opens on the gallery when that is where the current report came from", async () => {
    renderPicker({ definitionId: null, cannedKey: "ar_aging" });

    expect(screen.getByRole("radio", { name: "Report gallery" })).toHaveAttribute(
      "aria-checked",
      "true",
    );
    expect(await screen.findByRole("option", { name: /AR Aging/ })).toHaveAttribute(
      "aria-selected",
      "true",
    );
  });

  it("filters the gallery, which is a fixed catalog, without a round trip", async () => {
    const user = userEvent.setup();
    renderPicker({ definitionId: null, cannedKey: null });

    await user.click(screen.getByRole("radio", { name: "Report gallery" }));
    await user.type(screen.getByLabelText("Search the report gallery"), "receivables");

    await waitFor(() => expect(screen.queryByRole("option", { name: /On-Time/ })).toBeNull());
    expect(screen.getByRole("option", { name: /AR Aging/ })).toBeInTheDocument();
  });

  it("says nothing matches rather than showing an empty box", async () => {
    stubDefinitions([]);
    const user = userEvent.setup();
    renderPicker();

    await user.type(screen.getByLabelText("Search saved reports"), "zzz");

    expect(await screen.findByText("Nothing matches")).toBeInTheDocument();
  });

  it("offers a retry when the library cannot be read", () => {
    mocks.useReportDefinitionOptions.mockReturnValue({
      data: undefined,
      isLoading: false,
      error: new Error("boom"),
      hasNextPage: false,
      isFetchingNextPage: false,
      fetchNextPage,
      refetch: vi.fn(),
    });
    renderPicker();

    expect(screen.getByText("Could not load reports.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });
});
