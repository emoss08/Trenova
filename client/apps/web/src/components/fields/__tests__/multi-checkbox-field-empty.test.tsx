import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm, useWatch } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
import { MultiCheckboxField } from "../multi-checkbox-field";

afterEach(cleanup);

type Values = { tasks: string[] | null };

const OPTIONS = [
  { value: "Chat", label: "Chat" },
  { value: "Embedding", label: "Embedding", icon: <svg data-testid="trust-mark" /> },
];

function Harness({ emptyAs }: { emptyAs?: "null" | "array" }) {
  const form = useForm<Values>({ defaultValues: { tasks: ["Chat"] } });
  const tasks = useWatch({ control: form.control, name: "tasks" });

  return (
    <>
      <MultiCheckboxField
        control={form.control}
        name="tasks"
        label="Tasks"
        options={OPTIONS}
        emptyAs={emptyAs}
      />
      <output data-testid="value">{JSON.stringify(tasks)}</output>
    </>
  );
}

/**
 * Some lists treat no choice as "any" and store null; others are a plain list
 * where none chosen is an empty list. Clearing the last box writes whichever
 * the form says, never the other.
 */
describe("MultiCheckboxField empty selection", () => {
  it("stores null by default when the last choice is cleared", async () => {
    render(<Harness />);
    await userEvent.click(screen.getByRole("checkbox", { name: "Chat" }));

    expect(screen.getByTestId("value")).toHaveTextContent("null");
  });

  it("stores an empty list when the form asks for one", async () => {
    render(<Harness emptyAs="array" />);
    await userEvent.click(screen.getByRole("checkbox", { name: "Chat" }));

    expect(screen.getByTestId("value")).toHaveTextContent("[]");
  });

  it("shows an option's mark beside its label", () => {
    render(<Harness />);

    expect(screen.getByTestId("trust-mark")).toBeInTheDocument();
  });
});
