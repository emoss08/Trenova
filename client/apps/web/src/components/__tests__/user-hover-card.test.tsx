import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { UserHoverCard } from "../user-hover-card";

vi.mock("@trenova/shared/components/ui/hover-card", () => ({
  HoverCard: ({
    children,
    onOpenChange,
  }: {
    children: ReactNode;
    onOpenChange: (open: boolean) => void;
  }) => (
    <div>
      <button type="button" onClick={() => onOpenChange(true)}>
        open
      </button>
      {children}
    </div>
  ),
  HoverCardTrigger: ({ render: trigger }: { render: ReactNode }) => trigger,
  HoverCardContent: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));

vi.mock("@/lib/queries/user", () => ({
  user: {
    detail: (id: string) => ({
      queryKey: ["user", id],
      queryFn: async () => ({
        id,
        name: "Dana Reyes",
        username: "dreyes",
        emailAddress: "dana@example.com",
      }),
    }),
  },
}));

vi.mock("@/components/resolved-user-avatar", () => ({
  ResolvedUserAvatar: ({ userId, name }: { userId?: string; name?: string }) => (
    <span data-testid="avatar" data-user={userId} data-name={name} />
  ),
}));

describe("UserHoverCard", () => {
  it("draws the person's avatar in the app, with nothing fetched from another host", async () => {
    const user = userEvent.setup();
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    const { container } = render(
      <QueryClientProvider client={client}>
        <UserHoverCard userId="usr_1" username="dreyes" />
      </QueryClientProvider>,
    );

    await user.click(screen.getByRole("button", { name: "open" }));

    expect(await screen.findByText("Dana Reyes")).toBeInTheDocument();
    const avatar = screen.getByTestId("avatar");
    expect(avatar).toHaveAttribute("data-user", "usr_1");
    expect(avatar).toHaveAttribute("data-name", "Dana Reyes");
    expect(container.querySelector("img")).toBeNull();
  });
});
