import { buildKnownIdentifiers } from "@/components/formula-editor/known-identifiers";
import { fireEvent, render, waitFor } from "@testing-library/react";
import { useForm } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";
import { ActiveEditorProvider, useActiveEditorInsert } from "../active-editor";
import { ExpressionEditor } from "../expression-editor";

vi.mock("@trenova/shared/components/theme-provider", () => ({
  useTheme: () => ({ theme: "light", setTheme: vi.fn() }),
}));

function InsertButton() {
  const insert = useActiveEditorInsert();
  return (
    <button type="button" onClick={() => insert("totalDistance")}>
      insert
    </button>
  );
}

function Harness() {
  const { control } = useForm<{ main: string; line: string }>({
    defaultValues: { main: "", line: "" },
  });
  const known = buildKnownIdentifiers(undefined, []);
  return (
    <ActiveEditorProvider>
      <div data-testid="main">
        <ExpressionEditor name="main" control={control} knownIdentifiers={known} lint={false} />
      </div>
      <div data-testid="line">
        <ExpressionEditor
          name="line"
          control={control}
          knownIdentifiers={known}
          lint={false}
          variant="mini"
        />
      </div>
      <InsertButton />
    </ActiveEditorProvider>
  );
}

describe("active editor insert routing", () => {
  it("inserts into the editor that was focused last, not always the first", async () => {
    const { getByTestId, getByText } = render(<Harness />);

    await waitFor(() => expect(getByTestId("line").querySelector(".cm-content")).toBeTruthy());
    const lineContent = getByTestId("line").querySelector(".cm-content") as HTMLElement;
    fireEvent.focusIn(lineContent);

    fireEvent.click(getByText("insert"));

    await waitFor(() => {
      expect(lineContent.textContent).toContain("totalDistance");
    });
    const mainContent = getByTestId("main").querySelector(".cm-content") as HTMLElement;
    expect(mainContent.textContent).not.toContain("totalDistance");
  });
});
