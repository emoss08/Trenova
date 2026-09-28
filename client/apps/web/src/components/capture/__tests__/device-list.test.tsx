import type { CaptureDevice } from "@/lib/graphql/capture";
import { act, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DeviceList } from "../device-list";
import { captureDevice } from "./capture-graphql-server";

const NOW = 1_800_000_000;

function device(overrides: Record<string, unknown> = {}): CaptureDevice {
  return captureDevice(overrides) as unknown as CaptureDevice;
}

function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((res, rej) => {
    resolve = res;
    reject = rej;
  });
  return { promise, resolve, reject };
}

async function openRevoke(user: ReturnType<typeof userEvent.setup>) {
  await user.click(screen.getByRole("button", { name: "Revoke" }));
  return screen.findByRole("alertdialog");
}

describe("DeviceList", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  it("keeps 'Last seen' current while the page stays open, even with a fixed now", () => {
    vi.useFakeTimers();
    vi.setSystemTime(NOW * 1000);

    render(
      <DeviceList
        devices={[device({ lastSeenAt: NOW - 120 })]}
        now={NOW}
        showOwner={false}
        canRevoke={false}
        onRevoke={() => {}}
        revokingId={undefined}
      />,
    );

    expect(screen.getByText("Last seen 2m ago")).toBeInTheDocument();

    act(() => {
      vi.advanceTimersByTime(5 * 60_000);
    });

    expect(screen.getByText("Last seen 7m ago")).toBeInTheDocument();
  });

  it("keeps the revoke dialog open until the revoke settles, then closes it", async () => {
    const user = userEvent.setup();
    const pending = deferred<unknown>();
    const onRevoke = vi.fn(() => pending.promise);

    render(
      <DeviceList
        devices={[device()]}
        now={NOW}
        showOwner={false}
        canRevoke
        onRevoke={onRevoke}
        revokingId={undefined}
      />,
    );

    const dialog = await openRevoke(user);
    await user.type(within(dialog).getByLabelText("Reason"), "  Laptop was stolen  ");
    await user.click(within(dialog).getByRole("button", { name: "Revoke" }));

    expect(onRevoke).toHaveBeenCalledWith(
      expect.objectContaining({ id: "cdev_office" }),
      "Laptop was stolen",
    );
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();

    await act(async () => {
      pending.resolve(undefined);
      await pending.promise;
    });

    await waitFor(() => expect(screen.queryByRole("alertdialog")).toBeNull());
  });

  it("shows a failed revoke inside the dialog and leaves it open", async () => {
    const user = userEvent.setup();
    const onRevoke = vi.fn(() =>
      Promise.reject(
        new GraphQLRequestError({
          kind: "graphql",
          status: 200,
          message: "You do not have permission to update capture_device",
          graphQLErrors: [
            {
              message: "You do not have permission to update capture_device",
              extensions: {},
              code: "FORBIDDEN",
              type: "https://api.trenova.app/problems/authorization-error",
            },
          ],
        }),
      ),
    );

    render(
      <DeviceList
        devices={[device()]}
        now={NOW}
        showOwner={false}
        canRevoke
        onRevoke={onRevoke}
        revokingId={undefined}
      />,
    );

    const dialog = await openRevoke(user);
    await user.click(within(dialog).getByRole("button", { name: "Revoke" }));

    expect(
      await within(dialog).findByText("You do not have permission to revoke this computer."),
    ).toBeInTheDocument();
    expect(screen.getByRole("alertdialog")).toBeInTheDocument();
  });

  it("puts a reason the server rejects on the Reason field", async () => {
    const user = userEvent.setup();
    const onRevoke = vi.fn(() =>
      Promise.reject(
        new GraphQLRequestError({
          kind: "graphql",
          status: 200,
          message: "Reason must be at most 255 characters",
          graphQLErrors: [
            {
              message: "Reason must be at most 255 characters",
              extensions: {},
              code: "INVALID",
              type: "https://api.trenova.app/problems/validation-error",
              errors: [
                {
                  field: "reason",
                  code: "INVALID",
                  message: "Reason must be at most 255 characters",
                },
              ],
            },
          ],
        }),
      ),
    );

    render(
      <DeviceList
        devices={[device()]}
        now={NOW}
        showOwner={false}
        canRevoke
        onRevoke={onRevoke}
        revokingId={undefined}
      />,
    );

    const dialog = await openRevoke(user);
    await user.type(within(dialog).getByLabelText("Reason"), "Laptop was stolen");
    await user.click(within(dialog).getByRole("button", { name: "Revoke" }));

    expect(
      await within(dialog).findByText("Reason must be at most 255 characters"),
    ).toBeInTheDocument();
    expect(within(dialog).getByLabelText("Reason")).toHaveAttribute("aria-invalid", "true");
  });

  it("refuses a reason longer than the server stores, counted in bytes as the server counts", async () => {
    const user = userEvent.setup();
    const onRevoke = vi.fn(() => Promise.resolve());

    render(
      <DeviceList
        devices={[device()]}
        now={NOW}
        showOwner={false}
        canRevoke
        onRevoke={onRevoke}
        revokingId={undefined}
      />,
    );

    const dialog = await openRevoke(user);
    const reason = within(dialog).getByLabelText("Reason");
    await user.click(reason);
    await user.paste("\u00e9".repeat(128));
    await user.click(within(dialog).getByRole("button", { name: "Revoke" }));

    expect(await within(dialog).findByText(/too long/)).toBeInTheDocument();
    expect(onRevoke).not.toHaveBeenCalled();
  });
});
