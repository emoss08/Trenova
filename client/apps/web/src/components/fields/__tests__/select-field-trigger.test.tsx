import { cleanup, render, screen } from "@testing-library/react";
import { useForm } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { SelectField } from "../select-field";

afterEach(cleanup);

type Values = { provider: string };

function Harness({ value }: { value: string }) {
  const form = useForm<Values>({ defaultValues: { provider: value } });

  return (
    <SelectField
      control={form.control}
      name="provider"
      label="Preferred provider"
      placeholder="Automatic"
      options={[
        {
          value: "with-mark",
          label: "Local qwen",
          icon: <svg data-testid="vendor-mark" />,
        },
        { value: "with-colour", label: "Draft", color: "#d97706" },
        { value: "plain", label: "Nothing special" },
      ]}
    />
  );
}

/**
 * An option's mark is the reason the list carries one. A picker that shows the
 * vendor's logo while open and a bare line of text once closed makes the person
 * re-open it to confirm what they chose, so whatever identifies an option in
 * the list identifies it on the field too.
 */
describe("SelectField trigger", () => {
  it("shows the selected option's mark on the closed field", () => {
    render(<Harness value="with-mark" />);

    expect(screen.getByTestId("vendor-mark")).toBeInTheDocument();
    expect(screen.getByText("Local qwen")).toBeInTheDocument();
  });

  it("falls back to the colour dot for an option with no mark", () => {
    const { container } = render(<Harness value="with-colour" />);

    expect(screen.queryByTestId("vendor-mark")).not.toBeInTheDocument();
    expect(container.querySelector('[style*="background-color"]')).not.toBeNull();
  });

  it("shows neither for a plain option", () => {
    const { container } = render(<Harness value="plain" />);

    expect(screen.queryByTestId("vendor-mark")).not.toBeInTheDocument();
    expect(container.querySelector('[style*="background-color"]')).toBeNull();
    expect(screen.getByText("Nothing special")).toBeInTheDocument();
  });

  it("shows the placeholder when nothing is selected", () => {
    render(<Harness value="" />);

    expect(screen.queryByTestId("vendor-mark")).not.toBeInTheDocument();
    expect(screen.getByText("Automatic")).toBeInTheDocument();
  });
});
