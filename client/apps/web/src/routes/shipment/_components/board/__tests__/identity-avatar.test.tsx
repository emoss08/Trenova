import { cleanup, fireEvent, render } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { IdentityAvatar } from "../identity-avatar";

afterEach(cleanup);

const PHOTO = {
  url: "https://files.example/wrk_1/photo.jpg?sig=a",
  thumbnailUrl: "https://files.example/wrk_1/thumb.jpg?sig=a",
};

describe("IdentityAvatar", () => {
  it("draws a person's lazy thumbnail over their initials, which stay underneath", () => {
    const { container } = render(
      <IdentityAvatar id="wrk_1" initials="JP" shape="circle" photo={PHOTO} />,
    );

    const image = container.querySelector("img");
    expect(image?.getAttribute("src")).toBe(PHOTO.thumbnailUrl);
    expect(image?.getAttribute("loading")).toBe("lazy");
    expect(container.textContent).toBe("JP");
  });

  it("leaves the initials alone when the photo fails", () => {
    const { container } = render(
      <IdentityAvatar id="wrk_1" initials="JP" shape="circle" photo={PHOTO} />,
    );

    fireEvent.error(container.querySelector("img") as HTMLImageElement);

    expect(container.querySelector("img")).toBeNull();
    expect(container.textContent).toBe("JP");
  });

  it("keeps a company to its initials, and a person without a photo too", () => {
    const company = render(
      <IdentityAvatar id="car_1" initials="KS" shape="square" photo={PHOTO} />,
    );
    expect(company.container.querySelector("img")).toBeNull();
    company.unmount();

    const person = render(<IdentityAvatar id="wrk_2" initials="KJ" shape="circle" photo={null} />);
    expect(person.container.querySelector("img")).toBeNull();
    expect(person.container.textContent).toBe("KJ");
  });
});
