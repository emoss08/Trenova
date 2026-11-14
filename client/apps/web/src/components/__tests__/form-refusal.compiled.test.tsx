import { MultiCheckboxField } from "@/components/fields/multi-checkbox-field";
import { InputField } from "@/components/fields/input-field";
import { handleMutationError } from "@/hooks/use-api-mutation";
import { Form } from "@trenova/shared/components/ui/form";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { FormProvider, useForm, type UseFormReturn } from "react-hook-form";
import { beforeEach, describe, expect, it, vi } from "vitest";

// Runs under the React Compiler, as the app is built. The form object react-hook-form hands
// out is the same from one render to the next, so an alert that read its errors off that
// object was memoized on it and never drawn: a refused save showed nothing at the head of
// the form. Under the plain unit project that cannot be seen.

const toastError = vi.hoisted(() => vi.fn());
vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { error: toastError, success: vi.fn() },
}));

type Control = {
  journalPostingMode: string;
  autoPostSourceEvents: string[];
  defaultExpenseAccountId: string;
};

const EVENTS = [
  { value: "InvoicePosted", label: "Invoice posted" },
  { value: "VendorBillPosted", label: "Vendor bill posted" },
];

// The refusal the accounting control save returns, as the server writes it: several
// messages for one field, and one for a field this form does not draw.
const REFUSAL = new ApiRequestError(400, {
  type: "https://api.trenova.app/problems/validation-error",
  title: "Validation Failed",
  status: 400,
  detail: "At least one auto-post source event is required",
  instance: "/api/v1/accounting-controls/",
  code: "INVALID",
  errors: [
    {
      path: ["autoPostSourceEvents"],
      message: "At least one auto-post source event is required",
      code: "REQUIRED",
      location: "body",
    },
    {
      path: ["autoPostSourceEvents"],
      message: "Auto-post source events must include InvoicePosted",
      code: "INVALID_OPERATION",
      location: "body",
    },
    {
      path: ["defaultExpenseAccountId"],
      message: "Default expense account is required",
      code: "REQUIRED",
      location: "body",
    },
    {
      path: ["defaultApAccountId"],
      message: "Default AP account is required",
      code: "REQUIRED",
      location: "body",
    },
  ],
} as ConstructorParameters<typeof ApiRequestError>[1]);

let held: UseFormReturn<Control>;

function ControlForm({ onSave = vi.fn() }: { onSave?: () => void }) {
  const form = useForm<Control>({
    defaultValues: {
      journalPostingMode: "Automatic",
      autoPostSourceEvents: [],
      defaultExpenseAccountId: "",
    },
  });
  remember(form);

  return (
    <FormProvider {...form}>
      <Form onSubmit={form.handleSubmit(onSave)}>
        <MultiCheckboxField
          control={form.control}
          name="autoPostSourceEvents"
          label="Auto-post source events"
          options={EVENTS}
          emptyAs="array"
        />
        <InputField
          control={form.control}
          name="defaultExpenseAccountId"
          label="Default expense account"
        />
        <button type="submit">Save changes</button>
      </Form>
    </FormProvider>
  );
}

/** The form's own alert, as opposed to the message under a field, which is an alert too. */
async function formAlert(): Promise<HTMLElement> {
  const title = await screen.findByText("This change could not be saved");
  return title.closest('[role="alert"]') as HTMLElement;
}

/** Keeps the form a test drives, as a screen keeps the one it hands its mutation. */
function remember(form: UseFormReturn<Control>) {
  held = form;
}

function refuse() {
  act(() => handleMutationError({ error: REFUSAL, form: held }));
}

beforeEach(() => {
  toastError.mockClear();
  Element.prototype.scrollIntoView = vi.fn();
});

describe("a save the server refused", () => {
  it("is said in an alert at the head of the form, with every message the server gave", async () => {
    render(<ControlForm />);
    refuse();

    const alert = await formAlert();
    expect(
      within(alert)
        .getAllByRole("listitem")
        .map((item) => item.textContent),
    ).toEqual([
      "Default AP account is required",
      "At least one auto-post source event is required",
      "Auto-post source events must include InvoicePosted",
      "Default expense account is required",
    ]);
  });

  it("is brought into view, since the form is saved from a dock far below it", async () => {
    render(<ControlForm />);
    refuse();

    const alert = await formAlert();
    expect(alert.scrollIntoView).toHaveBeenCalledWith({ block: "nearest", behavior: "smooth" });
  });

  it("puts each field's first message on the field, a group of checkboxes included", async () => {
    render(<ControlForm />);
    refuse();
    await formAlert();

    const events = screen.getByRole("group", { name: "Auto-post source events" }).parentElement!;
    expect(
      within(events).getByText("At least one auto-post source event is required"),
    ).toBeInTheDocument();
    expect(
      within(events).queryByText("Auto-post source events must include InvoicePosted"),
    ).not.toBeInTheDocument();
    expect(held.getFieldState("defaultExpenseAccountId").error?.message).toBe(
      "Default expense account is required",
    );
  });

  it("toasts only what no field on the form shows", () => {
    render(<ControlForm />);
    refuse();

    expect(toastError).toHaveBeenCalledExactlyOnceWith("This change could not be saved", {
      description: "Default AP account is required",
    });
  });

  it("is cleared from the alert when the next save is attempted", async () => {
    const onSave = vi.fn();
    render(<ControlForm onSave={onSave} />);
    refuse();
    await formAlert();

    fireEvent.click(screen.getByRole("checkbox", { name: "Invoice posted" }));
    fireEvent.submit(screen.getByRole("button", { name: "Save changes" }).closest("form")!);

    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    expect(screen.queryByText("This change could not be saved")).not.toBeInTheDocument();
  });
});
