import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TabsTab } from "@trenova/shared/components/ui/tabs";
import { describe, expect, it, vi } from "vitest";
import {
  initialsOf,
  RecordHeader,
  RecordMark,
  RecordTabs,
  TabCount,
  TabDot,
} from "../record-header";

describe("initialsOf", () => {
  it("takes the first letter of each of the first two words", () => {
    expect(initialsOf("Maria Alvarez")).toBe("MA");
    expect(initialsOf("  apex   capital funding ")).toBe("AC");
    expect(initialsOf("Triumph")).toBe("T");
  });

  it("stands a question mark in for a name with no letters", () => {
    expect(initialsOf("   ")).toBe("?");
  });
});

describe("RecordHeader", () => {
  it("names the record beside its mark, badges, facts and actions", () => {
    render(
      <RecordHeader
        mark={<RecordMark name="Maria Alvarez" />}
        title="Maria Alvarez"
        badges={<span>Active</span>}
        meta={<span>Joliet, IL</span>}
        actions={<button type="button">Add to record</button>}
      />,
    );

    expect(screen.getByRole("heading", { level: 2, name: "Maria Alvarez" })).toBeInTheDocument();
    expect(screen.getByText("MA")).toHaveAttribute("aria-hidden", "true");
    expect(screen.getByText("Active")).toBeInTheDocument();
    expect(screen.getByText("Joliet, IL")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add to record" })).toBeInTheDocument();
  });

  it("reports the tab chosen and reads a tab's dot and count", async () => {
    const user = userEvent.setup();
    const onValueChange = vi.fn();
    render(
      <RecordTabs value="overview" onValueChange={onValueChange}>
        <TabsTab value="overview">Overview</TabsTab>
        <TabsTab value="qualification">
          Qualification
          <TabDot tone="danger" label="Needs action" />
        </TabsTab>
        <TabsTab value="files">
          Files
          <TabCount>14</TabCount>
        </TabsTab>
      </RecordTabs>,
    );

    expect(screen.getByRole("img", { name: "Needs action" })).toHaveClass("bg-danger");
    expect(screen.getByRole("tab", { name: /Files/ })).toHaveTextContent("Files14");

    await user.click(screen.getByRole("tab", { name: /Qualification/ }));

    expect(onValueChange).toHaveBeenCalledWith("qualification");
  });
});
