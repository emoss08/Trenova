import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import type { RateMatrix, RateMatrixCell, RateMatrixDimension } from "@trenova/shared/types/rate";
import { useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";
import { MatrixGridEditor } from "../matrix-grid-editor";

const listCells = vi.fn();

vi.mock("@/services/api", () => ({
  apiService: {
    get rateMatrixService() {
      return { listCells, replaceCells: vi.fn() };
    },
  },
}));

const DIMENSIONS = [
  { position: 0, kind: "Zone", matchMode: "Exact", label: "" },
  { position: 1, kind: "WeightBreak", matchMode: "Range", label: "" },
] as RateMatrixDimension[];

function Harness({ valueType = "Money" }: { valueType?: RateMatrix["valueType"] }) {
  const form = useForm<RateMatrix>({
    defaultValues: { dimensions: DIMENSIONS, valueType } as RateMatrix,
  });
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false } } }),
  );
  return (
    <QueryClientProvider client={client}>
      <FormProvider {...form}>
        <MatrixGridEditor rateMatrixId="rm_1" />
      </FormProvider>
    </QueryClientProvider>
  );
}

function cell(fields: Partial<RateMatrixCell>): RateMatrixCell {
  return { d0Key: "", d1Key: "", d2Key: "", d3Key: "", value: 1, ...fields } as RateMatrixCell;
}

describe("MatrixGridEditor", () => {
  // Each zone's breaks are checked on their own (testdata/matrix.json); the old browser check
  // pooled every zone's bands and missed this gap behind zone A's bands.
  it("lists each finding once, named by the axis the screen shows, with the zones it is in", async () => {
    listCells.mockResolvedValue([
      cell({ d0Key: "A", d1Min: 0, d1Max: 1000 }),
      cell({ d0Key: "A", d1Min: 1000 }),
      cell({ d0Key: "C", d1Min: 0, d1Max: 500 }),
      cell({ d0Key: "C", d1Min: 800 }),
      cell({ d0Key: "D", d1Min: 0, d1Max: 500 }),
      cell({ d0Key: "D", d1Min: 800 }),
    ]);

    render(<Harness />);

    const finding = await screen.findByText(
      "Weight break jumps from 500 to 800 with nothing in between.",
    );
    expect(finding).toHaveTextContent("Seen in C and D");
    expect(screen.getAllByRole("listitem")).toHaveLength(1);
  });

  it("shows each cell in the unit of the matrix's value type", async () => {
    listCells.mockResolvedValue([cell({ d0Key: "A", d1Min: 0, d1Max: 1000, value: 42.5 })]);

    render(<Harness valueType="Money/cwt" />);

    expect(await screen.findByDisplayValue("42.5")).toBeInTheDocument();
    expect(screen.getByText("$/cwt")).toBeInTheDocument();
  });

  it("shows a plain number with no unit", async () => {
    listCells.mockResolvedValue([cell({ d0Key: "A", d1Min: 0, d1Max: 1000, value: 3 })]);

    render(<Harness valueType="Number" />);

    expect(await screen.findByDisplayValue("3")).toBeInTheDocument();
    expect(screen.queryByText("$")).toBeNull();
  });

  it("refuses a fraction in a whole-number matrix and holds the save until it is fixed", async () => {
    listCells.mockResolvedValue([cell({ d0Key: "A", d1Min: 0, d1Max: 1000, value: 3 })]);

    render(<Harness valueType="Int" />);

    const input = await screen.findByDisplayValue("3");
    fireEvent.change(input, { target: { value: "3.5" } });

    expect(input).toHaveAttribute("aria-invalid", "true");
    expect(
      screen.getByText("1 cell is not a whole number. This matrix holds whole numbers."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save rates" })).toBeDisabled();

    fireEvent.change(input, { target: { value: "4" } });

    expect(input).toHaveAttribute("aria-invalid", "false");
    expect(screen.queryByText(/not a whole number/)).toBeNull();
    expect(screen.getByRole("button", { name: "Save rates" })).toBeEnabled();
  });
});
