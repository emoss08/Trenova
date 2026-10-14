import { act, fireEvent, render, screen } from "@testing-library/react";
import { useForm, type UseFormReturn } from "react-hook-form";
import { describe, expect, it } from "vitest";
import { useRegister } from "@/hooks/use-register";

// Runs under the React Compiler, as the app is built. reset() forgets every registered
// field; an input that stays on screen must be registered again by the next render, or
// what is typed into it never reaches the form.

type Preset = { name: string; note: string };

const BLANK: Preset = { name: "", note: "" };

function PresetForm({ onForm }: { onForm: (form: UseFormReturn<Preset>) => void }) {
  const form = useForm<Preset>({ defaultValues: BLANK });
  onForm(form);
  const name = useRegister(form.control, "name");

  return <input aria-label="Name" {...name} />;
}

function renderForm() {
  let form!: UseFormReturn<Preset>;
  render(<PresetForm onForm={(f) => (form = f)} />);
  return { form: () => form, input: screen.getByLabelText("Name") };
}

describe("useRegister under the React Compiler", () => {
  it("hands what is typed to the form", () => {
    const { form, input } = renderForm();

    fireEvent.change(input, { target: { value: "Late fees" } });

    expect(form().getValues("name")).toBe("Late fees");
    expect(form().getFieldState("name").isDirty).toBe(true);
  });

  it("shows the values a reset puts in", () => {
    const { form, input } = renderForm();

    act(() => form().reset({ name: "Saved preset", note: "" }));

    expect(input).toHaveValue("Saved preset");
  });

  it("still hands what is typed to the form after a reset with new values", () => {
    const { form, input } = renderForm();

    fireEvent.change(input, { target: { value: "Late fees" } });
    act(() => form().reset({ name: "Late fees", note: "" }));
    fireEvent.change(input, { target: { value: "Late fees, carriers" } });

    expect(form().getValues("name")).toBe("Late fees, carriers");
    expect(form().getFieldState("name").isDirty).toBe(true);
  });

  it("still hands what is typed to the form after a reset back to blank", () => {
    const { form, input } = renderForm();

    fireEvent.change(input, { target: { value: "Late fees" } });
    act(() => form().reset(BLANK));
    expect(input).toHaveValue("");
    fireEvent.change(input, { target: { value: "Disputed" } });

    expect(form().getValues("name")).toBe("Disputed");
    expect(form().getFieldState("name").isDirty).toBe(true);
  });

  it("validates with the rules it was given", async () => {
    let form!: UseFormReturn<Preset>;
    function RequiredName({ onForm }: { onForm: (form: UseFormReturn<Preset>) => void }) {
      const f = useForm<Preset>({ defaultValues: BLANK });
      onForm(f);
      const name = useRegister(f.control, "name", { required: "Name the preset" });
      return <input aria-label="Name" {...name} />;
    }
    render(<RequiredName onForm={(f) => (form = f)} />);

    await act(async () => {
      await form.trigger("name");
    });

    expect(form.getFieldState("name").error?.message).toBe("Name the preset");
  });
});
