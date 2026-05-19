import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ControlledSelectField } from "../designer/components/designer-fields";
import {
  messageStatusOptions,
  templateElementSourceOptions,
  validationModeOptions,
} from "../designer/utils/edi-designer-options";
import { transformOperationDefinitions } from "../designer/utils/edi-designer-utils";

function ControlledSelectHarness({
  initialValue = "",
  options = messageStatusOptions,
  onChange,
}: {
  initialValue?: string;
  options?: React.ComponentProps<typeof ControlledSelectField>["options"];
  onChange?: (value: string) => void;
}) {
  const [value, setValue] = React.useState(initialValue);
  const [renderCount, setRenderCount] = React.useState(0);

  return (
    <div>
      <button type="button" onClick={() => setRenderCount((current) => current + 1)}>
        Force rerender
      </button>
      <button type="button" onClick={() => setValue("Generated")}>
        Set generated
      </button>
      <ControlledSelectField
        label="Status"
        value={value}
        onValueChange={(nextValue) => {
          setValue(nextValue);
          onChange?.(nextValue);
        }}
        options={options}
        placeholder="All statuses"
      />
      <output aria-label="selected value">{value}</output>
      <output aria-label="render count">{renderCount}</output>
    </div>
  );
}

async function chooseOption(label: string) {
  const user = userEvent.setup();
  await user.click(getSelectTrigger());
  await user.click(await screen.findByText(label));
}

function getSelectTrigger() {
  const trigger = screen
    .getAllByRole("button")
    .find((button) => button.getAttribute("aria-haspopup") === "dialog");
  if (!trigger) throw new Error("Select trigger was not rendered");
  return trigger;
}

describe("ControlledSelectField", () => {
  afterEach(() => {
    cleanup();
  });

  it("reflects external value updates", async () => {
    const user = userEvent.setup();

    render(<ControlledSelectHarness />);

    expect(screen.getByRole("button", { name: /all statuses/i })).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Set generated" }));

    expect(getSelectTrigger()).toHaveTextContent("Generated");
    expect(screen.getByLabelText("selected value")).toHaveTextContent("Generated");
  });

  it("does not call onValueChange for parent rerenders or external value sync", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();

    render(<ControlledSelectHarness onChange={onChange} />);

    await user.click(screen.getByRole("button", { name: "Force rerender" }));
    await user.click(screen.getByRole("button", { name: "Set generated" }));

    expect(onChange).not.toHaveBeenCalled();
    expect(getSelectTrigger()).toHaveTextContent("Generated");
  });

  it("selects a new value and clears to an empty string", async () => {
    const onChange = vi.fn();

    render(<ControlledSelectHarness onChange={onChange} />);

    expect(onChange).not.toHaveBeenCalled();

    await chooseOption("Failed");

    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenLastCalledWith("Failed");
    expect(screen.getByLabelText("selected value")).toHaveTextContent("Failed");

    const clearLabel = screen.getByText("Clear");
    fireEvent.click(clearLabel.parentElement as HTMLElement);

    expect(onChange).toHaveBeenCalledTimes(2);
    expect(onChange).toHaveBeenLastCalledWith("");
    expect(onChange.mock.calls).toEqual([["Failed"], [""]]);
    expect(screen.getByLabelText("selected value")).toBeEmptyDOMElement();
  });

  it("renders representative EDI option sets", async () => {
    const cases = [
      { name: "status filter", options: messageStatusOptions, expected: "Generated" },
      {
        name: "element source type",
        options: templateElementSourceOptions,
        expected: "Partner Setting",
      },
      { name: "validation mode", options: validationModeOptions, expected: "Warn Only" },
      {
        name: "transform operation",
        options: transformOperationDefinitions.map((definition) => ({
          value: definition.operation,
          label: definition.label,
        })),
        expected: "Replace",
      },
    ];

    for (const testCase of cases) {
      const { unmount } = render(
        <ControlledSelectHarness options={testCase.options} initialValue="" />,
      );

      await userEvent.click(getSelectTrigger());

      expect(
        within(await screen.findByRole("listbox")).getByText(testCase.expected),
        testCase.name,
      ).toBeInTheDocument();

      unmount();
      cleanup();
    }
  });
});
