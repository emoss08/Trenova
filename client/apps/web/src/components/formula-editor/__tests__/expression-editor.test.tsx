import { buildKnownIdentifiers } from "@/components/formula-editor/known-identifiers";
import { EditorView } from "@codemirror/view";
import { render, waitFor } from "@testing-library/react";
import type { ReactCodeMirrorRef } from "@uiw/react-codemirror";
import { useEffect, useRef } from "react";
import { useForm } from "react-hook-form";
import { describe, expect, it, vi } from "vitest";
import { ExpressionEditor } from "../expression-editor";

vi.mock("@trenova/shared/components/theme-provider", () => ({
  useTheme: () => ({ theme: "system", setTheme: vi.fn() }),
}));

vi.mock("@/hooks/use-media-query", () => ({
  useMediaQuery: () => true,
}));

function Harness({
  onReady,
}: {
  onReady: (ref: React.RefObject<ReactCodeMirrorRef | null>) => void;
}) {
  const { control } = useForm<{ expression: string }>({ defaultValues: { expression: "a + 1" } });
  const ref = useRef<ReactCodeMirrorRef>(null);
  useEffect(() => {
    onReady(ref);
  }, [onReady]);
  return (
    <ExpressionEditor
      name="expression"
      control={control}
      knownIdentifiers={buildKnownIdentifiers(undefined, [])}
      editorRef={ref}
      lint={false}
    />
  );
}

describe("ExpressionEditor theme", () => {
  it("renders dark when the provider says system and the OS prefers dark", async () => {
    let captured: React.RefObject<ReactCodeMirrorRef | null> | null = null;
    render(<Harness onReady={(ref) => (captured = ref)} />);

    await waitFor(() => expect(captured?.current?.view).toBeTruthy());
    const view = captured!.current!.view!;
    expect(view.state.facet(EditorView.darkTheme)).toBe(true);
  });
});
