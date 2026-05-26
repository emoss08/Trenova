import { AddressField } from "@/components/fields/address-field";
import { CheckboxField } from "@/components/fields/checkbox-field";
import { ColorField } from "@/components/fields/color-field";
import { AutoCompleteDateField, DateField } from "@/components/fields/date-field/date-field";
import { AutoCompleteDateTimeField } from "@/components/fields/date-field/datetime-field";
import { InputField } from "@/components/fields/input-field";
import { JsonEditorField } from "@/components/fields/json-editor-field";
import { MultiSelectAutocompleteField } from "@/components/fields/multi-select-field";
import { NumberField } from "@/components/fields/number-field";
import { PhoneNumberField } from "@/components/fields/phone-number-field";
import { SelectField } from "@/components/fields/select-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { SwitchField } from "@/components/fields/switch-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { TimePicker } from "@/components/fields/time-picker/time-picker";
import { Badge } from "@/components/ui/badge";
import type { Meta, StoryObj } from "@storybook/react-vite";
import { expect, userEvent, within } from "storybook/test";
import { useEffect, useState, type ReactNode } from "react";

import { QuerySeedProvider, StoryForm, StorySection } from "./form-story-utils";

type DriverOption = {
  id: string;
  name: string;
  status: "Available" | "Assigned" | "Inactive";
};

type FieldStoryValues = {
  inputValue: string;
  numberValue: number | null;
  textareaValue: string;
  selectValue: string;
  checkboxValue: boolean;
  switchValue: boolean;
  dateValue: number | null;
  autocompleteDateValue: number | null;
  dateTimeValue: number | null;
  colorValue: string;
  sensitiveValue: string;
  phoneValue: string;
  jsonValue: string;
  multiValues: string[];
  addressLine1: string;
  name: string;
  city: string;
  stateId: string;
  postalCode: string;
  placeId: string;
  longitude: number;
  latitude: number;
};

const fixedDate = Math.floor(new Date("2026-05-25T12:00:00Z").getTime() / 1000);
const fixedDateTime = Math.floor(new Date("2026-05-25T16:30:00Z").getTime() / 1000);

const driverOptions: DriverOption[] = [
  { id: "drv-001", name: "Ada Lovelace", status: "Available" },
  { id: "drv-002", name: "Grace Hopper", status: "Assigned" },
  { id: "drv-003", name: "Katherine Johnson", status: "Inactive" },
];

const emptyValues: FieldStoryValues = {
  inputValue: "",
  numberValue: null,
  textareaValue: "",
  selectValue: "",
  checkboxValue: false,
  switchValue: false,
  dateValue: null,
  autocompleteDateValue: null,
  dateTimeValue: null,
  colorValue: "",
  sensitiveValue: "",
  phoneValue: "",
  jsonValue: '{\n  "mode": "debug"\n}',
  multiValues: [],
  addressLine1: "",
  name: "",
  city: "",
  stateId: "",
  postalCode: "",
  placeId: "",
  longitude: 0,
  latitude: 0,
};

const filledValues: FieldStoryValues = {
  inputValue: "PRO-100482",
  numberValue: 1250,
  textareaValue: "Hold at dock door 12 until the receiver confirms temperature logs.",
  selectValue: "expedited",
  checkboxValue: true,
  switchValue: true,
  dateValue: fixedDate,
  autocompleteDateValue: fixedDate,
  dateTimeValue: fixedDateTime,
  colorValue: "#4682b4",
  sensitiveValue: "lane-access-token",
  phoneValue: "+14155552671",
  jsonValue: '{\n  "priority": "high",\n  "temperature": "frozen"\n}',
  multiValues: ["drv-001", "drv-002"],
  addressLine1: "1 Market St",
  name: "Ferry Building",
  city: "San Francisco",
  stateId: "CA",
  postalCode: "94105",
  placeId: "storybook-place",
  longitude: -122.393,
  latitude: 37.795,
};

const validationErrors: Record<string, string> = {
  inputValue: "Reference is required.",
  numberValue: "Amount must be greater than zero.",
  textareaValue: "Dispatch instructions are required.",
  selectValue: "Choose a service level.",
  switchValue: "Enable this control to continue.",
  autocompleteDateValue: "Pickup date is required.",
  dateTimeValue: "Appointment time is required.",
  colorValue: "Choose a route color.",
  sensitiveValue: "Secret value is required.",
  phoneValue: "Use a valid US phone number.",
  jsonValue: "JSON payload is required.",
  multiValues: "Select at least one driver.",
  addressLine1: "Address line 1 is required.",
};

const selectOptions = [
  { label: "Standard", value: "standard", description: "Normal service window" },
  { label: "Expedited", value: "expedited", description: "Priority handling", color: "#2563eb" },
  { label: "Hazmat", value: "hazmat", description: "Special compliance review", color: "#f97316" },
];

const googleMapsDisabledSeed = [
  {
    queryKey: ["integration-config", "GoogleMaps"],
    value: { enabled: false, fields: [] },
  },
];

function MockDriverOptions({ children }: { children: ReactNode }) {
  useEffect(() => {
    const originalFetch = window.fetch;

    window.fetch = async (input) => {
      const url = input instanceof Request ? input.url : String(input);
      const option = driverOptions.find((driver) => url.endsWith(`/${driver.id}`));
      const body = option
        ? option
        : {
            results: driverOptions,
            next: null,
            count: driverOptions.length,
          };

      return new Response(JSON.stringify(body), {
        headers: { "content-type": "application/json" },
      });
    };

    return () => {
      window.fetch = originalFetch;
    };
  }, []);

  return children;
}

function TimePickerFieldStory() {
  const [date, setDate] = useState<Date | undefined>(new Date("2026-05-25T13:30:00"));

  return (
    <div className="grid gap-2">
      <TimePicker date={date} setDate={setDate} />
      <p className="text-2xs text-muted-foreground">
        Current value: {date ? date.toLocaleTimeString() : "Unset"}
      </p>
    </div>
  );
}

function DriverOptionRow(option: DriverOption) {
  return (
    <div className="flex w-full items-center justify-between gap-2">
      <span>{option.name}</span>
      <Badge variant={option.status === "Available" ? "active" : "secondary"}>
        {option.status}
      </Badge>
    </div>
  );
}

function FieldsCatalog({
  values,
  disabled = false,
  errors,
}: {
  values: FieldStoryValues;
  disabled?: boolean;
  errors?: Record<string, string>;
}) {
  return (
    <QuerySeedProvider seeds={googleMapsDisabledSeed}>
      <MockDriverOptions>
        <StoryForm<FieldStoryValues> defaultValues={values} forcedErrors={errors}>
          {({ control }) => (
            <>
              <StorySection title="Text and Numbers">
                <InputField
                  control={control}
                  name="inputValue"
                  label="Reference"
                  description="Shipment or billing reference."
                  placeholder="PRO number"
                  disabled={disabled}
                  readOnly={disabled}
                  rules={{ required: "Reference is required." }}
                />
                <NumberField
                  control={control}
                  name="numberValue"
                  label="Invoice Amount"
                  description="Currency-style numeric input."
                  placeholder="0.00"
                  prefix="$"
                  thousandSeparator
                  decimalScale={2}
                  fixedDecimalScale
                  min={0}
                  step={25}
                  disabled={disabled}
                  readOnly={disabled}
                  rules={{ required: "Amount is required." }}
                />
                <TextareaField
                  control={control}
                  name="textareaValue"
                  label="Dispatch Instructions"
                  description="Free-form operational notes with optional presets."
                  placeholder="Enter driver instructions"
                  disabled={disabled}
                  readOnly={disabled}
                  rules={{ required: "Instructions are required." }}
                  presets={[
                    {
                      id: "dock",
                      label: "Dock hold",
                      description: "Hold at the assigned dock until release is confirmed.",
                    },
                  ]}
                />
                <SensitiveField
                  control={control}
                  name="sensitiveValue"
                  label="Sensitive Token"
                  description="Password-style field with show/hide debugging."
                  placeholder="Enter token"
                  disabled={disabled}
                  readOnly={disabled}
                  rules={{ required: "Token is required." }}
                />
              </StorySection>

              <StorySection title="Choices">
                <SelectField
                  control={control}
                  name="selectValue"
                  label="Service Level"
                  description="Searchable command-backed select field."
                  placeholder="Select service"
                  options={selectOptions}
                  isClearable
                  isReadOnly={disabled}
                  rules={{ required: "Service level is required." }}
                />
                <MultiSelectAutocompleteField<DriverOption, FieldStoryValues>
                  control={control}
                  name="multiValues"
                  label="Drivers"
                  description="Async multi-select with fetched options."
                  link="/storybook/drivers/"
                  preload
                  placeholder="Select drivers"
                  getDisplayValue={(option) => option.name}
                  getOptionValue={(option) => option.id}
                  renderOption={DriverOptionRow}
                  disabled={disabled}
                  readOnly={disabled}
                  rules={{ required: "At least one driver is required." }}
                />
                <CheckboxField
                  control={control}
                  name="checkboxValue"
                  label="Require lumper receipt"
                  description="Boolean checkbox field."
                  outlined
                  disabled={disabled}
                />
                <SwitchField
                  control={control}
                  name="switchValue"
                  label="Auto-dispatch"
                  description="Toggle with validation and warning states."
                  outlined
                  readOnly={disabled}
                  disabled={disabled}
                  rules={{ required: "Auto-dispatch must be enabled." }}
                />
              </StorySection>

              <StorySection title="Dates and Time">
                <DateField
                  control={control}
                  name="dateValue"
                  label="Calendar Date"
                  description="Calendar popover date picker."
                  placeholder="Pick a date"
                  clearable
                  disabled={disabled}
                  readOnly={disabled}
                  rules={{ required: "Date is required." }}
                />
                <AutoCompleteDateField
                  control={control}
                  name="autocompleteDateValue"
                  label="Pickup Date"
                  description="Natural-language date autocomplete."
                  placeholder="Tomorrow"
                  clearable
                  readOnly={disabled}
                  rules={{ required: "Pickup date is required." }}
                />
                <AutoCompleteDateTimeField
                  control={control}
                  name="dateTimeValue"
                  label="Appointment"
                  description="Date and time autocomplete."
                  placeholder="t 0800"
                  clearable
                  disabled={disabled}
                  rules={{ required: "Appointment is required." }}
                />
                <TimePickerFieldStory />
              </StorySection>

              <StorySection title="Specialized">
                <ColorField
                  control={control}
                  name="colorValue"
                  label="Route Color"
                  description="Preset swatches plus custom value entry."
                  disabled={disabled}
                  rules={{ required: "Color is required." }}
                />
                <PhoneNumberField
                  control={control}
                  name="phoneValue"
                  label="Dispatch Phone"
                  description="US phone number formatting."
                  placeholder="(555) 555-5555"
                  disabled={disabled}
                  rules={{ required: "Phone number is required." }}
                />
                <JsonEditorField
                  control={control}
                  name="jsonValue"
                  label="Accessorial Metadata"
                  description="CodeMirror JSON editor field."
                  disabled={disabled}
                  height="120px"
                  rules={{ required: "JSON metadata is required." }}
                />
                <AddressField<FieldStoryValues>
                  control={control}
                  label="Address"
                  description="Address entry with Google lookup disabled by seeded query data."
                  disabled={disabled}
                  readOnly={disabled}
                  rules={{ required: "Address is required." }}
                />
              </StorySection>
            </>
          )}
        </StoryForm>
      </MockDriverOptions>
    </QuerySeedProvider>
  );
}

const meta = {
  title: "Fields/Form Fields",
  parameters: {
    docs: {
      description: {
        component:
          "React Hook Form field catalog for debugging controlled values, validation errors, disabled/read-only states, popovers, autocomplete, and async option loading.",
      },
    },
  },
} satisfies Meta;

export default meta;

type Story = StoryObj<typeof meta>;

export const Default: Story = {
  render: () => <FieldsCatalog values={emptyValues} />,
};

export const Filled: Story = {
  render: () => <FieldsCatalog values={filledValues} />,
};

export const DisabledAndReadOnly: Story = {
  render: () => <FieldsCatalog values={filledValues} disabled />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await expect(canvas.getByPlaceholderText("Address Line 1")).toBeDisabled();
    await expect(canvas.getByRole("button", { name: /May 25th, 2026/ })).toBeDisabled();

    const drivers = canvas.getByRole("combobox", { name: "Drivers" });
    await expect(drivers).toBeDisabled();
    await expect(within(document.body).queryByText("Katherine Johnson")).not.toBeInTheDocument();
  },
};

export const ValidationErrors: Story = {
  render: () => <FieldsCatalog values={emptyValues} errors={validationErrors} />,
};

export const InteractionDebug: Story = {
  render: () => <FieldsCatalog values={emptyValues} />,
  play: async ({ canvasElement }) => {
    const canvas = within(canvasElement);

    await userEvent.type(canvas.getByPlaceholderText("PRO number"), "PRO-9981");
    await expect(canvas.getByText(/PRO-9981/)).toBeInTheDocument();

    await userEvent.click(canvas.getByText("Select service"));
    await userEvent.click(await within(document.body).findByText("Expedited"));
    await expect(canvas.getByText(/expedited/)).toBeInTheDocument();

    await userEvent.click(canvas.getByText("Select drivers"));
    await userEvent.click(await within(document.body).findByText("Ada Lovelace"));
    await expect(canvas.getByText(/drv-001/)).toBeInTheDocument();

    await userEvent.type(canvas.getByLabelText("Pickup Date"), "t+1");
    await userEvent.keyboard("{Enter}");
    await expect(canvas.getByLabelText("Pickup Date")).not.toHaveValue("");
  },
};
