import { describe, expect, it } from "vitest";
import { extensionGlow } from "../extension-tile";

/*
The types are the server's agentextension.Type values
(services/tms/internal/modules/agent/domain/agentextension/enums.go):
TypeBrowserUse "BrowserUse" and TypeParallel "Parallel".
*/

describe("extensionGlow", () => {
  it("glows Browser Use orange", () => {
    expect(extensionGlow("BrowserUse")).toEqual({
      from: "var(--accent-orange)",
      to: "var(--accent-amber)",
    });
  });

  it("glows Parallel violet", () => {
    expect(extensionGlow("Parallel")).toEqual({
      from: "var(--accent-violet)",
      to: "var(--accent-indigo)",
    });
  });

  it("glows any other extension in the brand's colours", () => {
    expect(extensionGlow("Samsara")).toEqual({
      from: "var(--brand)",
      to: "var(--accent-sky)",
    });
  });
});
