import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import {
  testExpressionResponseSchema,
  type FormulaTemplate,
  type TestExpressionRequest,
} from "@trenova/shared/types/formula-template";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { DeskFormulaTemplate } from "../desk-formula-template";
import { shipmentSchema } from "@/components/formula-editor/__tests__/formula-fixtures";

const service = vi.hoisted(() => ({
  get: vi.fn(),
  getSchema: vi.fn(),
  getUsage: vi.fn(),
  listVersions: vi.fn(),
  readiness: vi.fn(),
  test: vi.fn(),
}));
const access = vi.hoisted(() => ({ write: true }));

vi.mock("@/services/api", () => ({ apiService: { formulaTemplateService: service } }));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: access.write, isLoading: false }),
}));

/** A template as the server's GET /formula-templates/{id}/ returns it, after the client schema. */
function template(overrides: Partial<FormulaTemplate> = {}): FormulaTemplate {
  return {
    id: "ft_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    name: "Reefer per mile",
    description: "Linehaul for reefer loads",
    type: "FreightCharge",
    expression:
      "max(baseRate * totalDistance, minimumLinehaul) + reeferFee + (teamService ? 150 : 0)",
    status: "Active",
    schemaId: "shipment",
    languageVersion: 1,
    variableDefinitions: [
      {
        name: "baseRate",
        type: "Number",
        description: "Rate per mile",
        required: true,
        defaultValue: 2.85,
      },
      {
        name: "minimumLinehaul",
        type: "Number",
        description: "",
        required: false,
        defaultValue: 500,
      },
      {
        name: "reeferFee",
        type: "Number",
        description: "Reefer unit",
        required: false,
        defaultValue: 75,
      },
      {
        name: "teamService",
        type: "Boolean",
        description: "",
        required: false,
        defaultValue: false,
      },
      { name: "lanes", type: "Array", description: "", required: false, defaultValue: ["GA"] },
    ],
    breakdownDefinitions: [],
    minCharge: 250,
    maxCharge: null,
    roundingMode: "HalfUp",
    roundingPrecision: 2,
    currentVersionNumber: 3,
    version: 3,
    createdAt: 1_789_000_000,
    updatedAt: 1_790_000_000,
    ...overrides,
  };
}

/** What the engine answers for a request: the rate per mile over a 500-mile sample, and the fee. */
function priced(request: TestExpressionRequest) {
  const rate = Number(request.variables.baseRate);
  const result = rate * 500 + Number(request.variables.reeferFee);
  return testExpressionResponseSchema.parse({
    valid: true,
    result: String(result),
    diagnostics: [],
    error: null,
    breakdown: [],
    guardrail: { applied: false, rawAmount: result },
    rounding: { mode: "HalfUp", precision: 2, applied: false, unroundedAmount: result },
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
  service.getSchema.mockResolvedValue(shipmentSchema);
  service.getUsage.mockResolvedValue({
    inUse: true,
    usages: [
      { type: "shipment", count: 12 },
      { type: "rate_matrix", count: 2 },
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
      { key: "expression", label: "Formula runs", status: "pass" },
      { key: "scenarios", label: "Scenarios pass", status: "fail", detail: "One scenario is off" },
    ],
    scenarioTotal: 4,
    scenarioPassed: 3,
  });
  service.test.mockImplementation(async (request: TestExpressionRequest) => priced(request));
});

describe("DeskFormulaTemplate", () => {
  it("draws the next template's outline, not the one it left, while the next one is read", async () => {
    const { container, step } = draw();
    await screen.findByRole("heading", { name: "Reefer per mile" });

    service.get.mockReturnValue(new Promise(() => {}));
    step("ft_0");

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
    expect(screen.getByText("Freight charge")).toBeInTheDocument();
    expect(screen.getByText("v3")).toBeInTheDocument();
    expect(screen.getByText("Active")).toBeInTheDocument();
    expect(screen.getByText("Prices shipments")).toBeInTheDocument();
    expect(await screen.findByText("Used by 12 shipments and 2 rate matrices")).toBeInTheDocument();
  });

  it("sets the template's own variables apart from the shipment's fields in the formula", async () => {
    draw();

    const formula = await screen.findByLabelText("Formula", { selector: "pre" });
    expect(formula).toHaveTextContent(template().expression);
    expect(within(formula).getByText("baseRate")).toHaveClass("dk-ft-var");
    await waitFor(() =>
      expect(within(formula).getByText("totalDistance")).toHaveClass("dk-ft-field"),
    );
    expect(within(formula).getByText("max")).toHaveClass("dk-ft-fn");
  });

  it("marks a variable in the formula while its row is pointed at", async () => {
    draw();

    const formula = await screen.findByLabelText("Formula", { selector: "pre" });
    const row = screen.getByLabelText("reeferFee").closest(".dk-ft-v");
    expect(row).not.toBeNull();
    fireEvent.mouseEnter(row as Element);

    expect(within(formula).getByText("reeferFee")).toHaveClass("dk-on");
    expect(within(formula).getByText("baseRate")).not.toHaveClass("dk-on");
  });

  it("states the limits and rounding the price is held to", async () => {
    draw();

    await screen.findByRole("heading", { name: "Reefer per mile" });
    const rules = document.querySelector(".dk-ft-rules") as HTMLElement;
    expect(within(rules).getByText("$250.00")).toBeInTheDocument();
    expect(within(rules).getByText("None")).toBeInTheDocument();
    expect(within(rules).getByText("Half up, 2 places")).toBeInTheDocument();
  });

  it("prices a sample load with the template's defaults", async () => {
    draw();

    expect(await screen.findAllByText("$1,500.00")).not.toHaveLength(0);
    const request = service.test.mock.calls[0][0] as TestExpressionRequest;
    expect(request.expression).toBe(template().expression);
    expect(request.variables).toMatchObject({
      baseRate: 2.85,
      reeferFee: 75,
      teamService: false,
      totalDistance: 500,
    });
    expect(request.minCharge).toBe("250");
    expect(request).not.toHaveProperty("maxCharge");
    expect(request).not.toHaveProperty("shipmentId");
  });

  it("prices again with the values a person sets", async () => {
    draw();
    await screen.findAllByText("$1,500.00");

    fireEvent.change(screen.getByLabelText("baseRate"), { target: { value: "3" } });
    fireEvent.click(screen.getByRole("button", { name: "Yes" }));

    expect(await screen.findAllByText("$1,575.00", {}, { timeout: 2000 })).not.toHaveLength(0);
    const last = service.test.mock.calls.at(-1)?.[0] as TestExpressionRequest;
    expect(last.variables).toMatchObject({ baseRate: 3, teamService: true });
    expect(screen.getByRole("button", { name: "Reset to defaults" })).toBeInTheDocument();
  });

  it("leaves the default in place for a figure that does not read as one", async () => {
    draw();
    await screen.findAllByText("$1,500.00");
    const calls = service.test.mock.calls.length;

    fireEvent.change(screen.getByLabelText("baseRate"), { target: { value: "three" } });

    expect(screen.getByLabelText("baseRate")).toHaveAttribute("aria-invalid", "true");
    await new Promise((resolve) => setTimeout(resolve, 600));
    expect(service.test.mock.calls.length).toBe(calls);
  });

  it("puts every variable back to its default", async () => {
    draw();
    await screen.findAllByText("$1,500.00");

    fireEvent.change(screen.getByLabelText("baseRate"), { target: { value: "3" } });
    fireEvent.click(screen.getByRole("button", { name: "Reset to defaults" }));

    expect(screen.getByLabelText("baseRate")).toHaveValue("2.85");
    expect(screen.queryByRole("button", { name: "Reset to defaults" })).not.toBeInTheDocument();
  });

  it("shows a list or record variable's default without a field for it", async () => {
    draw();
    await screen.findByRole("heading", { name: "Reefer per mile" });

    expect(screen.queryByLabelText("lanes")).not.toBeInTheDocument();
    expect(screen.getByText('["GA"]')).toBeInTheDocument();
  });

  it("says why the engine could not price it, in the engine's words", async () => {
    service.test.mockResolvedValue(
      testExpressionResponseSchema.parse({
        valid: false,
        diagnostics: [],
        error: {
          scope: "expression",
          code: "missingField",
          message:
            "no value for {0} on this record; write {1} so the formula still prices when it is missing",
          args: ["reeferFee", "coalesce(reeferFee, 0)"],
          from: 25,
          to: 34,
          fields: ["reeferFee"],
          guards: ["coalesce(reeferFee, 0)"],
        },
        statements: [],
      }),
    );
    draw();

    expect(
      await screen.findByText(
        "no value for reeferFee on this record; write coalesce(reeferFee, 0) so the formula still prices when it is missing",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("Does not price")).toBeInTheDocument();
  });

  it("says why it could not price a formula the check refused", async () => {
    service.test.mockResolvedValue(
      testExpressionResponseSchema.parse({
        valid: false,
        diagnostics: [
          {
            scope: "expression",
            from: 0,
            to: 10,
            severity: "error",
            code: "unknown-variable",
            message:
              "{0} isn't a shipment field or one of this template's inputs. Declare it as an input, or pick a field from the reference.",
            args: ["`reeferFees`"],
            name: "reeferFees",
            fixes: [],
          },
        ],
      }),
    );
    draw();

    expect(
      await screen.findByText(
        "`reeferFees` isn't a shipment field or one of this template's inputs. Declare it as an input, or pick a field from the reference.",
      ),
    ).toBeInTheDocument();
  });

  it("lists every warning the server's check found, breakdown lines included", async () => {
    service.test.mockImplementation(async (request: TestExpressionRequest) => ({
      ...priced(request),
      diagnostics: [
        {
          scope: "breakdowns[0].expression",
          from: 0,
          to: 6,
          severity: "warning" as const,
          code: "nullable" as const,
          message:
            "{0} is empty on some shipments, and the formula would fail to rate them. Give it a fallback: {1}",
          args: ["`weight`", "coalesce(weight, 0)"],
          name: "weight",
          fixes: [],
        },
      ],
    }));
    draw();

    expect(
      await screen.findByText(
        "`weight` is empty on some shipments, and the formula would fail to rate them. Give it a fallback: coalesce(weight, 0)",
      ),
    ).toBeInTheDocument();
  });

  it("prices nothing for a person who may not write formulas, and says why", async () => {
    access.write = false;
    draw();

    await screen.findByRole("heading", { name: "Reefer per mile" });
    expect(
      screen.getByText("Pricing a sample load needs permission to write formula templates."),
    ).toBeInTheDocument();
    expect(screen.queryByLabelText("baseRate")).not.toBeInTheDocument();
    expect(screen.getByText("2.85")).toBeInTheDocument();
    expect(screen.queryByText("Sample price")).not.toBeInTheDocument();
    expect(service.test).not.toHaveBeenCalled();
  });

  it("lists what stands between a draft and approval", async () => {
    service.get.mockResolvedValue(template({ status: "Draft" }));
    draw();

    expect(await screen.findByText("Before it can be approved")).toBeInTheDocument();
    expect(screen.getByText("3 of 4 scenarios pass")).toBeInTheDocument();
    expect(screen.getByText("Scenarios pass").closest(".dk-rec-ck")).toHaveClass("dk-bad");
    expect(screen.getByText("Formula runs").closest(".dk-rec-ck")).toHaveClass("dk-ok");
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

  it("clears what the person set when it steps to another template", async () => {
    const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    service.get.mockImplementation(async (id: string) =>
      template({ id, name: id === "ft_2" ? "Dry van per mile" : "Reefer per mile" }),
    );
    const props = { order: ["ft_1", "ft_2"], onSelect: vi.fn(), onBack: vi.fn() };
    const { rerender } = render(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <DeskFormulaTemplate recordId="ft_1" {...props} />
        </MemoryRouter>
      </QueryClientProvider>,
    );
    await screen.findByRole("heading", { name: "Reefer per mile" });
    fireEvent.change(screen.getByLabelText("baseRate"), { target: { value: "9" } });

    rerender(
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>
          <DeskFormulaTemplate recordId="ft_2" {...props} />
        </MemoryRouter>
      </QueryClientProvider>,
    );

    await screen.findByRole("heading", { name: "Dry van per mile" });
    await waitFor(() => expect(screen.getByLabelText("baseRate")).toHaveValue("2.85"));
  });
});
