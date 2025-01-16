import { v4 as uuidv4 } from "uuid";

// Type definitions
export type PopoutWindowOptions = {
  mode?: "create" | "edit";
  recordId?: string;
  width?: number;
  height?: number;
  left?: number;
  top?: number;
  hideHeader?: boolean;
  hideAside?: boolean;
};

// Define a type for processed options where some fields remain optional
type ProcessedPopoutWindowOptions = {
  mode: "create" | "edit";
  recordId?: string;
  width: number;
  height: number;
  left: number;
  top: number;
  hideHeader: boolean;
  hideAside: boolean;
};

type PopoutWindowState = {
  id: string;
  window: Window | null;
  path: string;
  queryParams: Record<string, string>;
  options: PopoutWindowOptions;
  createdAt: Date;
  lastActive?: Date;
};

// Constants

type MessageHandler = (event: MessageEvent) => void;
type WindowEventType = "load" | "unload" | "focus" | "blur";

interface PopoutWindowEvents {
  onReady?: (windowId: string) => void;
  onClose?: (windowId: string) => void;
  onError?: (error: Error, windowId?: string) => void;
}

// Constants
const DEFAULT_OPTIONS = {
  mode: "create" as const,
  width: 1280,
  height: 720,
  hideHeader: true,
  hideAside: false,
} as const;

const WINDOW_FEATURES = [
  "toolbar=no",
  "location=no",
  "directories=no",
  "status=no",
  "menubar=no",
  "scrollbars=yes",
  "resizable=yes",
  "copyhistory=no",
] as const;

/**
 * Enterprise-grade PopoutWindowManager for managing multiple window instances
 * with robust error handling, event management, and resource cleanup.
 */
class PopoutWindowManager {
  private static instance: PopoutWindowManager;
  private readonly windows: Map<string, PopoutWindowState>;
  private readonly messageHandlers: Set<MessageHandler>;
  private readonly events: PopoutWindowEvents;
  private isInitialized: boolean;
  private readonly origin: string;

  private constructor() {
    this.windows = new Map();
    this.messageHandlers = new Set();
    this.events = {};
    this.isInitialized = false;
    this.origin = window.location.origin;

    this.initialize();
  }

  /**
   * Initialize the PopoutWindowManager with necessary event listeners
   * @private
   */
  private initialize(): void {
    if (this.isInitialized) return;

    try {
      window.addEventListener("message", this.handleMessage.bind(this));
      window.addEventListener("beforeunload", this.closeAllWindows.bind(this));
      window.addEventListener("unload", this.cleanup.bind(this));

      // Periodic cleanup of stale windows
      setInterval(this.cleanupStaleWindows.bind(this), 60000);

      this.isInitialized = true;

      if (process.env.NODE_ENV !== "production") {
        console.debug("[Trenova] PopoutWindowManager initialized");
      }
    } catch (error) {
      console.error(
        "[Trenova] Failed to initialize PopoutWindowManager:",
        error,
      );
      throw new Error("PopoutWindowManager initialization failed");
    }
  }

  /**
   * Get singleton instance of PopoutWindowManager
   */
  static getInstance(): PopoutWindowManager {
    if (!PopoutWindowManager.instance) {
      PopoutWindowManager.instance = new PopoutWindowManager();
    }
    return PopoutWindowManager.instance;
  }

  /**
   * Open a new popout window with the specified options
   */
  openWindow(
    path: string,
    incomingQueryParams: Record<string, string | number | boolean> = {},
    options: PopoutWindowOptions = {},
  ): string {
    try {
      const id = uuidv4();
      const finalOptions = this.processOptions(options);
      const queryParams = this.sanitizeQueryParams({
        ...incomingQueryParams,
        ...this.createWindowParams(id, finalOptions),
      });

      const url = this.buildWindowUrl(path, queryParams);
      const windowFeatures = this.buildWindowFeatures(finalOptions);

      const popoutWindow = window.open(url, id, windowFeatures);

      if (!popoutWindow) {
        throw new Error(
          "Failed to open popup window - it may have been blocked",
        );
      }

      this.registerWindow(id, popoutWindow, path, queryParams, finalOptions);
      this.attachWindowEvents(id, popoutWindow);

      return id;
    } catch (error) {
      console.error("[Trenova] Error opening window:", error);
      this.events.onError?.(error as Error);
      throw error;
    }
  }

  /**
   * Close a specific window by ID
   */
  closeWindow(id: string): void {
    try {
      const windowState = this.windows.get(id);
      if (windowState?.window && !windowState.window.closed) {
        windowState.window.close();
      }
      this.windows.delete(id);
      this.events.onClose?.(id);
    } catch (error) {
      console.error(`[Trenova] Error closing window ${id}:`, error);
      this.events.onError?.(error as Error, id);
    }
  }

  /**
   * Register event handlers for popout windows
   */
  on<K extends keyof PopoutWindowEvents>(
    event: K,
    handler: PopoutWindowEvents[K],
  ): void {
    this.events[event] = handler;
  }

  /**
   * Send a message to a specific window
   */
  sendMessage(id: string, type: string, data: unknown): void {
    try {
      const windowState = this.windows.get(id);
      if (windowState?.window && !windowState.window.closed) {
        windowState.window.postMessage({ type, data }, this.origin);
      }
    } catch (error) {
      console.error(`[Trenova] Error sending message to window ${id}:`, error);
      this.events.onError?.(error as Error, id);
    }
  }

  /**
   * Get all active window IDs
   */
  getActiveWindows(): string[] {
    return (
      Array.from(this.windows.entries())
        // eslint-disable-next-line @typescript-eslint/no-unused-vars
        .filter(([_, state]) => !state.window?.closed)
        .map(([id]) => id)
    );
  }

  private processOptions(
    options: PopoutWindowOptions,
  ): ProcessedPopoutWindowOptions {
    const screenCenter = {
      left:
        window.screen.width / 2 - (options.width || DEFAULT_OPTIONS.width) / 2,
      top:
        window.screen.height / 2 -
        (options.height || DEFAULT_OPTIONS.height) / 2,
    };

    return {
      ...DEFAULT_OPTIONS,
      ...options,
      left: options.left ?? screenCenter.left,
      top: options.top ?? screenCenter.top,
      width: options.width ?? DEFAULT_OPTIONS.width,
      height: options.height ?? DEFAULT_OPTIONS.height,
      hideHeader: options.hideHeader ?? DEFAULT_OPTIONS.hideHeader,
      hideAside: options.hideAside ?? DEFAULT_OPTIONS.hideAside,
    };
  }

  private createWindowParams(
    id: string,
    options: ProcessedPopoutWindowOptions,
  ) {
    return {
      mode: options.mode,
      recordId: options.recordId || "",
      popoutId: id,
      width: options.width.toString(),
      height: options.height.toString(),
      left: options.left.toString(),
      top: options.top.toString(),
      hideHeader: options.hideHeader.toString(),
      hideAside: options.hideAside.toString(),
    };
  }

  private buildWindowUrl(
    path: string,
    queryParams: Record<string, string>,
  ): string {
    const searchParams = new URLSearchParams(queryParams);
    return `${path}?${searchParams.toString()}`;
  }

  private buildWindowFeatures(options: ProcessedPopoutWindowOptions): string {
    const dynamicFeatures = [
      `width=${options.width}`,
      `height=${options.height}`,
      `top=${options.top}`,
      `left=${options.left}`,
    ];

    return [...WINDOW_FEATURES, ...dynamicFeatures].join(", ");
  }

  private registerWindow(
    id: string,
    window: Window,
    path: string,
    queryParams: Record<string, string>,
    options: PopoutWindowOptions,
  ): void {
    this.windows.set(id, {
      id,
      window,
      path,
      queryParams,
      options,
      createdAt: new Date(),
    });
  }

  private attachWindowEvents(id: string, window: Window): void {
    const events: WindowEventType[] = ["load", "unload", "focus", "blur"];

    events.forEach((eventType) => {
      window.addEventListener(eventType, () => {
        const windowState = this.windows.get(id);
        if (windowState) {
          windowState.lastActive = new Date();
          if (eventType === "load") {
            this.sendMessage(id, "popout-ready", {});
            this.events.onReady?.(id);
          }
        }
      });
    });
  }

  private handleMessage(event: MessageEvent): void {
    try {
      if (event.origin !== this.origin) return;

      const { type, popoutId } = event.data;

      if (type === "popout-closed") {
        this.closeWindow(popoutId);
      }

      this.messageHandlers.forEach((handler) => handler(event));
    } catch (error) {
      console.error("[Trenova] Error handling message:", error);
      this.events.onError?.(error as Error);
    }
  }

  private cleanupStaleWindows(): void {
    const now = new Date();
    for (const [id, state] of this.windows.entries()) {
      if (
        state.window?.closed ||
        (state.lastActive &&
          now.getTime() - state.lastActive.getTime() > 8 * 60 * 60 * 1000) // 8 hours
      ) {
        this.closeWindow(id);
      }
    }
  }

  private closeAllWindows(): void {
    this.windows.forEach((_, id) => this.closeWindow(id));
  }

  private cleanup(): void {
    this.closeAllWindows();
    window.removeEventListener("message", this.handleMessage);
    window.removeEventListener("beforeunload", this.closeAllWindows);
    window.removeEventListener("unload", this.cleanup);
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

// Export singleton instance
export const popoutWindowManager = PopoutWindowManager.getInstance();

// Export function to get instance
export function getPopoutWindowManager(): PopoutWindowManager {
  return PopoutWindowManager.getInstance();
}
