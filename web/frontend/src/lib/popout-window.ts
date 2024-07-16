import { v4 as uuidv4 } from "uuid";

export type PopoutWindowOptions = {
  width?: number;
  height?: number;
  left?: number;
  top?: number;
  hideHeader?: boolean;
  hideAside?: boolean;
};

type PopoutWindowState = {
  id: string;
  window: Window | null;
  path: string;
  queryParams: Record<string, string>;
  options: PopoutWindowOptions;
};

class PopoutWindowManager {
  private static instance: PopoutWindowManager;
  private windows: Map<string, PopoutWindowState> = new Map();

  private constructor() {
    window.addEventListener("message", this.handleMessage.bind(this));
    window.addEventListener("beforeunload", this.closeAllWindows.bind(this));
  }

  static getInstance(): PopoutWindowManager {
    if (!PopoutWindowManager.instance) {
      PopoutWindowManager.instance = new PopoutWindowManager();
    }
    return PopoutWindowManager.instance;
  }

  openWindow(
    path: string,
    incomingQueryParams: Record<string, string | number | boolean> = {},
    options: PopoutWindowOptions = {},
  ): string {
    const id = uuidv4();
    const {
      width = 1280,
      height = 720,
      left = window.screen.width / 2 - 640,
      top = window.screen.height / 2 - 360,
      hideHeader = true,
      hideAside = false,
    } = options;

    const queryParams = this.sanitizeQueryParams({
      ...incomingQueryParams,
      popoutId: id,
      width: width.toString(),
      height: height.toString(),
      left: left.toString(),
      top: top.toString(),
      hideHeader: hideHeader.toString(),
      hideAside: hideAside.toString(),
    });

    const url = `${path}?${new URLSearchParams(queryParams).toString()}`;
    const popoutWindow = window.open(
      url,
      id,
      `toolbar=no, location=no, directories=no, status=no, menubar=no, scrollbars=no, resizable=yes, copyhistory=no, width=${width}, height=${height}, top=${top}, left=${left}`,
    );

    if (popoutWindow) {
      this.windows.set(id, {
        id,
        window: popoutWindow,
        path,
        queryParams,
        options,
      });
      popoutWindow.addEventListener("load", () => {
        this.sendMessage(id, "popout-ready", {});
      });
    }

    return id;
  }

  closeWindow(id: string): void {
    const windowState = this.windows.get(id);
    if (windowState && windowState.window) {
      windowState.window.close();
      this.windows.delete(id);
    }
  }

  private closeAllWindows(): void {
    this.windows.forEach((windowState) => {
      if (windowState.window) {
        windowState.window.close();
      }
    });
    this.windows.clear();
  }

  private sendMessage(id: string, type: string, data: any): void {
    const windowState = this.windows.get(id);
    if (windowState && windowState.window) {
      windowState.window.postMessage({ type, data }, "*");
    }
  }

  private handleMessage(event: MessageEvent): void {
    const { type, popoutId } = event.data;
    if (type === "popout-closed") {
      this.windows.delete(popoutId);
    }
    // Handle other message types as needed
  }

  private sanitizeQueryParams(
    params: Record<string, string | number | boolean>,
  ): Record<string, string> {
    return Object.entries(params).reduce(
      (acc, [key, value]) => {
        acc[key] = String(value);
        return acc;
      },
      {} as Record<string, string>,
    );
  }
}

export const popoutWindowManager = PopoutWindowManager.getInstance();

// Export a function to get the PopoutWindowManager instance
export function getPopoutWindowManager(): PopoutWindowManager {
  return PopoutWindowManager.getInstance();
}
