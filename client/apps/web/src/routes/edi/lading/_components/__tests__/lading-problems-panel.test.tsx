import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { LadingProblemsPanel } from "../lading-problems-panel";
import type { Problem } from "../studio-model";

const working = {
  files: [{ path: "maps/tender-in.lading", source: "map tenderIn\n  from acme.204\n" }],
  samples: [],
};

const problem: Problem = {
  path: "maps/tender-in.lading",
  range: { from: 13, to: 21 },
  severity: "error",
  code: "unknown-guide",
  message: { message: "No guide is named acme.204" },
  fixes: [],
  related: [],
};

describe("LadingProblemsPanel", () => {
  it("says the project checks clean, with nothing to filter, when there is no finding", () => {
    render(<LadingProblemsPanel problems={[]} working={working} onOpen={() => {}} />);
    expect(screen.getByRole("heading", { name: "No problems" })).toBeTruthy();
    expect(
      screen.getByText(
        "Every file checks clean. Findings show here as you type, under the file they are in.",
      ),
    ).toBeTruthy();
    expect(screen.queryByRole("radiogroup", { name: "Severity" })).toBeNull();
  });

  it("lists a finding under its file instead of the empty state", () => {
    render(<LadingProblemsPanel problems={[problem]} working={working} onOpen={() => {}} />);
    expect(screen.queryByRole("heading", { name: "No problems" })).toBeNull();
    expect(screen.getByText("No guide is named acme.204")).toBeTruthy();
    expect(screen.getByText("Ln 2")).toBeTruthy();
  });
});
