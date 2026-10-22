import { act, cleanup, render, screen, within } from "@testing-library/react";
import { setLocale } from "@trenova/shared/i18n/runtime";
import userEvent from "@testing-library/user-event";
import type { TariffInputDeclaration, TariffInputValues } from "@trenova/shared/types/tariff";
import { FormProvider, useForm, useWatch } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { TariffInputsFields } from "../tariff-inputs-fields";

afterEach(cleanup);

// Declarations as `tariff.check` derives them (services/crates/rules/src/tariff/check/inputs.rs):
// the type as written, its descriptor, and the default in wire form. A standard library record's
// field docs arrive as catalog messages; an organisation library's as plain text.
const rate: TariffInputDeclaration = {
  name: "rate",
  doc: "Paid per loaded mile.",
  type: "Money/mi",
  descriptor: { kind: "quantity", dimension: "Money/Distance", unit: "$/mi" },
  required: true,
  range: null,
};

const minimumCharge: TariffInputDeclaration = {
  name: "minimumCharge",
  doc: null,
  type: "Money",
  descriptor: { kind: "money" },
  required: false,
  default: { $q: "450", u: "$" },
  range: null,
};

const service: TariffInputDeclaration = {
  name: "service",
  doc: "The service level. Guaranteed pays a premium.",
  type: "Service",
  descriptor: { kind: "enum", name: "org.service.Service", variants: ["Standard", "Guaranteed"] },
  required: false,
  default: "Standard",
  range: null,
};

const breaks: TariffInputDeclaration = {
  name: "breaks",
  doc: "Weight breaks, lightest first.",
  type: "List<Break>",
  descriptor: {
    kind: "list",
    item: {
      kind: "record",
      name: "Break",
      fields: [
        {
          name: "from",
          doc: { message: "The lightest weight the break prices." },
          type: { kind: "quantity", dimension: "Weight", unit: "lb" },
        },
        {
          name: "rate",
          doc: { message: "What the break charges per hundredweight." },
          type: { kind: "quantity", dimension: "Money/Weight", unit: "$/cwt" },
        },
        {
          name: "minimum",
          doc: { message: "The least a shipment priced at the break pays, if anything." },
          type: { kind: "optional", inner: { kind: "money" } },
        },
      ],
    },
  },
  required: true,
  range: null,
};

function Values() {
  const values = useWatch({ name: "inputs" }) as unknown;
  return <pre data-testid="values">{JSON.stringify(values ?? {})}</pre>;
}

function Harness({
  declarations,
  initial = {},
  inherited,
}: {
  declarations: TariffInputDeclaration[];
  initial?: Record<string, unknown>;
  inherited?: TariffInputValues;
}) {
  const form = useForm<{ inputs: Record<string, unknown> }>({
    defaultValues: { inputs: initial },
  });
  return (
    <FormProvider {...form}>
      <TariffInputsFields
        declarations={declarations}
        name="inputs"
        inherited={inherited}
        inheritedFrom={inherited ? "agreement" : undefined}
      />
      <Values />
    </FormProvider>
  );
}

function values(): Record<string, unknown> {
  return JSON.parse(screen.getByTestId("values").textContent ?? "{}") as Record<string, unknown>;
}

describe("TariffInputsFields", () => {
  it("labels each input from its doc comment's first sentence, or its name", () => {
    render(<Harness declarations={[rate, minimumCharge, service]} />);

    expect(screen.getByText("Paid per loaded mile")).toBeInTheDocument();
    expect(screen.getByText("Minimum charge")).toBeInTheDocument();
    expect(screen.getByText("The service level")).toBeInTheDocument();
    expect(screen.getByText("Guaranteed pays a premium.")).toBeInTheDocument();
  });

  it("writes a typed rate as a wire quantity in the declared unit", async () => {
    const user = userEvent.setup();
    render(<Harness declarations={[rate]} />);

    await user.type(screen.getByRole("textbox", { name: "" }), "2.35");

    expect(values()).toEqual({ rate: { $q: "2.35", u: "$/mi" } });
  });

  it("writes money as a $ quantity and leaves it unset again when cleared", async () => {
    const user = userEvent.setup();
    render(<Harness declarations={[minimumCharge]} />);
    const field = screen.getByPlaceholderText("Default: $450.00");

    await user.type(field, "500");
    expect(values()).toEqual({ minimumCharge: { $q: "500", u: "$" } });

    await user.clear(field);
    expect(values()).toEqual({});
  });

  it("writes money per pallet in the unit the module reads it in", async () => {
    const user = userEvent.setup();
    const palletFee: TariffInputDeclaration = {
      name: "palletFee",
      doc: null,
      type: "Money/pallet",
      descriptor: { kind: "money" },
      required: false,
      default: { $q: "5", u: "$/pallet" },
      range: null,
    };
    render(<Harness declarations={[palletFee]} />);

    await user.type(screen.getByPlaceholderText("Default: $5.00/pallet"), "6");

    expect(values()).toEqual({ palletFee: { $q: "6", u: "$/pallet" } });
    expect(screen.getByText("$/pallet")).toBeInTheDocument();
  });

  it("shows what an unset input inherits from the agreement before the default", () => {
    render(
      <Harness
        declarations={[minimumCharge]}
        inherited={{ minimumCharge: { $q: "600", u: "$" } }}
      />,
    );

    expect(screen.getByPlaceholderText("Agreement: $600.00")).toBeInTheDocument();
  });

  it("says a required input nothing else sets is required", () => {
    render(<Harness declarations={[rate]} />);

    expect(screen.getByPlaceholderText("Required")).toBeInTheDocument();
  });

  it("hands a set value back to the agreement on reset", async () => {
    const user = userEvent.setup();
    render(
      <Harness
        declarations={[minimumCharge]}
        initial={{ minimumCharge: { $q: "500", u: "$" } }}
        inherited={{ minimumCharge: { $q: "600", u: "$" } }}
      />,
    );

    await user.click(screen.getByRole("button", { name: "Use the agreement's value" }));

    expect(values()).toEqual({});
  });

  it("keeps a stored value's own unit when its amount is edited", async () => {
    const user = userEvent.setup();
    render(<Harness declarations={[rate]} initial={{ rate: { $q: "1.4", u: "$/km" } }} />);
    const field = screen.getByDisplayValue("1.4");

    await user.clear(field);
    await user.type(field, "1.5");

    expect(values()).toEqual({ rate: { $q: "1.5", u: "$/km" } });
  });

  it("offers only the units of the input's dimension", async () => {
    const user = userEvent.setup();
    render(<Harness declarations={[rate]} initial={{ rate: { $q: "2.35", u: "$/mi" } }} />);

    await user.click(screen.getByRole("combobox", { name: "Unit of Paid per loaded mile" }));

    expect(screen.getByRole("option", { name: "$/mi" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "$/km" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "$/cwt" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("option", { name: "$/km" }));
    expect(values()).toEqual({ rate: { $q: "2.35", u: "$/km" } });
  });

  it("edits weight breaks as a table, one row per break", async () => {
    const user = userEvent.setup();
    render(
      <Harness
        declarations={[breaks]}
        initial={{
          breaks: [
            { from: { $q: "0", u: "lb" }, rate: { $q: "42", u: "$/cwt" }, minimum: null },
            {
              from: { $q: "500", u: "lb" },
              rate: { $q: "35.5", u: "$/cwt" },
              minimum: { $q: "95", u: "$" },
            },
          ],
        }}
      />,
    );

    const table = screen.getByRole("table", { name: "Weight breaks, lightest first" });
    const headers = within(table)
      .getAllByRole("columnheader")
      .map((header) => header.textContent);
    expect(headers).toEqual([
      "The lightest weight the break prices",
      "What the break charges per hundredweight",
      "The least a shipment priced at the break pays, if anything",
      "Actions",
    ]);
    expect(within(table).getAllByRole("row")).toHaveLength(3);

    await user.click(screen.getByRole("button", { name: "Remove row 1" }));
    await user.click(screen.getByRole("button", { name: "Add row" }));

    expect(values()).toEqual({
      breaks: [
        {
          from: { $q: "500", u: "lb" },
          rate: { $q: "35.5", u: "$/cwt" },
          minimum: { $q: "95", u: "$" },
        },
        { from: { $q: "0", u: "lb" }, rate: { $q: "0", u: "$/cwt" }, minimum: null },
      ],
    });
  });

  it("labels a standard library record's columns in the reader's language", async () => {
    await act(() => setLocale("es"));
    try {
      render(<Harness declarations={[breaks]} initial={{ breaks: [] }} />);

      expect(
        screen.getByRole("columnheader", { name: "El peso más ligero que tarifica el escalón" }),
      ).toBeInTheDocument();
    } finally {
      await act(() => setLocale("en"));
    }
  });

  it("labels an organisation library's record columns as its author wrote them", () => {
    const stops: TariffInputDeclaration = {
      name: "stops",
      doc: "Stop fees.",
      type: "List<StopFee>",
      descriptor: {
        kind: "list",
        item: {
          kind: "record",
          name: "StopFee",
          fields: [
            { name: "after", doc: "Free stops {before} the fee.", type: { kind: "int" } },
            { name: "fee", doc: null, type: { kind: "money" } },
          ],
        },
      },
      required: true,
      range: null,
    };
    render(<Harness declarations={[stops]} initial={{ stops: [] }} />);

    const table = screen.getByRole("table", { name: "Stop fees" });
    expect(
      within(table)
        .getAllByRole("columnheader")
        .map((header) => header.textContent),
    ).toEqual(["Free stops {before} the fee", "Fee", "Actions"]);
  });

  it("writes none for an emptied optional field of a break", async () => {
    const user = userEvent.setup();
    render(
      <Harness
        declarations={[breaks]}
        initial={{
          breaks: [
            {
              from: { $q: "0", u: "lb" },
              rate: { $q: "42", u: "$/cwt" },
              minimum: { $q: "95", u: "$" },
            },
          ],
        }}
      />,
    );

    await user.clear(
      screen.getByRole("textbox", {
        name: "The least a shipment priced at the break pays, if anything, row 1",
      }),
    );

    expect(values()).toEqual({
      breaks: [{ from: { $q: "0", u: "lb" }, rate: { $q: "42", u: "$/cwt" }, minimum: null }],
    });
  });

  it("starts an unset list from the value it inherits when set here", async () => {
    const user = userEvent.setup();
    const agreementBreaks = [
      { from: { $q: "0", u: "lb" }, rate: { $q: "40", u: "$/cwt" }, minimum: null },
    ];
    render(<Harness declarations={[breaks]} inherited={{ breaks: agreementBreaks }} />);

    expect(screen.getByText("Uses the agreement's: 1 row")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Set here" }));

    expect(values()).toEqual({ breaks: agreementBreaks });
    expect(
      screen.getByRole("table", { name: "Weight breaks, lightest first" }),
    ).toBeInTheDocument();
  });

  it("says when a tariff declares no inputs", () => {
    render(<Harness declarations={[]} />);

    expect(screen.getByText("This tariff declares no inputs.")).toBeInTheDocument();
  });
});
