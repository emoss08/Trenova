import { CodeLigaturesPreference } from "@/components/code-ligatures-preference";
import { useEditorPreferenceStore } from "@/stores/editor-preference-store";
import { act, render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

describe("CodeLigaturesPreference", () => {
  afterEach(() => {
    act(() => useEditorPreferenceStore.getState().setCodeLigatures(true));
    delete document.documentElement.dataset.codeLigatures;
  });

  it("leaves ligatures on by default", () => {
    render(<CodeLigaturesPreference />);

    expect(document.documentElement.dataset.codeLigatures).toBeUndefined();
  });

  it("turns ligatures off at the root when the person turns them off, and back on", () => {
    render(<CodeLigaturesPreference />);

    act(() => useEditorPreferenceStore.getState().setCodeLigatures(false));
    expect(document.documentElement.dataset.codeLigatures).toBe("off");

    act(() => useEditorPreferenceStore.getState().setCodeLigatures(true));
    expect(document.documentElement.dataset.codeLigatures).toBeUndefined();
  });
});
