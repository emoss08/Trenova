import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { ReportParameterOption } from "@/lib/report-parameter-options";
import type { ReportParameterDef } from "@/types/report";
import { ParameterField } from "../report-parameter-field";

/** The 26 features the stock "Desk Token Usage by Person" report lets a person choose from. */
const FEATURE_CODES = [
  "AgentTurn",
  "AssistantTurn",
  "BillingQueueBrief",
  "ChapterTitle",
  "CustomerEmailDraft",
  "DeskBriefing",
  "DetentionSummary",
  "DispatchSuggestion",
  "DocumentClassify",
  "DocumentIntelligenceExtract",
  "EdiMapping",
  "EmailTriage",
  "ExceptionTriage",
  "FuelAudit",
  "InvoiceReview",
  "LaneForecast",
  "LoadMatch",
  "MemoryReview",
  "PodCheck",
  "RateConfirmation",
  "ReportBuilder",
  "SafetyReview",
  "SettlementReview",
  "ShipmentIntake",
  "TenderReply",
  "VoiceCall",
];

const NAMED: Record<string, string> = {
  AgentTurn: "Agent turn",
  BillingQueueBrief: "Billing queue brief",
  DocumentIntelligenceExtract: "Document extraction",
};

const featureOptions: ReportParameterOption[] = FEATURE_CODES.map((value, index) => ({
  value,
  label: NAMED[value] ?? `Feature ${index + 1}`,
}));

const features: ReportParameterDef = {
  name: "features",
  label: "Features",
  type: "string",
  required: false,
  multi: true,
  allowedValues: FEATURE_CODES,
};

function Harness({
  param,
  initial,
  options,
  onChange,
}: {
  param: ReportParameterDef;
  initial?: unknown;
  options?: readonly ReportParameterOption[];
  onChange: (value: unknown) => void;
}) {
  const [value, setValue] = useState<unknown>(initial);
  return (
    <ParameterField
      param={param}
      value={value}
      compact
      options={options}
      onChange={(next) => {
        setValue(next);
        onChange(next);
      }}
    />
  );
}

afterEach(() => {
  cleanup();
});

describe("ParameterField with several values from a fixed list", () => {
  it("draws one trigger and no checkboxes, however many values are allowed", () => {
    render(<Harness param={features} options={featureOptions} onChange={vi.fn()} />);

    expect(screen.getAllByRole("combobox")).toHaveLength(1);
    expect(screen.queryAllByRole("checkbox")).toHaveLength(0);
    expect(screen.getByRole("combobox", { name: "Features" })).toHaveTextContent("Select values");
  });

  it("lists every allowed value by its name, never its code", async () => {
    const user = userEvent.setup();
    render(<Harness param={features} options={featureOptions} onChange={vi.fn()} />);

    await user.click(screen.getByRole("combobox", { name: "Features" }));

    expect(screen.getAllByRole("option")).toHaveLength(26);
    expect(screen.getByRole("option", { name: "Agent turn" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Document extraction" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "AgentTurn" })).not.toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "DocumentIntelligenceExtract" })).toBeNull();
  });

  it("sends the chosen codes in the list's order and names what is chosen", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<Harness param={features} options={featureOptions} onChange={onChange} />);
    const trigger = screen.getByRole("combobox", { name: "Features" });

    await user.click(trigger);
    await user.click(screen.getByRole("option", { name: "Billing queue brief" }));

    expect(onChange).toHaveBeenLastCalledWith(["BillingQueueBrief"]);
    expect(trigger).toHaveTextContent("Billing queue brief");

    if (!screen.queryByRole("option", { name: "Agent turn" })) await user.click(trigger);
    await user.click(screen.getByRole("option", { name: "Agent turn" }));

    expect(onChange).toHaveBeenLastCalledWith(["AgentTurn", "BillingQueueBrief"]);
    expect(trigger).toHaveTextContent("2 Selected");
  });

  it("sends undefined when the last chosen value is cleared", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <Harness
        param={features}
        options={featureOptions}
        initial={["AgentTurn"]}
        onChange={onChange}
      />,
    );
    const trigger = screen.getByRole("combobox", { name: "Features" });
    expect(trigger).toHaveTextContent("Agent turn");

    await user.click(trigger);
    await user.click(screen.getByRole("option", { name: "Agent turn" }));

    expect(onChange).toHaveBeenLastCalledWith(undefined);
    expect(trigger).toHaveTextContent("Select values");
  });

  it("coerces each chosen value to the parameter's type", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <Harness
        param={{
          name: "years",
          label: "Years",
          type: "int",
          required: false,
          multi: true,
          allowedValues: ["2025", "2026"],
        }}
        onChange={onChange}
      />,
    );

    await user.click(screen.getByRole("combobox", { name: "Years" }));
    await user.click(screen.getByRole("option", { name: "2026" }));

    expect(onChange).toHaveBeenLastCalledWith([2026]);
  });

  it("writes each code in words when no names are given", async () => {
    const user = userEvent.setup();
    render(<Harness param={features} onChange={vi.fn()} />);

    await user.click(screen.getByRole("combobox", { name: "Features" }));

    expect(screen.getByRole("option", { name: "Agent Turn" })).toBeInTheDocument();
    expect(
      screen.getByRole("option", { name: "Document Intelligence Extract" }),
    ).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "AgentTurn" })).not.toBeInTheDocument();
  });
});

describe("ParameterField with one value from a fixed list", () => {
  const kind: ReportParameterDef = {
    name: "kind",
    label: "Drift kind",
    type: "string",
    required: false,
    allowedValues: ["AmountMismatch", "MissingInLedger"],
  };
  const kindOptions: ReportParameterOption[] = [
    { value: "AmountMismatch", label: "Amount mismatch" },
    { value: "MissingInLedger", label: "Missing in the ledger" },
  ];

  it("shows the chosen value's name and lists names, not codes", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(
      <Harness param={kind} options={kindOptions} initial="AmountMismatch" onChange={onChange} />,
    );
    const trigger = screen.getByRole("combobox", { name: "Drift kind" });

    expect(trigger).toHaveTextContent("Amount mismatch");
    expect(trigger).not.toHaveTextContent("AmountMismatch");

    await user.click(trigger);
    expect(screen.queryByRole("option", { name: "MissingInLedger" })).not.toBeInTheDocument();
    await user.click(screen.getByRole("option", { name: "Missing in the ledger" }));

    expect(onChange).toHaveBeenLastCalledWith("MissingInLedger");
    expect(trigger).toHaveTextContent("Missing in the ledger");
  });
});
