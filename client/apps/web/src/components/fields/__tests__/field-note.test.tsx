import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { useForm } from "react-hook-form";
import { describe, expect, it } from "vitest";
import { FieldLayoutProvider, FieldNote } from "../field-components";
import { InputField } from "../input-field";

function Field({ description, error }: { description?: ReactNode; error?: string }) {
  const form = useForm({
    defaultValues: { npi: "" },
    errors: error ? { npi: { type: "server", message: error } } : undefined,
  });
  return (
    <InputField control={form.control} name="npi" label="Examiner NPI" description={description} />
  );
}

describe("a field's description", () => {
  it("is tied to its control by id when it is plain text", () => {
    render(<Field description="10 digits" />);

    const note = screen.getByText("10 digits");
    expect(note.id).toBe("input-npi-description");
    expect(screen.getByLabelText("Examiner NPI").getAttribute("aria-describedby")).toBe(note.id);
  });

  it("carries a tone and announces its changes politely when it is a FieldNote", () => {
    render(
      <Field
        description={
          <FieldNote tone="danger" live>
            Expired 3 days ago
          </FieldNote>
        }
      />,
    );

    const note = screen.getByText("Expired 3 days ago");
    expect(note.id).toBe("input-npi-description");
    expect(note.className).toContain("text-danger-foreground");
    expect(note.getAttribute("aria-live")).toBe("polite");
    expect(screen.getByLabelText("Examiner NPI").getAttribute("aria-describedby")).toBe(note.id);
  });

  it("keeps its tone over a settings row's description type", () => {
    render(
      <FieldLayoutProvider layout="row">
        <Field description={<FieldNote tone="warning">Due in 12 days</FieldNote>} />
      </FieldLayoutProvider>,
    );

    const note = screen.getByText("Due in 12 days");
    expect(note.className).toContain("text-warning-foreground");
    expect(note.className).not.toContain("text-muted-foreground");
    expect(note.className).toContain("text-xs");
  });

  it("gives way to the field's error, and the control is described by the error", () => {
    render(<Field description="10 digits" error="Enter 10 digits" />);

    expect(screen.queryByText("10 digits")).toBeNull();
    expect(screen.getByLabelText("Examiner NPI").getAttribute("aria-describedby")).toBe(
      "input-npi-error",
    );
    expect(screen.getByRole("alert").id).toBe("input-npi-error");
  });
});
