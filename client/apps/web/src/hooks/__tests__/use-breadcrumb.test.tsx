import { useBreadcrumbStore } from "@/stores/breadcrumb-store";
import { act, render, renderHook, screen } from "@testing-library/react";
import { createMemoryRouter, MemoryRouter, RouterProvider } from "react-router";
import { beforeEach, describe, expect, it } from "vitest";
import { useBreadcrumbs } from "../use-breadcrumb";
import { useBreadcrumbLabel } from "../use-breadcrumb-label";

const TEMPLATE_ID = "ft_01HZX2K3M4N5P6Q7R8S9T0V1";
const LIST_PATH = "/billing/configuration-files/formula-templates";
const EDIT_PATH = `${LIST_PATH}/${TEMPLATE_ID}/edit`;

function Trail() {
  const crumbs = useBreadcrumbs();
  return (
    <ol data-testid="trail">
      {crumbs.map((crumb) => (
        <li key={crumb.id}>{crumb.crumb}</li>
      ))}
    </ol>
  );
}

function trailText(): string[] {
  return Array.from(screen.getByTestId("trail").querySelectorAll("li")).map(
    (item) => item.textContent ?? "",
  );
}

function EditPage({ name }: { name: string | undefined }) {
  useBreadcrumbLabel(name);
  return <Trail />;
}

function renderRouter(initialPath: string, name: string | undefined) {
  const router = createMemoryRouter(
    [
      { path: LIST_PATH, element: <Trail /> },
      { path: `${LIST_PATH}/:id/edit`, element: <EditPage name={name} /> },
    ],
    { initialEntries: [initialPath] },
  );
  return { router, ...render(<RouterProvider router={router} />) };
}

beforeEach(() => {
  useBreadcrumbStore.setState({ labels: {} });
});

describe("useBreadcrumbs", () => {
  it("shows the record name for an edit page instead of Details and a repeated list name", () => {
    renderRouter(EDIT_PATH, "Base Linehaul");

    expect(trailText()).toEqual([
      "Billing Management",
      "Configuration Files",
      "Formula Templates",
      "Base Linehaul",
    ]);
  });

  it("falls back to Details while the record has not loaded", () => {
    renderRouter(EDIT_PATH, undefined);

    expect(trailText()).toEqual([
      "Billing Management",
      "Configuration Files",
      "Formula Templates",
      "Details",
    ]);
  });

  it("withdraws the record name when the page unmounts", async () => {
    const { router } = renderRouter(EDIT_PATH, "Base Linehaul");
    expect(useBreadcrumbStore.getState().labels).toEqual({ [EDIT_PATH]: "Base Linehaul" });

    await act(async () => {
      await router.navigate(LIST_PATH);
    });

    expect(useBreadcrumbStore.getState().labels).toEqual({});
    expect(trailText()).toEqual(["Billing Management", "Configuration Files", "Formula Templates"]);
  });
});

describe("useBreadcrumbLabel", () => {
  function wrapper({ children }: { children: React.ReactNode }) {
    return <MemoryRouter initialEntries={[EDIT_PATH]}>{children}</MemoryRouter>;
  }

  it("publishes the label for the current path and replaces it when it changes", () => {
    const { rerender } = renderHook(({ label }) => useBreadcrumbLabel(label), {
      wrapper,
      initialProps: { label: "Draft name" as string | undefined },
    });
    expect(useBreadcrumbStore.getState().labels).toEqual({ [EDIT_PATH]: "Draft name" });

    rerender({ label: "Saved name" });
    expect(useBreadcrumbStore.getState().labels).toEqual({ [EDIT_PATH]: "Saved name" });

    rerender({ label: undefined });
    expect(useBreadcrumbStore.getState().labels).toEqual({});
  });

  it("ignores blank labels and trims the rest", () => {
    const { rerender, unmount } = renderHook(({ label }) => useBreadcrumbLabel(label), {
      wrapper,
      initialProps: { label: "   " },
    });
    expect(useBreadcrumbStore.getState().labels).toEqual({});

    rerender({ label: "  Padded  " });
    expect(useBreadcrumbStore.getState().labels).toEqual({ [EDIT_PATH]: "Padded" });

    unmount();
    expect(useBreadcrumbStore.getState().labels).toEqual({});
  });

  it("can name an ancestor crumb by explicit path", () => {
    renderHook(() => useBreadcrumbLabel("Templates", `${LIST_PATH}/`), { wrapper });
    expect(useBreadcrumbStore.getState().labels).toEqual({ [LIST_PATH]: "Templates" });
  });

  it("does not clear a label another owner has since replaced", () => {
    const { unmount } = renderHook(() => useBreadcrumbLabel("First"), { wrapper });
    act(() => {
      useBreadcrumbStore.getState().setLabel(EDIT_PATH, "Second");
    });

    unmount();
    expect(useBreadcrumbStore.getState().labels).toEqual({ [EDIT_PATH]: "Second" });
  });
});
