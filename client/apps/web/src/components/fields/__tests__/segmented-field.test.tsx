import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm, useWatch } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { SegmentedField } from "../segmented-field";

afterEach(cleanup);

type Values = { onCap: "Next" | "Stop"; threshold: number };

function Harness({ error }: { error?: string }) {
  const form = useForm<Values>({ defaultValues: { onCap: "Next", threshold: 5 } });
  const [onCap, threshold] = useWatch({ control: form.control, name: ["onCap", "threshold"] });

  return (
    <>
      <SegmentedField
        control={form.control}
        name="onCap"
        label="At the cap"
        description="What happens when the month's cap is reached"
        options={[
          { value: "Next", label: "Hand to next" },
          { value: "Stop", label: "Stop" },
        ]}
        rules={error ? { validate: () => error } : undefined}
      />
      <SegmentedField
        control={form.control}
        name="threshold"
        label="Clean approvals"
        options={[
          { value: 5, label: "5" },
          { value: 10, label: "10" },
        ]}
      />
      <button type="button" onClick={() => void form.trigger("onCap")}>
        Check
      </button>
      <output data-testid="on-cap">{onCap}</output>
      <output data-testid="threshold">{JSON.stringify(threshold)}</output>
    </>
  );
}

/** A choice of a few, all shown at once, written straight into the form. */
describe("SegmentedField", () => {
  it("names the group by its label and marks the chosen segment", () => {
    render(<Harness />);

    expect(screen.getByRole("radiogroup", { name: "At the cap" })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: "Hand to next" })).toHaveAttribute(
      "aria-checked",
      "true",
    );
  });

  it("writes the segment picked into the form", async () => {
    render(<Harness />);
    await userEvent.click(screen.getByRole("radio", { name: "Stop" }));

    expect(screen.getByTestId("on-cap")).toHaveTextContent("Stop");
  });

  it("writes a numeric option back as a number, not as the text it is keyed by", async () => {
    render(<Harness />);
    await userEvent.click(screen.getByRole("radio", { name: "10" }));

    expect(screen.getByTestId("threshold")).toHaveTextContent(/^10$/);
  });

  it("shows the description, and the error in its place once there is one", async () => {
    render(<Harness error="Pick one" />);
    expect(screen.getByText("What happens when the month's cap is reached")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Check" }));

    expect(await screen.findByText("Pick one")).toBeInTheDocument();
  });
});
