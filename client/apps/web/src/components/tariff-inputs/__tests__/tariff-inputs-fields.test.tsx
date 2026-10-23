import { act, cleanup, render, screen, within } from "@testing-library/react";
import { setLocale } from "@trenova/shared/i18n/runtime";
import userEvent from "@testing-library/user-event";
import type { TariffInputDeclaration, TariffInputValues } from "@trenova/shared/types/tariff";
import { zodResolver } from "@hookform/resolvers/zod";
import { translate } from "@trenova/shared/i18n/runtime";
import { tariffInputValuesSchemaFor } from "@trenova/shared/lib/tariff-inputs";
import { FormProvider, useForm, useWatch, type Resolver } from "react-hook-form";
import { z } from "zod";
import { afterEach, describe, expect, it, vi } from "vitest";
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
  descriptor: {
    kind: "enum",
    name: "org.service.Service",
    variants: ["Standard", "Guaranteed"],
    ordered: false,
  },
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

const residential: TariffInputDeclaration = {
  name: "residential",
  doc: "Whether delivery is residential.",
  type: "Bool",
  descriptor: { kind: "bool" },
  required: false,
  default: true,
  range: { from: 120, to: 131 },
};

function Values() {
  const values = useWatch({ name: "inputs" }) as unknown;
  return <pre data-testid="values">{JSON.stringify(values ?? {})}</pre>;
}

function Harness({
  declarations,
  initial = {},
  inherited,
  validate = false,
  onReveal,
}: {
  declarations: TariffInputDeclaration[];
  initial?: Record<string, unknown>;
  inherited?: TariffInputValues;
  /** Validate as a contract form does: the shared input check in a resolver, on every change. */
  validate?: boolean;
  onReveal?: (declaration: TariffInputDeclaration) => void;
}) {
  const form = useForm<{ inputs: Record<string, unknown> }>({
    defaultValues: { inputs: initial },
    ...(validate
      ? {
          mode: "onChange" as const,
          resolver: zodResolver(
            z.object({ inputs: tariffInputValuesSchemaFor(declarations, translate) }),
          ) as Resolver<{ inputs: Record<string, unknown> }>,
        }
      : {}),
  });
  return (
    <FormProvider {...form}>
      <TariffInputsFields
        declarations={declarations}
        name="inputs"
        inherited={inherited}
        inheritedFrom={inherited ? "agreement" : undefined}
        onReveal={onReveal}
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

    await user.type(screen.getByRole("textbox", { name: "Paid per loaded mile" }), "2.35");

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

  it("shows an unset yes-or-no input as its default and writes it when switched", async () => {
    const user = userEvent.setup();
    render(<Harness declarations={[residential]} />);

    const toggle = screen.getByRole("switch", { name: "Whether delivery is residential" });
    expect(toggle).toBeChecked();
    expect(values()).toEqual({});

    await user.click(toggle);

    expect(values()).toEqual({ residential: false });
    expect(toggle).not.toBeChecked();
  });

  it("links each input to its declaration when the form sits beside the source", async () => {
    const user = userEvent.setup();
    const onReveal = vi.fn();
    render(<Harness declarations={[residential, minimumCharge]} onReveal={onReveal} />);

    await user.click(screen.getByRole("button", { name: "Show residential in the source" }));

    expect(onReveal).toHaveBeenCalledWith(residential);
    expect(screen.getByRole("button", { name: "Show minimumCharge in the source" })).toBeDisabled();
  });

  it("leaves out the declaration links when no source is beside the form", () => {
    render(<Harness declarations={[residential]} />);

    expect(
      screen.queryByRole("button", { name: "Show residential in the source" }),
    ).not.toBeInTheDocument();
  });

  it("says when a tariff declares no inputs", () => {
    render(<Harness declarations={[]} />);

    expect(screen.getByText("This tariff declares no inputs.")).toBeInTheDocument();
  });

  describe("an input's where bounds", () => {
    const boundedRate: TariffInputDeclaration = {
      ...rate,
      required: false,
      default: { $q: "2.35", u: "$/mi" },
      constraint: {
        range: { min: { $q: "0.5", u: "$/mi" }, max: { $q: "15", u: "$/mi" }, maxInclusive: true },
      },
    };

    it("shows the bounds as the field's help text", () => {
      render(<Harness declarations={[boundedRate]} />);
      expect(screen.getByText("From $0.50/mi to $15.00/mi")).toBeInTheDocument();
    });

    it("keeps the bounds beside the doc when the field links to its declaration", () => {
      render(<Harness declarations={[boundedRate]} onReveal={vi.fn()} />);
      expect(screen.getByText("From $0.50/mi to $15.00/mi")).toBeInTheDocument();
      expect(screen.getByRole("button", { name: "Show rate in the source" })).toBeInTheDocument();
      expect(screen.queryByText(/object Object/)).not.toBeInTheDocument();
    });

    it("puts the bounds after the doc's description", () => {
      render(
        <Harness
          declarations={[
            {
              ...service,
              type: "Int",
              descriptor: { kind: "int" },
              default: { $n: "2" },
              constraint: { range: { min: { $n: "0" }, max: { $n: "10" }, maxInclusive: false } },
            },
          ]}
        />,
      );
      expect(
        screen.getByText("Guaranteed pays a premium. · At least 0 and below 10"),
      ).toBeInTheDocument();
    });

    it("refuses a value outside the bounds as it is typed, and clears once it fits", async () => {
      const user = userEvent.setup();
      render(<Harness declarations={[boundedRate]} validate />);
      const field = screen.getByPlaceholderText("Default: $2.35/mi");

      await user.type(field, "235");
      expect(await screen.findByText("Must be from $0.50/mi to $15.00/mi")).toBeInTheDocument();

      await user.clear(field);
      await user.type(field, "15");
      await act(async () => {});
      expect(screen.queryByText("Must be from $0.50/mi to $15.00/mi")).not.toBeInTheDocument();
    });

    it("offers a text input bounded by a list as a select of that list", async () => {
      const user = userEvent.setup();
      render(
        <Harness
          declarations={[
            {
              name: "level",
              doc: "Service level.",
              type: "Text",
              descriptor: { kind: "text" },
              required: false,
              default: "Standard",
              constraint: { oneOf: ["Standard", "Guaranteed"] },
              range: null,
            },
          ]}
        />,
      );
      expect(screen.queryByText(/One of/)).not.toBeInTheDocument();
      await user.click(screen.getByRole("button", { name: /Default: Standard/ }));
      const options = await screen.findAllByRole("option");
      expect(options.map((option) => option.textContent)).toEqual(["Standard", "Guaranteed"]);
      await user.click(options[1]!);
      expect(values()).toEqual({ level: "Guaranteed" });
    });

    it("offers only the variants an enum's list allows", async () => {
      const user = userEvent.setup();
      render(
        <Harness
          declarations={[
            {
              ...service,
              descriptor: {
                kind: "enum",
                name: "Service",
                variants: ["Standard", "Guaranteed", "Expedited"],
                ordered: false,
              },
              constraint: { oneOf: ["Standard", "Expedited"] },
            },
          ]}
        />,
      );
      await user.click(screen.getByRole("button", { name: /Default: Standard/ }));
      const options = await screen.findAllByRole("option");
      expect(options.map((option) => option.textContent)).toEqual(["Standard", "Expedited"]);
    });

    // `input cls: FreightClass = Class70 where Class50..=Class100`, as tariff.check sends it
    // (testdata/tariff/check/ordered-enums.json), with the variant list cut after Class110.
    const freightClass: TariffInputDeclaration = {
      name: "cls",
      doc: null,
      type: "FreightClass",
      descriptor: {
        kind: "enum",
        name: "FreightClass",
        ordered: true,
        variants: ["Class50", "Class55", "Class70", "Class77_5", "Class100", "Class110"],
      },
      required: false,
      default: "Class70",
      constraint: { range: { min: "Class55", max: "Class100", maxInclusive: true } },
      range: { from: 17, to: 20 },
    };

    it("offers only the variants an ordered enum's range spans", async () => {
      const user = userEvent.setup();
      render(<Harness declarations={[freightClass]} />);
      await user.click(screen.getByRole("button", { name: /Default: Class70/ }));
      const options = await screen.findAllByRole("option");
      expect(options.map((option) => option.textContent)).toEqual([
        "Class55",
        "Class70",
        "Class77_5",
        "Class100",
      ]);
    });

    // `input zip: Text = "30301" where "30000"..="39999"` (testdata/tariff/check/text-ranges.json).
    const zip: TariffInputDeclaration = {
      name: "zip",
      doc: null,
      type: "Text",
      descriptor: { kind: "text" },
      required: false,
      default: "30301",
      constraint: { range: { min: "30000", max: "39999", maxInclusive: true } },
      range: { from: 17, to: 20 },
    };

    it("shows a text range as help text and refuses text past it as it is typed", async () => {
      const user = userEvent.setup();
      render(<Harness declarations={[zip]} validate />);
      expect(screen.getByText("From 30000 to 39999")).toBeInTheDocument();
      const field = screen.getByPlaceholderText("Default: 30301");
      await user.type(field, "4");
      expect(await screen.findByText("Must be from 30000 to 39999")).toBeInTheDocument();
      await user.clear(field);
      await user.type(field, "35004");
      await act(async () => {});
      expect(screen.queryByText("Must be from 30000 to 39999")).not.toBeInTheDocument();
    });
  });

  describe("a time-of-day input", () => {
    // `input open: Time = @08:00 where @06:00..@17:00` and `input close: Time? = none`, as
    // tariff.check sends them (testdata/tariff/check/time.json).
    const open: TariffInputDeclaration = {
      name: "open",
      doc: "When the dock opens.",
      type: "Time",
      descriptor: { kind: "time" },
      required: false,
      default: { $time: "08:00:00" },
      constraint: {
        range: { min: { $time: "06:00:00" }, max: { $time: "17:00:00" }, maxInclusive: false },
      },
      range: { from: 42, to: 46 },
    };
    const close: TariffInputDeclaration = {
      name: "close",
      doc: null,
      type: "Time?",
      descriptor: { kind: "optional", inner: { kind: "time" } },
      required: false,
      default: null,
      constraint: null,
      range: { from: 90, to: 95 },
    };

    it("writes the time typed as the module reads it, with seconds", async () => {
      const user = userEvent.setup();
      render(<Harness declarations={[open]} />);
      expect(screen.getByText("At least 6:00 AM and below 5:00 PM")).toBeInTheDocument();
      const field = screen.getByLabelText("When the dock opens");
      expect(field).toHaveAttribute("type", "time");
      await user.type(field, "07:45");
      expect(values()).toEqual({ open: { $time: "07:45:00" } });
    });

    it("shows a stored time with its seconds", () => {
      render(<Harness declarations={[open]} initial={{ open: { $time: "07:30:15" } }} />);
      const field = screen.getByLabelText("When the dock opens");
      expect(field).toHaveValue("07:30:15");
      expect(field).toHaveAttribute("step", "1");
    });

    it("refuses a time outside its range as it is entered, and clears once it fits", async () => {
      const user = userEvent.setup();
      render(<Harness declarations={[open]} validate />);
      const field = screen.getByLabelText("When the dock opens");
      await user.type(field, "17:00");
      expect(
        await screen.findByText("Must be at least 6:00 AM and below 5:00 PM"),
      ).toBeInTheDocument();
      await user.clear(field);
      await user.type(field, "16:59");
      await act(async () => {});
      expect(
        screen.queryByText("Must be at least 6:00 AM and below 5:00 PM"),
      ).not.toBeInTheDocument();
    });

    it("writes none when an optional time is emptied", async () => {
      const user = userEvent.setup();
      render(<Harness declarations={[close]} initial={{ close: { $time: "18:00:00" } }} />);
      await user.clear(screen.getByLabelText("Close"));
      expect(values()).toEqual({ close: null });
    });
  });
});
