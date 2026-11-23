import type { WorkerRecord } from "@/routes/worker/use-worker-state";
import { CLOSED_WORKER_OPEN_STATE } from "@/routes/worker/use-worker-state";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { Kept } from "@/components/kept";
import { useState, type ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { mariaRecord } from "../../__tests__/worker-fixtures";
import { WorkerProfileDock, type ProfileDock } from "../profile-dock";
import ProfileTab from "../profile-tab";

const mocks = vi.hoisted(() => ({
  update: vi.fn(),
  personalVisible: true,
  canUpdate: true,
  onOpen: vi.fn(),
}));

vi.mock("@/services/worker", () => ({ workerService: { update: mocks.update } }));
vi.mock("@/hooks/use-protected-fields", () => ({
  useProtectedFieldAccess: () => mocks.personalVisible,
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string) => ({
    allowed: resource === "worker" ? mocks.canUpdate : true,
  }),
}));
vi.mock("@/routes/worker/worker-areas", () => ({
  useWorkerAreas: () => ({ isDrawable: () => true }),
}));
vi.mock("@/components/work-pane", () => ({
  useWorkPane: () => ({ closeThen: (then: () => void) => then(), isOpen: false }),
}));
vi.mock("@/components/autocomplete-fields", () => ({
  FleetCodeAutocompleteField: () => null,
  JobPositionAutocompleteField: () => null,
  UsStateAutocompleteField: () => null,
}));
vi.mock("@/components/custom-fields-section", () => ({ CustomFieldsSection: () => null }));
vi.mock("../worker-home-address-field", () => ({ WorkerHomeAddressField: () => null }));
vi.mock("../profile-change-alert", () => ({ ProfileChangeAlert: () => null }));
vi.mock("@/lib/queries/worker", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/queries/worker")>();
  return {
    worker: {
      ...actual.worker,
      overview: (workerId: string) => ({
        queryKey: ["worker", "overview", workerId],
        queryFn: () => new Promise(() => undefined),
      }),
    },
  };
});

let client: QueryClient;

beforeEach(() => {
  client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  mocks.update.mockReset();
  mocks.personalVisible = true;
  mocks.canUpdate = true;
  mocks.onOpen.mockReset();
});
afterEach(cleanup);

function Wrapper({ children }: { children: ReactNode }) {
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>;
}

/**
 * The page as far as the Profile dock goes: the Profile tab kept in an Activity beside another
 * tab, and the dock the page draws from what the tab hands up, outside the tab.
 */
function ProfilePage({
  worker,
  onDirtyChange,
}: {
  worker: WorkerRecord;
  onDirtyChange: (dirty: boolean) => void;
}) {
  const [tab, setTab] = useState<"profile" | "employment">("profile");
  const [dock, setDock] = useState<ProfileDock | null>(null);
  return (
    <>
      <button type="button" onClick={() => setTab(tab === "profile" ? "employment" : "profile")}>
        Switch tab
      </button>
      <Kept shown={tab === "profile"}>
        <ProfileTab
          workerId={worker.id}
          worker={worker}
          view={null}
          open={CLOSED_WORKER_OPEN_STATE}
          onOpen={mocks.onOpen}
          shown={tab === "profile"}
          onDock={setDock}
        />
      </Kept>
      {tab === "employment" ? <p>Employment history</p> : null}
      {dock ? <WorkerProfileDock dock={dock} onDirtyChange={onDirtyChange} lifted={false} /> : null}
    </>
  );
}

function tab(worker: WorkerRecord, onDirtyChange: (dirty: boolean) => void = vi.fn()) {
  return <ProfilePage worker={worker} onDirtyChange={onDirtyChange} />;
}

const saveButton = () => screen.queryByRole("button", { name: "Save changes" });

describe("the Profile save dock, drawn by the page", () => {
  it("stays in view on another tab while Profile holds changes", async () => {
    const onDirtyChange = vi.fn();
    render(tab(mariaRecord(), onDirtyChange), { wrapper: Wrapper });

    fireEvent.change(screen.getByLabelText(/First name/), { target: { value: "Mariana" } });
    fireEvent.click(screen.getByRole("button", { name: "Switch tab" }));

    expect(await screen.findByText("Employment history")).toBeTruthy();
    expect(saveButton()).toBeTruthy();
    expect(onDirtyChange).toHaveBeenLastCalledWith(true);
  });

  it("saves the Profile form from another tab", async () => {
    mocks.update.mockImplementation(async (_id: string, values: unknown) => values);
    const onDirtyChange = vi.fn();
    render(tab(mariaRecord(), onDirtyChange), { wrapper: Wrapper });

    fireEvent.change(screen.getByLabelText(/First name/), { target: { value: "Mariana" } });
    fireEvent.click(screen.getByRole("button", { name: "Switch tab" }));
    fireEvent.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(mocks.update).toHaveBeenCalledTimes(1));
    expect(mocks.update.mock.calls[0]?.[1]).toMatchObject({ firstName: "Mariana" });
    await waitFor(() => expect(saveButton()).toBeNull());
    expect(onDirtyChange).toHaveBeenLastCalledWith(false);
    expect(mocks.onOpen).not.toHaveBeenCalled();
  });

  it("brings the person to Profile when the form refuses a save made from another tab", async () => {
    render(tab(mariaRecord()), { wrapper: Wrapper });

    fireEvent.change(screen.getByLabelText(/First name/), { target: { value: "" } });
    fireEvent.click(screen.getByRole("button", { name: "Switch tab" }));
    fireEvent.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(mocks.onOpen).toHaveBeenCalledWith({ tab: "profile" }));
    expect(mocks.update).not.toHaveBeenCalled();
  });

  it("brings the person to Profile when the server refuses a save made from another tab", async () => {
    mocks.update.mockRejectedValue(new Error("refused"));
    render(tab(mariaRecord()), { wrapper: Wrapper });

    fireEvent.change(screen.getByLabelText(/First name/), { target: { value: "Mariana" } });
    fireEvent.click(screen.getByRole("button", { name: "Switch tab" }));
    fireEvent.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(mocks.onOpen).toHaveBeenCalledWith({ tab: "profile" }));
  });

  it("goes away after Reset", async () => {
    render(tab(mariaRecord()), { wrapper: Wrapper });

    fireEvent.change(screen.getByLabelText(/First name/), { target: { value: "Mariana" } });
    fireEvent.click(await screen.findByRole("button", { name: "Reset" }));

    await waitFor(() => expect(saveButton()).toBeNull());
    expect((screen.getByLabelText(/First name/) as HTMLInputElement).value).toBe("Maria");
  });

  it("is never drawn for a reader who may not update the worker", async () => {
    mocks.canUpdate = false;
    render(tab(mariaRecord()), { wrapper: Wrapper });

    expect(screen.getByLabelText(/First name/)).toBeDisabled();
    await waitFor(() => expect(saveButton()).toBeNull());
  });
});

describe("the Profile tab", () => {
  it("saves the whole record, with the edit, through the full-record update", async () => {
    mocks.update.mockImplementation(async (_id: string, values: unknown) => values);
    const onDirtyChange = vi.fn();
    render(tab(mariaRecord(), onDirtyChange), { wrapper: Wrapper });

    fireEvent.change(screen.getByLabelText(/First name/), { target: { value: "Mariana" } });
    await waitFor(() => expect(onDirtyChange).toHaveBeenLastCalledWith(true));
    fireEvent.click(await screen.findByRole("button", { name: "Save changes" }));

    await waitFor(() => expect(mocks.update).toHaveBeenCalledTimes(1));
    const [id, sent] = mocks.update.mock.calls[0] as [string, Record<string, unknown>];
    expect(id).toBe("wrk_1");
    expect(sent).toMatchObject({
      id: "wrk_1",
      version: 3,
      firstName: "Mariana",
      lastName: "Alvarez",
      city: "Joliet",
      stateId: "st_il",
      phoneNumber: "(815) 555-0142",
      profile: expect.objectContaining({
        licenseNumber: "A123-4567-8901",
        endorsement: "X",
        hireDate: 1_662_940_800,
        version: 2,
      }),
    });
    expect(sent.profile).not.toHaveProperty("complianceStatus");
    expect(sent.profile).not.toHaveProperty("returnToDutyStatus");
    expect(sent).not.toHaveProperty("position");
    expect(sent).not.toHaveProperty("manager");
    await waitFor(() => expect(onDirtyChange).toHaveBeenLastCalledWith(false));
  });

  it("keeps the person's edits when another tab's write moves the record to a new version", async () => {
    const { rerender } = render(tab(mariaRecord()), { wrapper: Wrapper });

    fireEvent.change(screen.getByLabelText(/First name/), { target: { value: "Mariana" } });
    rerender(tab(mariaRecord({ version: 4, city: "Lockport" })));

    await waitFor(() =>
      expect((screen.getByLabelText(/^City/) as HTMLInputElement).value).toBe("Lockport"),
    );
    expect((screen.getByLabelText(/First name/) as HTMLInputElement).value).toBe("Mariana");
  });

  it("draws personal details as hidden and stops requiring them for a role without them", () => {
    mocks.personalVisible = false;
    const worker = mariaRecord();
    render(
      tab({
        ...worker,
        profile: { ...worker.profile!, licenseNumber: null, twicCardNumber: null, dob: null },
      } as WorkerRecord),
      { wrapper: Wrapper },
    );

    const licence = screen.getByLabelText(/License number/) as HTMLInputElement;
    expect(licence.disabled).toBe(true);
    expect(licence.placeholder).toBe("Hidden");
    const twic = screen.getByLabelText(/Card number/) as HTMLInputElement;
    expect(twic.disabled).toBe(true);
    expect(screen.getAllByText("Your role does not include personal details.").length).toBe(3);
  });

  it("shows status, termination and compliance as set elsewhere, not as fields", () => {
    render(tab(mariaRecord()), { wrapper: Wrapper });

    expect(screen.getByText("Status: Active.")).toBeTruthy();
    expect(screen.getByText("Termination date: not terminated.")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Record event" })).toBeTruthy();
  });

  it("asks for the hazmat expiry only for a hazmat endorsement", () => {
    const worker = mariaRecord();
    render(tab({ ...worker, profile: { ...worker.profile!, endorsement: "N" } } as WorkerRecord), {
      wrapper: Wrapper,
    });

    expect(screen.queryByLabelText(/Hazmat endorsement expires/)).toBeNull();
  });
});
