import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import {
  formulaPreviewResponseSchema,
  formulaTemplateSchema,
  type FormulaPreviewRequest,
  type FormulaTemplate,
} from "@trenova/shared/types/formula-template";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DeskFormulaTemplate } from "../desk-formula-template";

const service = vi.hoisted(() => ({
  get: vi.fn(),
  getContext: vi.fn(),
  getUsage: vi.fn(),
  listVersions: vi.fn(),
  readiness: vi.fn(),
  preview: vi.fn(),
}));
const access = vi.hoisted(() => ({ write: true }));

vi.mock("@/services/api", () => ({ apiService: { formulaTemplateService: service } }));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: access.write, isLoading: false }),
}));

const SOURCE = `tariff "Reefer per mile"

/// Paid per loaded mile.
input rate: Money/mi = $2.85/mi
input reeferFee: Money

charge "Linehaul" (quantity: shipment.distance, rate: rate)

if shipment.reefer {
  charge "Reefer unit" as REEF = reeferFee
}

minimum $250
`;

/** A template as the server's GET /formula-templates/{id}/ returns it. */
function template(overrides: Partial<FormulaTemplate> = {}): FormulaTemplate {
  return {
    ...formulaTemplateSchema.parse({
      id: "ft_1",
      organizationId: "org_1",
      businessUnitId: "bu_1",
      name: "Reefer per mile",
      description: "Linehaul for reefer loads",
      source: SOURCE,
      status: "Active",
      inputs: [
        {
          name: "rate",
          doc: "Paid per loaded mile.",
          type: "Money/mi",
          descriptor: { kind: "quantity", dimension: "Money/Distance", unit: "$/mi" },
          required: false,
          default: { $q: "2.85", u: "$/mi" },
        },
        {
          name: "reeferFee",
          doc: null,
          type: "Money",
          descriptor: { kind: "money" },
          required: true,
        },
      ],
      imports: [{ path: "org.accessorials", libraryId: "fl_1", versionNumber: 2 }],
      testCount: 4,
      currentVersionNumber: 3,
      version: 3,
      createdAt: 1_789_000_000,
      updatedAt: 1_790_000_000,
    }),
    ...overrides,
  };
}

/** The server's preview of the sample load: the linehaul at 500 mi and the minimum met. */
function priced(_: FormulaPreviewRequest) {
  return formulaPreviewResponseSchema.parse({
    diagnostics: [],
    bill: {
      lines: [
        {
          kind: "charge",
          label: "Linehaul",
          code: null,
          freight: true,
          amount: { $q: "1425", u: "$" },
          quantity: { $q: "500", u: "mi" },
          rate: { $q: "2.85", u: "$/mi" },
          range: { from: 100, to: 160 },
        },
      ],
      notes: [],
      freightTotal: { $q: "1425", u: "$" },
      accessorialTotal: { $q: "0", u: "$" },
      total: { $q: "1425", u: "$" },
    },
    trace: [],
    tests: [],
  });
}

function draw(props: Partial<Parameters<typeof DeskFormulaTemplate>[0]> = {}) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const handlers = { onSelect: vi.fn(), onBack: vi.fn() };
  const tree = (more: Partial<typeof props> = {}) => (
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <DeskFormulaTemplate
          recordId="ft_1"
          order={["ft_0", "ft_1", "ft_2"]}
          {...handlers}
          {...props}
          {...more}
        />
      </MemoryRouter>
    </QueryClientProvider>
  );
  const view = render(tree());
  return {
    ...view,
    ...handlers,
    step: (recordId: string) => view.rerender(tree({ recordId })),
  };
}

beforeEach(() => {
  access.write = true;
  for (const fn of Object.values(service)) fn.mockReset();
  service.get.mockResolvedValue(template());
  service.getContext.mockResolvedValue({
    context: {
      libraries: [],
      tables: [],
      codes: [{ code: "REEF", description: "Reefer unit", active: true }],
      settings: {},
    },
    libraries: [],
  });
  service.getUsage.mockResolvedValue({
    inUse: true,
    usages: [
      { type: "shipment", count: 12 },
      { type: "rate_agreement_rule", count: 2 },
    ],
  });
  service.listVersions.mockResolvedValue({
    results: [
      {
        id: "ftv_3",
        versionNumber: 3,
        changeMessage: "Raised the reefer fee",
        createdAt: 1_790_000_000,
        createdById: "usr_2",
        createdBy: { name: "Avery Lane" },
      },
      {
        id: "ftv_2",
        versionNumber: 2,
        changeMessage: "",
        createdAt: 1_789_500_000,
        createdById: "usr_2",
        createdBy: { name: "Avery Lane" },
      },
    ],
    count: 3,
    totalCount: 3,
    next: null,
    prev: null,
  });
  service.readiness.mockResolvedValue({
    canSubmit: true,
    canApprove: false,
    checks: [
      { key: "source", label: "Source checks clean", status: "pass" },
      { key: "tests", label: "Tests pass", status: "fail", detail: "3 of 4" },
    ],
    testTotal: 4,
    testPassed: 3,
  });
  service.preview.mockImplementation(async (request: FormulaPreviewRequest) => priced(request));
});

describe("DeskFormulaTemplate", () => {
  it("draws the next template's outline, not the one it left, while the next one is read", async () => {
    const { step, container } = draw();
    await screen.findByRole("heading", { name: "Reefer per mile" });
    service.get.mockReturnValue(new Promise(() => undefined));
    step("ft_2");

    expect(screen.queryByRole("heading", { name: "Reefer per mile" })).toBeNull();
    expect(container.querySelector(".dk-rec-skel")).not.toBeNull();
  });

  it("holds the card's outline while the template is read, then draws the template", async () => {
    const { container } = draw();

    expect(screen.getByRole("status")).toHaveTextContent("Loading the formula template…");
    expect(container.querySelector(".dk-rec-skel .dk-rec-head")).not.toBeNull();

    expect(await screen.findByRole("heading", { name: "Reefer per mile" })).toBeInTheDocument();
    expect(container.querySelector(".dk-rec-skel")).toBeNull();
    expect(screen.getByText("Linehaul for reefer loads")).toBeInTheDocument();
    expect(screen.getByText("Tariff")).toBeInTheDocument();
    expect(screen.getByText("v3")).toBeInTheDocument();
    expect(screen.getByText("1 library")).toBeInTheDocument();
    expect(screen.getByText("Prices shipments")).toBeInTheDocument();
    expect(
      await screen.findByText("Used by 12 shipments and 2 agreement rules"),
    ).toBeInTheDocument();
  });

  it("shows the source coloured from the module's tokens", async () => {
    const { container } = draw();
    await screen.findByRole("heading", { name: "Reefer per mile" });

    const code = container.querySelector("pre.dk-ft-code");
    expect(code).toHaveTextContent('charge "Linehaul"');
    expect(code?.getAttribute("data-language")).toBe("tariff");
    expect(within(code as HTMLElement).getAllByText("charge")[0]).toHaveClass(
      "text-syntax-keyword",
    );
    expect(within(code as HTMLElement).getByText("REEF")).toHaveClass("text-syntax-meta");
  });

  it("lists what it bills, from the checker's outline, a charge under an if marked as such", async () => {
    draw();
    await screen.findByRole("heading", { name: "Reefer per mile" });

    const linehaul = screen.getByText("Linehaul", { selector: "b" }).closest(".dk-ft-item");
    expect(linehaul).not.toBeNull();
    const reefer = screen.getByText("Reefer unit", { selector: "b" }).closest(".dk-ft-item");
    expect(reefer).toHaveTextContent("when it applies");
    expect(reefer).toHaveTextContent("REEF");
    expect(screen.getByText("Minimum charge").closest(".dk-ft-item")).toHaveTextContent(
      "minimum $250",
    );
  });

  it("lists the inputs a contract sets, with their defaults", async () => {
    draw();
    await screen.findByRole("heading", { name: "Reefer per mile" });

    const rate = screen.getByText("rate", { selector: "code" }).closest(".dk-ft-v");
    expect(rate).toHaveTextContent("Money/mi · Paid per loaded mile.");
    expect(rate).toHaveTextContent("$2.85/mi");
    const fee = screen.getByText("reeferFee", { selector: "code" }).closest(".dk-ft-v");
    expect(fee).toHaveTextContent("Required");
    expect(fee).toHaveTextContent("Set by each contract");
  });

  it("rates the sample load on the server with the saved source and shows the bill", async () => {
    draw();

    expect(await screen.findAllByText("$1,425.00")).not.toHaveLength(0);
    expect(service.preview).toHaveBeenCalledWith({ source: SOURCE }, expect.anything());
    expect(screen.getByText("500 mi × $2.85/mi")).toBeInTheDocument();
  });

  it("says why the sample load got no bill, in the tariff's own words", async () => {
    service.preview.mockResolvedValue(
      formulaPreviewResponseSchema.parse({
        diagnostics: [],
        refusal: { message: "Set the reefer fee in the contract", range: { from: 0, to: 4 } },
      }),
    );
    draw();

    expect(await screen.findByText("Set the reefer fee in the contract")).toBeInTheDocument();
    expect(screen.getByText("Does not rate")).toBeInTheDocument();
  });

  it("rates nothing for a person who may not write templates, and says why", async () => {
    access.write = false;
    draw();

    await screen.findByRole("heading", { name: "Reefer per mile" });
    expect(
      screen.getByText("Rating a sample load needs permission to write formula templates."),
    ).toBeInTheDocument();
    expect(service.preview).not.toHaveBeenCalled();
  });

  it("lists what stands between a draft and approval", async () => {
    service.get.mockResolvedValue(template({ status: "Draft" }));
    draw();

    expect(await screen.findByText("Before it can be approved")).toBeInTheDocument();
    expect(screen.getByText("3 of 4 tests pass")).toBeInTheDocument();
    expect(screen.getByText("Tests pass").closest(".dk-rec-ck")).toHaveClass("dk-bad");
    expect(screen.getByText("Source checks clean").closest(".dk-rec-ck")).toHaveClass("dk-ok");
  });

  it("does not ask an active template what stands before approval", async () => {
    draw();
    await screen.findByRole("heading", { name: "Reefer per mile" });

    expect(service.readiness).not.toHaveBeenCalled();
    expect(screen.queryByText("Before it can be approved")).not.toBeInTheDocument();
  });

  it("lists the versions oldest first, with a way to the rest", async () => {
    draw();

    const second = await screen.findByText("v2 · No note");
    const third = screen.getByText("v3 · Raised the reefer fee");
    expect(second.compareDocumentPosition(third) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(screen.getByRole("link", { name: /All versions/ })).toHaveAttribute(
      "href",
      "/billing/configuration-files/formula-templates/ft_1/edit",
    );
  });

  it("steps through the table's rows and goes back to it", async () => {
    const { onSelect, onBack } = draw();
    await screen.findByRole("heading", { name: "Reefer per mile" });

    expect(screen.getByText("2 of 3")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Next template" }));
    fireEvent.click(screen.getByRole("button", { name: "Previous template" }));
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));

    expect(onSelect.mock.calls).toEqual([["ft_2"], ["ft_0"]]);
    expect(onBack).toHaveBeenCalledOnce();
  });

  it("does not step from a table of one", async () => {
    draw({ order: ["ft_1"] });
    await screen.findByRole("heading", { name: "Reefer per mile" });

    expect(screen.queryByRole("button", { name: "Next template" })).not.toBeInTheDocument();
  });

  it("opens the template in the Formula Studio", async () => {
    draw();

    expect(await screen.findByRole("link", { name: /Open in Formula Studio/ })).toHaveAttribute(
      "href",
      "/billing/configuration-files/formula-templates/ft_1/edit",
    );
  });

  it("says when the template cannot be read, with the way back", async () => {
    service.get.mockRejectedValue(new Error("not found"));
    const { onBack } = draw();

    expect(
      await screen.findByRole("heading", { name: "This formula template could not be read" }),
    ).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Back to the table" }));
    expect(onBack).toHaveBeenCalledOnce();
  });
});
