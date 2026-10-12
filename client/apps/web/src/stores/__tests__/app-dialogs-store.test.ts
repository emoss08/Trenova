import { beforeEach, describe, expect, it } from "vitest";
import { useAppDialogsStore } from "../app-dialogs-store";

describe("app dialogs store", () => {
  beforeEach(() => {
    useAppDialogsStore.setState({ active: null, settingsSection: null });
  });

  it("opens one dialog at a time", () => {
    const { openDialog } = useAppDialogsStore.getState();
    openDialog("settings");
    openDialog("shortcuts");

    expect(useAppDialogsStore.getState().active).toBe("shortcuts");
  });

  it("closes a dialog only when it is the one open", () => {
    const { openDialog, setDialogOpen } = useAppDialogsStore.getState();
    openDialog("notifications");
    setDialogOpen("settings", false);

    expect(useAppDialogsStore.getState().active).toBe("notifications");

    setDialogOpen("notifications", false);

    expect(useAppDialogsStore.getState().active).toBeNull();
  });

  it("toggles the same dialog closed and another one open", () => {
    const { toggleDialog } = useAppDialogsStore.getState();
    toggleDialog("shortcuts");
    expect(useAppDialogsStore.getState().active).toBe("shortcuts");

    toggleDialog("shortcuts");
    expect(useAppDialogsStore.getState().active).toBeNull();
  });

  it("opens settings at a section and forgets it once cleared", () => {
    const { openDialog, clearSettingsSection } = useAppDialogsStore.getState();
    openDialog("settings", { section: "notifications" });

    expect(useAppDialogsStore.getState()).toMatchObject({
      active: "settings",
      settingsSection: "notifications",
    });

    clearSettingsSection();
    expect(useAppDialogsStore.getState().settingsSection).toBeNull();
  });

  it("drops a stale section when settings opens without one or another dialog opens", () => {
    const { openDialog } = useAppDialogsStore.getState();
    openDialog("settings", { section: "security" });
    openDialog("settings");
    expect(useAppDialogsStore.getState().settingsSection).toBeNull();

    openDialog("settings", { section: "security" });
    openDialog("shortcuts", { section: "profile" });
    expect(useAppDialogsStore.getState().settingsSection).toBeNull();
  });
});
