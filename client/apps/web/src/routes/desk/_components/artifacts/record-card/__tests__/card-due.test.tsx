import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CardDue, worstFirst, type DueCheck } from "../card-due";

const check = (key: string, state: DueCheck["state"]): DueCheck => ({
  key,
  state,
  title: `${key} title`,
  detail: `${key} detail`,
});

describe("CardDue", () => {
  it("counts what needs a look beside the title", () => {
    render(
      <CardDue title="Due" checks={[check("a", "bad"), check("b", "warn"), check("c", "ok")]} />,
    );

    expect(screen.getByRole("heading", { name: "Due" })).toBeInTheDocument();
    expect(screen.getByText("2 need a look")).toHaveClass("dk-warn");
    expect(screen.getByText("a title").closest(".dk-rec-ck")).toHaveClass("dk-bad");
  });

  it("says nothing is due when every check holds", () => {
    render(<CardDue title="Due" checks={[check("a", "ok")]} />);
    expect(screen.getByText("Nothing due")).toHaveClass("dk-ok");
  });

  it("holds a row for a check still being read, and claims nothing while it waits", () => {
    const { container } = render(
      <CardDue title="Due" checks={[check("a", "ok")]} waiting="Reading the last inspection…" />,
    );

    expect(container.querySelector(".dk-rec-ck-wait")).toHaveAttribute("aria-busy");
    expect(screen.getByText("Reading the last inspection…")).toHaveClass("sr-only");
    expect(screen.queryByText("Nothing due")).toBeNull();
  });
});

describe("worstFirst", () => {
  it("never reorders checks that stand the same", () => {
    expect(worstFirst([check("x", "ok"), check("y", "ok")]).map((c) => c.key)).toEqual(["x", "y"]);
  });
});

describe("a check only worth knowing", () => {
  it("sorts below what is close and above what holds, and asks for no look", () => {
    const checks = worstFirst([
      { key: "ok", state: "ok", title: "Holds", detail: "" },
      { key: "info", state: "info", title: "Worth knowing", detail: "" },
      { key: "warn", state: "warn", title: "Close", detail: "" },
    ]);
    expect(checks.map((check) => check.key)).toEqual(["warn", "info", "ok"]);

    render(<CardDue title="Due" checks={[checks[1]!, checks[2]!]} />);
    expect(screen.getByText("Nothing due")).toBeInTheDocument();
  });
});
