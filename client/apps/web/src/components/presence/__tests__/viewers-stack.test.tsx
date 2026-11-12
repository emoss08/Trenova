import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ViewersStack } from "../viewers-stack";

vi.mock("@/components/resolved-user-avatar", () => ({
  ResolvedUserAvatar: ({ name }: { name: string }) => <i data-testid="avatar">{name}</i>,
}));

function people(count: number) {
  return Array.from({ length: count }, (_, index) => ({
    userId: `usr_${String(index).padStart(2, "0")}`,
    name: `Person ${String(index).padStart(2, "0")}`,
  }));
}

describe("ViewersStack", () => {
  it("draws nothing when nobody else is there", () => {
    const { container } = render(<ViewersStack viewers={[]} />);

    expect(container).toBeEmptyDOMElement();
  });

  it("takes the same room for a crowd: a few avatars and a count of the rest", () => {
    render(<ViewersStack viewers={people(40)} size="sm" max={3} />);

    expect(screen.getAllByTestId("avatar").map((avatar) => avatar.textContent)).toEqual([
      "Person 00",
      "Person 01",
      "Person 02",
    ]);
    expect(screen.getByText("+37")).toBeInTheDocument();
  });

  it("names people up to a limit and counts the others", () => {
    render(<ViewersStack viewers={people(10)} label={(names) => `Also on this case: ${names}`} />);

    expect(screen.getByRole("group")).toHaveAccessibleName(
      "Also on this case: Person 00, Person 01, Person 02, Person 03, Person 04, Person 05, Person 06, Person 07 and 2 others",
    );
  });

  it("keeps each person's place whatever order they arrived in", () => {
    const [first, second] = people(2);
    render(<ViewersStack viewers={[second!, first!]} />);

    expect(screen.getAllByTestId("avatar").map((avatar) => avatar.textContent)).toEqual([
      "Person 00",
      "Person 01",
    ]);
  });
});
