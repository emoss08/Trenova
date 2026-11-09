import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { DescriptionItem, DescriptionList } from "@trenova/shared/components/ui/description-list";

afterEach(cleanup);

describe("DescriptionList", () => {
  it("draws a ruled list as cells inside one bordered box", () => {
    const { container } = render(
      <DescriptionList layout="ruled" columns={3} className="mt-2">
        <DescriptionItem label="Advance rate">93%</DescriptionItem>
        <DescriptionItem label="Reserve rate">3%</DescriptionItem>
      </DescriptionList>,
    );

    const box = container.firstElementChild as HTMLElement;
    expect(box.tagName).toBe("DIV");
    expect(box.className.split(/\s+/)).toEqual(
      expect.arrayContaining(["border", "rounded-md", "overflow-hidden", "mt-2"]),
    );
    const list = box.querySelector("dl") as HTMLElement;
    expect(list.dataset.layout).toBe("ruled");
    expect(list.className.split(/\s+/)).toEqual(expect.arrayContaining(["grid", "md:grid-cols-3"]));

    const cell = screen.getByText("Advance rate").parentElement as HTMLElement;
    expect(cell.className.split(/\s+/)).toEqual(expect.arrayContaining(["border-r", "border-b"]));
  });

  it("keeps the stacked layout a bare list", () => {
    const { container } = render(
      <DescriptionList className="mt-2">
        <DescriptionItem label="Advance rate">93%</DescriptionItem>
      </DescriptionList>,
    );

    const list = container.firstElementChild as HTMLElement;
    expect(list.tagName).toBe("DL");
    expect(list.className.split(/\s+/)).toContain("mt-2");
  });
});
