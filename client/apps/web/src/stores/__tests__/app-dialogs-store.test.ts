import { beforeEach, describe, expect, it } from "vitest";
import { useAppDialogsStore } from "../app-dialogs-store";

describe("app dialogs store", () => {
  beforeEach(() => {
    useAppDialogsStore.setState({ active: null });
  });

  it("opens one dialog at a time", () => {
    const { openDialog } = useAppDialogsStore.getState();
    openDialog("notifications");
    openDialog("shortcuts");

    expect(useAppDialogsStore.getState().active).toBe("shortcuts");
  });

  it("closes a dialog only when it is the one open", () => {
    const { openDialog, setDialogOpen } = useAppDialogsStore.getState();
    openDialog("notifications");
    setDialogOpen("shortcuts", false);

    expect(useAppDialogsStore.getState().active).toBe("notifications");

    setDialogOpen("notifications", false);

    expect(useAppDialogsStore.getState().active).toBeNull();
  });

  it("toggles the same dialog closed and another one open", () => {
    const { toggleDialog } = useAppDialogsStore.getState();
    toggleDialog("shortcuts");
    expect(useAppDialogsStore.getState().active).toBe("shortcuts");

    toggleDialog("notifications");
    expect(useAppDialogsStore.getState().active).toBe("notifications");

    toggleDialog("notifications");
    expect(useAppDialogsStore.getState().active).toBeNull();
  });
});
