import { act, fireEvent, render, screen } from "@testing-library/react";
import { FormProvider, useForm, type UseFormReturn } from "react-hook-form";
import { describe, expect, it } from "vitest";
import { CustomStepForm } from "../custom-step-form";
import type { ChecklistEditorValues } from "../editor-schema";

// Runs under the React Compiler, as the app is built. Saving a checklist resets the
// editor to what the server returned while the step stays open; the step's inputs must
// keep feeding the form after that.

type Item = ChecklistEditorValues["items"][number];

function step(custom: Partial<NonNullable<Item["custom"]>> = {}): Item {
  return {
    key: "custom_lumper",
    mode: "Required",
    custom: {
      label: "Lumper receipt",
      check: "Manual",
      documentTypeId: "",
      stepLabel: "",
      prompt: "",
      ...custom,
    },
  };
}

function Editor({ onForm }: { onForm: (form: UseFormReturn<ChecklistEditorValues>) => void }) {
  const form = useForm<ChecklistEditorValues>({ defaultValues: { items: [step()] } });
  onForm(form);

  return (
    <FormProvider {...form}>
      <CustomStepForm
        index={0}
        kind="ReadyToBill"
        onDocumentPicked={() => undefined}
        onDone={() => undefined}
        onRemove={() => undefined}
      />
    </FormProvider>
  );
}

function renderStep() {
  let form!: UseFormReturn<ChecklistEditorValues>;
  render(<Editor onForm={(f) => (form = f)} />);
  return {
    form: () => form,
    name: screen.getByPlaceholderText("Lumper receipt checked"),
    prompt: screen.getByPlaceholderText("Find the lumper receipt and check it against the charge."),
  };
}

describe("CustomStepForm under the React Compiler", () => {
  it("names the step as it is typed", () => {
    const { form, name } = renderStep();

    fireEvent.change(name, { target: { value: "Lumper receipt checked" } });

    expect(form().getValues("items.0.custom.label")).toBe("Lumper receipt checked");
  });

  it("keeps taking the step's name, button and prompt after the checklist is saved", () => {
    const { form, name, prompt } = renderStep();

    act(() =>
      form().reset({ items: [step({ label: "Lumper receipt", stepLabel: "Check receipt" })] }),
    );
    expect(name).toHaveValue("Lumper receipt");
    const button = screen.getByDisplayValue("Check receipt");

    fireEvent.change(name, { target: { value: "Lumper receipt matched" } });
    fireEvent.change(button, { target: { value: "Match receipt" } });
    fireEvent.change(prompt, { target: { value: "Match the receipt to the lumper charge." } });

    expect(form().getValues("items.0.custom")).toEqual({
      label: "Lumper receipt matched",
      check: "Manual",
      documentTypeId: "",
      stepLabel: "Match receipt",
      prompt: "Match the receipt to the lumper charge.",
    });
    expect(form().getFieldState("items.0.custom.label").isDirty).toBe(true);
  });

  it("draws what is typed in the button preview", () => {
    const { name } = renderStep();

    fireEvent.change(name, { target: { value: "Scale ticket" } });

    expect(screen.getAllByText("Scale ticket").length).toBeGreaterThan(0);
  });
});
