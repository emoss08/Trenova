import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { clearCsrfToken, setCsrfToken } from "@trenova/shared/lib/api";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { CaptureRequestDialog } from "../capture-request-dialog";

// Fixtures follow the GraphQL contract: MyCaptureDevices returns
// CaptureDeviceFields, the pickers ask SelectOptions for CAPTURE_DEVICE and
// CAPTURE_PROFILE, and CreateCaptureRequest takes CreateCaptureRequestInput.

type Body = { operationName: string; variables: Record<string, unknown> };

function device(id: string, name: string, isOnline: boolean) {
  return {
    id,
    userId: "usr_1",
    name,
    machineName: name.toUpperCase(),
    windowsUser: "jdoe",
    agentVersion: "1.0.0",
    architecture: "X64",
    osVersion: "Windows 10.0.22631",
    status: "Active",
    lastSeenAt: 1_700_000_000,
    isOnline,
    lastIp: null,
    revokedAt: null,
    revokedReason: "",
    version: 1,
    createdAt: 1_700_000_000,
    sources: [],
  };
}

const OFFICE = device("cdev_office", "Office", false);
const FRONT_DESK = device("cdev_front", "Front desk", true);

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

function selectOptions(nodes: Record<string, unknown>[]) {
  return json({
    data: {
      selectOptions: {
        edges: nodes.map((node) => ({ cursor: "c", node })),
        pageInfo: { hasNextPage: false, endCursor: "c" },
        totalCount: nodes.length,
      },
    },
  });
}

function server(handlers: Partial<Record<string, (body: Body) => Response>>) {
  const bodies: Body[] = [];
  const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
    const body = JSON.parse(init?.body as string) as Body;
    bodies.push(body);
    const handler = handlers[body.operationName];
    if (handler === undefined) {
      throw new Error(`no handler for ${body.operationName}`);
    }
    return handler(body);
  });
  vi.stubGlobal("fetch", fetchMock);
  return bodies;
}

function renderDialog() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <CaptureRequestDialog
          open
          onOpenChange={() => {}}
          mode="Scan"
          kind="shipment"
          recordId="shp_1"
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("CaptureRequestDialog", () => {
  afterEach(() => {
    clearCsrfToken();
    cleanup();
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it("offers to try again when the person's computers could not be loaded", async () => {
    setCsrfToken("token");
    server({
      MyCaptureDevices: () => json({ errors: [{ message: "boom" }] }, 500),
    });

    renderDialog();

    expect(await screen.findByText("Your computers could not be loaded.")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Try again" })).toBeInTheDocument();
    expect(screen.queryByText(/No computer is set up to scan for you yet/)).toBeNull();
  });

  it("starts on the connected computer and sends the defaults as null", async () => {
    setCsrfToken("token");
    const user = userEvent.setup();
    const bodies = server({
      MyCaptureDevices: () => json({ data: { myCaptureDevices: [OFFICE, FRONT_DESK] } }),
      SelectOptions: (body) => {
        const input = body.variables.input as { resource: string };
        if (input.resource === "CAPTURE_DEVICE") {
          return selectOptions([
            {
              id: FRONT_DESK.id,
              label: FRONT_DESK.name,
              description: FRONT_DESK.machineName,
              meta: { isOnline: true, sources: [] },
            },
          ]);
        }
        return selectOptions([]);
      },
      CreateCaptureRequest: () =>
        json({ data: { createCaptureRequest: { id: "creq_1", __typename: "CaptureRequest" } } }),
    });

    renderDialog();

    await screen.findByRole("button", { name: "Start scan" });
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Start scan" })).toBeEnabled();
    });
    await user.click(screen.getByRole("button", { name: "Start scan" }));

    await waitFor(() => {
      expect(bodies.some((body) => body.operationName === "CreateCaptureRequest")).toBe(true);
    });
    const created = bodies.find((body) => body.operationName === "CreateCaptureRequest");
    expect(created?.variables.input).toEqual({
      deviceId: FRONT_DESK.id,
      mode: "Scan",
      targetType: "shipment",
      targetId: "shp_1",
      documentTypeId: null,
      profileId: null,
      sourceName: null,
    });
  });

  it("looks computers up in the CAPTURE_DEVICE select-option resource", async () => {
    setCsrfToken("token");
    const user = userEvent.setup();
    const bodies = server({
      MyCaptureDevices: () => json({ data: { myCaptureDevices: [FRONT_DESK] } }),
      SelectOptions: () =>
        selectOptions([
          {
            id: FRONT_DESK.id,
            label: FRONT_DESK.name,
            description: FRONT_DESK.machineName,
            meta: { isOnline: true, sources: [] },
          },
        ]),
    });

    renderDialog();

    await screen.findByRole("button", { name: "Start scan" });
    // Computer is the dialog's first picker; field labels are not tied to
    // their triggers anywhere in the app, so it is found by position.
    const [computer] = await screen.findAllByRole("combobox");
    await user.click(computer);

    await waitFor(() => {
      expect(
        bodies.some(
          (body) =>
            body.operationName === "SelectOptions" &&
            (body.variables.input as { resource: string }).resource === "CAPTURE_DEVICE",
        ),
      ).toBe(true);
    });
  });
});
