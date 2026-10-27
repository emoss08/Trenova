import { beforeEach, describe, expect, it } from "vitest";
import { useAppDialogsStore } from "../app-dialogs-store";

describe("app dialogs store", () => {
  beforeEach(() => {
    useAppDialogsStore.setState({ active: null, prepared: [] });
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

  it("keeps every dialog that was opened or asked for prepared, once each", () => {
    const { prepareDialog, openDialog, setDialogOpen, toggleDialog } =
      useAppDialogsStore.getState();
    prepareDialog("page-access");
    prepareDialog("page-access");
    expect(useAppDialogsStore.getState()).toMatchObject({
      active: null,
      prepared: ["page-access"],
    });

    openDialog("shortcuts");
    setDialogOpen("notifications", true);
    toggleDialog("notifications");
    toggleDialog("page-access");

    expect(useAppDialogsStore.getState()).toMatchObject({
      active: "page-access",
      prepared: ["page-access", "shortcuts", "notifications"],
    });
  });
});
