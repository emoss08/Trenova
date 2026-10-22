import { render, screen } from "@testing-library/react";
import { rulesSync } from "@trenova/shared/rules/runtime";
import type { RateAgreement, RateAgreementRule } from "@trenova/shared/types/rate";
import { FormProvider, useForm } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { LaneEditor } from "../lane-editor";

// The lane editor's own fields talk to the network; what is under test is what it shows
// about each lane, which the rules module answers.
vi.mock("@/components/autocomplete-fields", () => ({
  FormulaTemplateAutocompleteField: () => null,
}));
vi.mock("../lane-tariff-inputs", () => ({
  LaneTariffInputs: ({ index, templateId }: { index: number; templateId?: string }) => (
    <p>{`lane ${index} prices through ${templateId}`}</p>
  ),
}));
vi.mock("../lane-scope-fields", () => ({ LaneScopeFields: () => null }));
vi.mock("../lane-history-popover", () => ({ LaneHistoryPopover: () => null }));
vi.mock("@/hooks/use-lane-scope-labels", () => ({
  useLaneScopeLabels: () => (_type: string, value: string) => ({ us_tx: "TX", us_il: "IL" })[value],
}));

function lane(overrides: Partial<RateAgreementRule>): RateAgreementRule {
  return {
    label: "",
    status: "Active",
    originScopeType: "Any",
    originScopeValue: "",
    originCity: "",
    destinationScopeType: "Any",
    destinationScopeValue: "",
    destinationCity: "",
    direction: "Directional",
    priority: 0,
    effectiveFrom: 0,
    formulaTemplateId: "ft_per_mile",
    inputs: {},
    ...overrides,
  } as RateAgreementRule;
}

function Harness({ rules }: { rules: RateAgreementRule[] }) {
  const form = useForm<RateAgreement>({ defaultValues: { rules } as RateAgreement });
  return (
    <FormProvider {...form}>
      <LaneEditor />
    </FormProvider>
  );
}

afterEach(() => {
  vi.restoreAllMocks();
});

describe("LaneEditor", () => {
  it("shows each lane's key with names, its specificity, and the lane that can never apply", () => {
    const spy = vi.spyOn(rulesSync(), "call");
    render(
      <Harness
        rules={[
          lane({
            label: "Texas low",
            originScopeType: "State",
            originScopeValue: "us_tx",
            destinationScopeType: "State",
            destinationScopeValue: "us_il",
            priority: 0,
          }),
          lane({
            label: "Texas high",
            originScopeType: "State",
            originScopeValue: "us_tx",
            destinationScopeType: "State",
            destinationScopeValue: "us_il",
            priority: 10,
          }),
        ]}
      />,
    );

    expect(screen.getAllByText("ST:TX>ST:IL")).toHaveLength(2);
    expect(screen.getAllByText("specificity 1024000")).toHaveLength(2);
    // The lower-priority lane is the shadowed one, though it comes first.
    expect(
      screen.getByText(
        "Another lane covers the same geography with a higher priority, so this one can never apply.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("One lane can never apply")).toBeInTheDocument();

    // The whole agreement is asked about once per question.
    const ops = spy.mock.calls.map(([op]) => op);
    expect(ops.filter((op) => op === "lane.keys")).toHaveLength(1);
    expect(ops.filter((op) => op === "lane.specificity")).toHaveLength(1);
    expect(ops.filter((op) => op === "lane.coverage")).toHaveLength(1);
  });

  it("sets each lane's terms through its own tariff's inputs", () => {
    render(
      <Harness
        rules={[lane({ formulaTemplateId: "ft_per_mile" }), lane({ formulaTemplateId: "ft_ltl" })]}
      />,
    );

    expect(screen.getByText("lane 0 prices through ft_per_mile")).toBeInTheDocument();
    expect(screen.getByText("lane 1 prices through ft_ltl")).toBeInTheDocument();
    expect(screen.queryByText("Minimum charge")).not.toBeInTheDocument();
    expect(screen.queryByText("Rate matrix")).not.toBeInTheDocument();
  });

  it("says a lane matched by radius has no key", () => {
    render(<Harness rules={[lane({ originScopeType: "Radius" })]} />);
    expect(screen.getByText("matched by radius")).toBeInTheDocument();
  });
});
