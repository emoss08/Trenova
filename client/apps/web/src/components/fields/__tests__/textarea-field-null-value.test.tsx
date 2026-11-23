import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useForm, useWatch } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SensitiveTextareaField } from "../sensitive-textarea-field";
import { TextareaField } from "../textarea-field";

afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

type Values = { notes: string | null; secret: string | null };

function Harness() {
  const form = useForm<Values>({ defaultValues: { notes: null, secret: null } });
  const notes = useWatch({ control: form.control, name: "notes" });

  return (
    <>
      <TextareaField control={form.control} name="notes" label="Notes" />
      <SensitiveTextareaField control={form.control} name="secret" label="Private key" />
      <output data-testid="notes-value">{notes === null ? "null" : notes}</output>
    </>
  );
}

/**
 * Optional text a record has never held arrives as null (a work pane's notes default). The
 * field shows it as empty, without React warning that a textarea's value is null and without
 * turning the textarea uncontrolled.
 */
describe("a textarea field holding null", () => {
  it("shows an empty textarea and React stays silent", () => {
    const error = vi.spyOn(console, "error").mockImplementation(() => {});

    render(<Harness />);

    expect(screen.getByRole("textbox", { name: "Notes" })).toHaveValue("");
    expect(screen.getByRole("textbox", { name: "Private key" })).toHaveValue("");
    expect(error).not.toHaveBeenCalled();
  });

  it("takes what is typed into the form", () => {
    const error = vi.spyOn(console, "error").mockImplementation(() => {});
    render(<Harness />);

    fireEvent.change(screen.getByRole("textbox", { name: "Notes" }), {
      target: { value: "Called the clinic" },
    });

    expect(screen.getByRole("textbox", { name: "Notes" })).toHaveValue("Called the clinic");
    expect(screen.getByTestId("notes-value")).toHaveTextContent("Called the clinic");
    expect(error).not.toHaveBeenCalled();
  });
});
