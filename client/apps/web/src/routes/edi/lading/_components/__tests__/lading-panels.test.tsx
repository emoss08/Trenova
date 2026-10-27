import type { LadingCheck, LadingContext } from "@trenova/shared/edi/lading";
import type { LadingProject, LadingVersion } from "@trenova/shared/types/lading";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";
import { LadingPanels } from "../lading-panels";

// Each tab's body is a stand-in that keeps a count in its own state, so the test sees whether a
// tab's state outlives a visit to another tab. Mock factories run before the module's imports,
// so the stand-in is built in vi.hoisted from React's own exports.
const { counter } = await vi.hoisted(async () => {
  const { createElement, useState } = await import("react");
  return {
    counter: (name: string) =>
      function Counter() {
        const [runs, setRuns] = useState(0);
        return createElement(
          "button",
          { type: "button", onClick: () => setRuns((current) => current + 1) },
          `${name} ran ${runs} times`,
        );
      },
  };
});

vi.mock("../lading-problems-panel", () => ({ LadingProblemsPanel: counter("Problems") }));
vi.mock("../lading-tests-panel", () => ({ LadingTestsPanel: counter("Tests") }));
vi.mock("../lading-run-panel", () => ({ LadingRunPanel: counter("Run") }));
vi.mock("../lading-release-panel", () => ({ LadingReleasePanel: counter("Release") }));
vi.mock("../lading-settings-panel", () => ({ LadingSettingsPanel: counter("Settings") }));

function renderPanels() {
  return render(
    <LadingPanels
      project={{} as LadingProject}
      version={{} as LadingVersion}
      versions={[]}
      check={{} as LadingCheck}
      problems={[]}
      working={{ files: [], samples: [] }}
      context={{} as LadingContext}
      notes=""
      editable
      dirty={false}
      onNotesChange={() => {}}
      onOpen={() => {}}
      onValues={() => {}}
      onCoverage={() => {}}
    />,
  );
}

describe("LadingPanels", () => {
  it("keeps a tab's results when another tab is opened and the first opened again", async () => {
    const user = userEvent.setup();
    renderPanels();

    await user.click(screen.getByRole("tab", { name: "Tests" }));
    await user.click(screen.getByRole("button", { name: "Tests ran 0 times" }));
    await user.click(screen.getByRole("button", { name: "Tests ran 1 times" }));

    await user.click(screen.getByRole("tab", { name: "Run" }));
    expect(screen.queryByRole("button", { name: "Tests ran 2 times" })).toBeNull();
    await user.click(screen.getByRole("button", { name: "Run ran 0 times" }));

    await user.click(screen.getByRole("tab", { name: "Tests" }));
    expect(screen.getByRole("button", { name: "Tests ran 2 times" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Run ran 1 times" })).toBeNull();

    await user.click(screen.getByRole("tab", { name: "Run" }));
    expect(screen.getByRole("button", { name: "Run ran 1 times" })).toBeTruthy();
  });

  it("shows only the open tab's panel", async () => {
    const user = userEvent.setup();
    renderPanels();

    expect(screen.getByRole("button", { name: "Problems ran 0 times" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Settings ran 0 times" })).toBeNull();

    await user.click(screen.getByRole("tab", { name: "Settings" }));
    expect(screen.getByRole("button", { name: "Settings ran 0 times" })).toBeTruthy();
    expect(screen.queryByRole("button", { name: "Problems ran 0 times" })).toBeNull();
  });
});
