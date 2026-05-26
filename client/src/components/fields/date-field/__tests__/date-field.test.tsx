import { generateDateOnlyString, generateDateTimeString, toUnixTimeStamp } from "@/lib/date";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm, useWatch } from "react-hook-form";
import { describe, expect, it } from "vitest";
import { AutoCompleteDateField } from "../date-field";
import { AutoCompleteDateTimeField } from "../datetime-field";

type TestFormValues = {
  date: number | null;
  dateTime: number | null;
};

function FormValue({
  control,
  name,
}: {
  control: ReturnType<typeof useForm<TestFormValues>>["control"];
  name: keyof TestFormValues;
}) {
  const value = useWatch({ control, name });

  return <output aria-label={`${name} value`}>{value ?? "null"}</output>;
}

function DateFieldHarness({ initialDate }: { initialDate: Date }) {
  const form = useForm<TestFormValues>({
    defaultValues: {
      date: toUnixTimeStamp(initialDate) ?? null,
      dateTime: null,
    },
  });

  return (
    <>
      <AutoCompleteDateField control={form.control} label="Date" name="date" />
      <FormValue control={form.control} name="date" />
    </>
  );
}

function DateTimeFieldHarness({ initialDateTime }: { initialDateTime: Date }) {
  const form = useForm<TestFormValues>({
    defaultValues: {
      date: null,
      dateTime: toUnixTimeStamp(initialDateTime) ?? null,
    },
  });

  return (
    <>
      <AutoCompleteDateTimeField control={form.control} label="Date time" name="dateTime" />
      <FormValue control={form.control} name="dateTime" />
    </>
  );
}

describe("AutoCompleteDateField", () => {
  it("keeps the value cleared after the user backspaces the whole input", async () => {
    const user = userEvent.setup();
    const initialDate = new Date(2024, 0, 15);
    const expectedDisplayValue = generateDateOnlyString(initialDate);

    render(<DateFieldHarness initialDate={initialDate} />);

    const input = screen.getByRole("textbox", { name: "Date" });
    expect(input).toHaveValue(expectedDisplayValue);

    await user.click(input);
    await user.keyboard("{Control>}a{/Control}{Backspace}");

    await waitFor(() => {
      expect(input).toHaveValue("");
      expect(screen.getByLabelText("date value")).toHaveTextContent("null");
    });

    await user.tab();
    expect(input).toHaveValue("");

    await user.click(input);
    expect(input).toHaveValue("");
  });
});

describe("AutoCompleteDateTimeField", () => {
  it("keeps the value cleared after the user backspaces the whole input", async () => {
    const user = userEvent.setup();
    const initialDateTime = new Date(2024, 0, 15, 9, 30);
    const expectedDisplayValue = generateDateTimeString(initialDateTime);

    render(<DateTimeFieldHarness initialDateTime={initialDateTime} />);

    const input = screen.getByRole("textbox", { name: "Date time" });
    expect(input).toHaveValue(expectedDisplayValue);

    await user.click(input);
    await user.keyboard("{Control>}a{/Control}{Backspace}");

    await waitFor(() => {
      expect(input).toHaveValue("");
      expect(screen.getByLabelText("dateTime value")).toHaveTextContent("null");
    });

    await user.tab();
    expect(input).toHaveValue("");

    await user.click(input);
    expect(input).toHaveValue("");
  });
});
