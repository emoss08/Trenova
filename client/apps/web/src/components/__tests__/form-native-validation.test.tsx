import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Form } from "@trenova/shared/components/ui/form";
import {
  NumberFieldGroup,
  NumberFieldInput,
  NumberField as NumberFieldRoot,
} from "@trenova/shared/components/ui/number-field";
import { FormProvider, useForm } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";

// Base UI's NumberField mirrors its value into an aria-hidden number input carrying
// the same min/max/step, so the browser validates a control the user cannot see or
// focus. The rate agreement's simulation tab defaults its window to 365 days on a
// 30-day step, which is a step mismatch — enough to make the browser refuse the
// panel's save with "An invalid form control with name='' is not focusable".
function StepMismatchForm({ onSubmit }: { onSubmit: () => void }) {
  const form = useForm<{ note: string }>({ defaultValues: { note: "" } });

  return (
    <FormProvider {...form}>
      <Form onSubmit={onSubmit}>
        <NumberFieldRoot value={365} onValueChange={() => undefined} min={1} max={1095} step={30}>
          <NumberFieldGroup>
            <NumberFieldInput />
          </NumberFieldGroup>
        </NumberFieldRoot>
        <button type="submit">Save</button>
      </Form>
    </FormProvider>
  );
}

describe("Form native constraint validation", () => {
  it("submits even when a hidden number input fails a native constraint", async () => {
    const onSubmit = vi.fn();
    const { container } = render(<StepMismatchForm onSubmit={onSubmit} />);

    // Guards the fixture: if Base UI stops emitting the hidden input, or emits it
    // without the step, this test would pass for the wrong reason.
    const hidden = container.querySelector<HTMLInputElement>(
      'input[type="number"][aria-hidden="true"]',
    );
    expect(hidden).not.toBeNull();
    expect(hidden?.validity.stepMismatch).toBe(true);

    await userEvent.click(screen.getByRole("button", { name: "Save" }));

    expect(onSubmit).toHaveBeenCalledTimes(1);
  });
});
