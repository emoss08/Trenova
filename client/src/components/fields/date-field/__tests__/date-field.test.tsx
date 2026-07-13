import { generateDateOnlyString, generateDateTimeString, toUnixTimeStamp } from "@/lib/date";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useForm, useWatch } from "react-hook-form";
import { afterEach, describe, expect, it } from "vitest";
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

afterEach(() => {
  cleanup();
});

describe("AutoCompleteDateField", () => {
  it("keeps the value cleared after the user backspaces the whole input", async () => {
    const user = userEvent.setup();
    const initialDate = new Date(2024, 0, 15);
    const expectedDisplayValue = generateDateOnlyString(initialDate);

    render(<DateFieldHarness initialDate={initialDate} />);

    const input = screen.getByRole("combobox", { name: "Date" });
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

  it("commits typed text when the input loses focus", async () => {
    const user = userEvent.setup();
    const initialDate = new Date(2024, 0, 15);

    render(<DateFieldHarness initialDate={initialDate} />);

    const input = screen.getByRole("combobox", { name: "Date" });
    await user.click(input);
    await user.keyboard("{Control>}a{/Control}01/20/2024");
    await user.tab();

    const expectedDate = new Date(2024, 0, 20);
    expect(input).toHaveValue(generateDateOnlyString(expectedDate));
    expect(screen.getByLabelText("date value")).toHaveTextContent(
      String(toUnixTimeStamp(expectedDate)),
    );
  });

  it("reverts unparseable text on blur without changing the value", async () => {
    const user = userEvent.setup();
    const initialDate = new Date(2024, 0, 15);
    const initialTimestamp = toUnixTimeStamp(initialDate);

    render(<DateFieldHarness initialDate={initialDate} />);

    const input = screen.getByRole("combobox", { name: "Date" });
    await user.click(input);
    await user.keyboard("{Control>}a{/Control}zzzz");
    await user.tab();

    expect(input).toHaveValue(generateDateOnlyString(initialDate));
    expect(screen.getByLabelText("date value")).toHaveTextContent(String(initialTimestamp));
  });

  it("reverts typed text when Escape is pressed", async () => {
    const user = userEvent.setup();
    const initialDate = new Date(2024, 0, 15);
    const initialTimestamp = toUnixTimeStamp(initialDate);

    render(<DateFieldHarness initialDate={initialDate} />);

    const input = screen.getByRole("combobox", { name: "Date" });
    await user.click(input);
    await user.keyboard("{Control>}a{/Control}01/20/2024");
    await user.keyboard("{Escape}");

    expect(input).toHaveValue(generateDateOnlyString(initialDate));
    expect(screen.getByLabelText("date value")).toHaveTextContent(String(initialTimestamp));
  });

  it("commits the highlighted suggestion with Enter", async () => {
    const user = userEvent.setup();
    const initialDate = new Date(2024, 0, 15);

    render(<DateFieldHarness initialDate={initialDate} />);

    const input = screen.getByRole("combobox", { name: "Date" });
    await user.click(input);
    await user.keyboard("{Control>}a{/Control}t+1{Enter}");

    const tomorrow = new Date();
    tomorrow.setDate(tomorrow.getDate() + 1);
    tomorrow.setHours(0, 0, 0, 0);

    expect(input).toHaveValue(generateDateOnlyString(tomorrow));
    expect(screen.getByLabelText("date value")).toHaveTextContent(
      String(toUnixTimeStamp(tomorrow)),
    );
  });
});

describe("AutoCompleteDateTimeField", () => {
  it("keeps the value cleared after the user backspaces the whole input", async () => {
    const user = userEvent.setup();
    const initialDateTime = new Date(2024, 0, 15, 9, 30);
    const expectedDisplayValue = generateDateTimeString(initialDateTime);

    render(<DateTimeFieldHarness initialDateTime={initialDateTime} />);

    const input = screen.getByRole("combobox", { name: "Date time" });
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

  it("commits typed text with an explicit time on blur", async () => {
    const user = userEvent.setup();
    const initialDateTime = new Date(2024, 0, 15, 9, 30);

    render(<DateTimeFieldHarness initialDateTime={initialDateTime} />);

    const input = screen.getByRole("combobox", { name: "Date time" });
    await user.click(input);
    await user.keyboard("{Control>}a{/Control}01/20/2024 14:45");
    await user.tab();

    const expectedDateTime = new Date(2024, 0, 20, 14, 45);
    expect(input).toHaveValue(generateDateTimeString(expectedDateTime));
    expect(screen.getByLabelText("dateTime value")).toHaveTextContent(
      String(toUnixTimeStamp(expectedDateTime)),
    );
  });
});
