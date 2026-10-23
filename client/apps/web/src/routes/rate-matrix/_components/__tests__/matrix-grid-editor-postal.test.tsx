import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import type { RateMatrix, RateMatrixCell, RateMatrixDimension } from "@trenova/shared/types/rate";
import { useState } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { MatrixGridEditor, type MatrixGridVersion } from "../matrix-grid-editor";

const listCells = vi.fn();
const replaceCells = vi.fn();

vi.mock("@/services/api", () => ({
  apiService: {
    get rateMatrixService() {
      return { listCells, replaceCells };
    },
  },
}));

const POSTAL_BY_WEIGHT = [
  {
    position: 0,
    kind: "ZipRange",
    matchMode: "Range",
    label: "Origin",
    keyNormalization: "None",
    rangeOverflow: "Error",
  },
  {
    position: 1,
    kind: "WeightBreak",
    matchMode: "Range",
    label: "",
    keyNormalization: "None",
    rangeOverflow: "Error",
  },
] as RateMatrixDimension[];

function Harness({ version }: { version?: MatrixGridVersion }) {
  const form = useForm<RateMatrix>({
    defaultValues: { dimensions: POSTAL_BY_WEIGHT, valueType: "Money" } as RateMatrix,
  });
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false } } }),
  );
  return (
    <QueryClientProvider client={client}>
      <FormProvider {...form}>
        <MatrixGridEditor rateMatrixId="rm_1" version={version} />
      </FormProvider>
    </QueryClientProvider>
  );
}

function cell(fields: Partial<RateMatrixCell>): RateMatrixCell {
  return {
    d0Key: "",
    d1Key: "",
    d2Key: "",
    d3Key: "",
    d0KeyTo: "",
    d1KeyTo: "",
    d2KeyTo: "",
    d3KeyTo: "",
    value: 1,
    deficitEligible: true,
    ...fields,
  } as RateMatrixCell;
}

const CELLS = [
  cell({ id: "c1", d0Key: "100", d0KeyTo: "149", d1Min: 0, d1Max: 500, value: 2 }),
  cell({ id: "c2", d0Key: "100", d0KeyTo: "149", d1Min: 500, value: 3 }),
];

describe("MatrixGridEditor postal ranges", () => {
  beforeEach(() => {
    listCells.mockReset();
    replaceCells.mockReset();
    replaceCells.mockResolvedValue(undefined);
  });

  it("reads the draft's cells and shows each range as a row named by both ends", async () => {
    listCells.mockResolvedValue(CELLS);

    render(<Harness />);

    expect(await screen.findByRole("rowheader", { name: "100 through 149" })).toBeInTheDocument();
    expect(listCells).toHaveBeenCalledWith("rm_1", null, expect.anything());
    const ranges = screen.getByRole("list", { name: "Origin: postal ranges" });
    expect(within(ranges).getByText("100 through 149")).toBeInTheDocument();
  });

  it("adds a range as a row of blank cells and holds the save until each holds a rate", async () => {
    listCells.mockResolvedValue(CELLS);

    render(<Harness />);
    await screen.findByRole("rowheader", { name: "100 through 149" });

    fireEvent.change(screen.getAllByLabelText("First postal code of Origin").at(-1)!, {
      target: { value: " 150" },
    });
    fireEvent.change(screen.getAllByLabelText("Last postal code of Origin").at(-1)!, {
      target: { value: "199 " },
    });
    fireEvent.click(screen.getByRole("button", { name: "Add range" }));

    expect(screen.getByRole("rowheader", { name: "150 through 199" })).toBeInTheDocument();
    expect(screen.getByText("2 cells are not numbers.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save rates" })).toBeDisabled();

    fireEvent.change(screen.getByLabelText("Value at 150 through 199, 0 – under 500"), {
      target: { value: "4" },
    });
    fireEvent.change(screen.getByLabelText("Value at 150 through 199, 500 and over"), {
      target: { value: "5" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Save rates" }));

    await waitFor(() => expect(replaceCells).toHaveBeenCalledTimes(1));
    const saved = replaceCells.mock.calls[0]?.[1] as RateMatrixCell[];
    expect(saved.map((entry) => [entry.d0Key, entry.d0KeyTo, entry.d1Min, entry.value])).toEqual([
      ["100", "149", 0, 2],
      ["100", "149", 500, 3],
      ["150", "199", 0, 4],
      ["150", "199", 500, 5],
    ]);
  });

  it("refuses a range that overlaps another before it is added", async () => {
    listCells.mockResolvedValue(CELLS);

    render(<Harness />);
    await screen.findByRole("rowheader", { name: "100 through 149" });

    fireEvent.change(screen.getAllByLabelText("First postal code of Origin").at(-1)!, {
      target: { value: "140" },
    });
    fireEvent.change(screen.getAllByLabelText("Last postal code of Origin").at(-1)!, {
      target: { value: "160" },
    });

    expect(
      screen.getByText(
        "This range overlaps 100 through 149, so a postal code in both would match twice.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add range" })).toBeDisabled();
  });

  it("removes a range's cells from every sheet", async () => {
    listCells.mockResolvedValue([
      ...CELLS,
      cell({ id: "c3", d0Key: "150", d0KeyTo: "199", d1Min: 0, d1Max: 500, value: 4 }),
    ]);

    render(<Harness />);
    await screen.findByRole("rowheader", { name: "150 through 199" });

    fireEvent.click(screen.getByRole("button", { name: "Remove 100 through 149" }));

    expect(screen.queryByRole("rowheader", { name: "100 through 149" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Save rates" }));
    await waitFor(() => expect(replaceCells).toHaveBeenCalledTimes(1));
    const [, kept] = replaceCells.mock.calls[0] as [string, RateMatrixCell[]];
    expect(kept.map((entry) => entry.id)).toEqual(["c3"]);
  });

  it("shows a published version read-only, with that version's axes", async () => {
    listCells.mockResolvedValue(CELLS);

    render(
      <Harness
        version={{ versionNumber: 2, dimensions: POSTAL_BY_WEIGHT, valueType: "Money/cwt" }}
      />,
    );

    const value = await screen.findByDisplayValue("2");
    expect(value).toHaveAttribute("readonly");
    expect(listCells).toHaveBeenCalledWith("rm_1", 2, expect.anything());
    expect(screen.queryByRole("button", { name: "Save rates" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Add range" })).toBeNull();
    expect(
      screen.getByText(
        "Version 2 is published and cannot change. Template versions that pin it keep reading these rates.",
      ),
    ).toBeInTheDocument();
  });
});
