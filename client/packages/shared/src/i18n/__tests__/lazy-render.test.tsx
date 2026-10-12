import type * as Locales from "@trenova/shared/i18n/generated/locales";
import { afterCatalogs, requireCatalog, setLocale } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { act, render, screen } from "@testing-library/react";
import { lazy, Suspense } from "react";
import { describe, expect, it, vi } from "vitest";

// The shipment folder's Spanish strings, held open so the test decides when they land, the
// way a slow route chunk would.
const shipmentBundle = vi.hoisted(() => {
  let release: (messages: Record<string, string>) => void = () => undefined;
  const loaded = new Promise<Record<string, string>>((resolve) => {
    release = resolve;
  });
  return { loaded, release: () => release({ "No comments yet": "Aún no hay comentarios" }) };
});

vi.mock("@trenova/shared/i18n/generated/locales", async (importOriginal) => {
  const actual = await importOriginal<typeof Locales>();
  return {
    ...actual,
    CATALOG_LOADERS: {
      ...actual.CATALOG_LOADERS,
      es: { "routes/shipment": () => shipmentBundle.loaded },
    },
  };
});

function Comments() {
  const t = useT();
  return <p>{t("No comments yet")}</p>;
}

describe("a React.lazy component from another route folder", () => {
  it("keeps its skeleton until its strings land, then renders translated on its first frame", async () => {
    await act(() => setLocale("es"));

    const frames: string[] = [];
    function RecordedComments() {
      const t = useT();
      frames.push(t("No comments yet"));
      return <Comments />;
    }

    // What the build emits for `lazy(() => import("@/routes/shipment/_components/comments"))`:
    // the module evaluates and requires its folder's bundle, then the import waits for it.
    const LazyComments = lazy(() =>
      Promise.resolve()
        .then(() => {
          requireCatalog("routes/shipment");
          return { default: RecordedComments };
        })
        .then(afterCatalogs),
    );

    render(
      <Suspense fallback={<div role="status">Loading</div>}>
        <LazyComments />
      </Suspense>,
    );

    await act(() => new Promise((resolve) => setTimeout(resolve, 0)));
    expect(screen.getByRole("status")).toBeInTheDocument();
    expect(screen.queryByText("No comments yet")).not.toBeInTheDocument();

    await act(async () => {
      shipmentBundle.release();
      await shipmentBundle.loaded;
    });

    expect(await screen.findByText("Aún no hay comentarios")).toBeInTheDocument();
    expect(screen.queryByRole("status")).not.toBeInTheDocument();
    expect(frames).not.toContain("No comments yet");
  });
});
