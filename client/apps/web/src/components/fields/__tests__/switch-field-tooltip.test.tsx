import { cleanup, render, screen } from "@testing-library/react";
import { useForm } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { SwitchField } from "../switch-field";

afterEach(cleanup);

type Values = { enabled: boolean };

function Harness({ tooltip, recommended }: { tooltip?: string; recommended?: boolean }) {
  const form = useForm<Values>({ defaultValues: { enabled: false } });
  return (
    <SwitchField
      control={form.control}
      name="enabled"
      label="Capture jurisdiction miles"
      description="Ask PC*Miler for the state report."
      tooltip={tooltip}
      recommended={recommended}
    />
  );
}

// A tooltip on a switch is a caveat: cost, side effects, what the toggle does
// not do. It must be reachable without marking the switch "recommended".

describe("SwitchField tooltip", () => {
  it("shows an info trigger named after the field when a tooltip is given", () => {
    render(<Harness tooltip="May be billed as an extra transaction per route." />);
    expect(
      screen.getByRole("button", { name: "About Capture jurisdiction miles" }),
    ).toBeInTheDocument();
  });

  it("renders no info trigger without a tooltip", () => {
    render(<Harness />);
    expect(screen.queryByRole("button", { name: /About/ })).not.toBeInTheDocument();
  });

  it("leaves the recommended badge in charge of the tooltip when both are set", () => {
    render(<Harness tooltip="Why it is recommended" recommended />);
    expect(screen.getByText("Recommended")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /About/ })).not.toBeInTheDocument();
  });
});
