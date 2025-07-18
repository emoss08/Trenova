import { APP_ENV } from "@/constants/env";
import { v4 as uuidv4 } from "uuid";

// Type definitions
export type PopoutWindowOptions = {
  modalType?: "create" | "edit";
  width?: number;
  height?: number;
  left?: number;
  top?: number;
  hideHeader?: boolean;
  hideAside?: boolean;
  resizable?: boolean;
  scrollable?: boolean;
  title?: string;
  features?: WindowFeature[];
  rememberPosition?: boolean;
};

// Define a type for processed options where some fields remain optional
type ProcessedPopoutWindowOptions = {
  modalType: "create" | "edit";
  width: number;
  height: number;
  left: number;
  top: number;
  hideHeader: boolean;
  hideAside: boolean;
  resizable: boolean;
  scrollable: boolean;
  title?: string;
  features: WindowFeature[];
  rememberPosition: boolean;
};

type PopoutWindowState = {
  id: string;
  window: Window | null;
  path: string;
  queryParams: Record<string, string>;
  options: PopoutWindowOptions;
  createdAt: Date;
  lastActive?: Date;
  isFocused?: boolean;
  isMinimized?: boolean;
};

type WindowFeature =
  | "toolbar"
  | "location"
  | "directories"
  | "status"
  | "menubar"
  | "copyhistory";

type MessageType =
  | "popout-ready"
  | "popout-closed"
  | "popout-focus"
  | "popout-blur"
  | "popout-state-change"
  | "popout-error";

type PopoutMessage = {
  type: MessageType;
  popoutId?: string;
  data?: any;
  error?: string;
};

type MessageHandler = (event: MessageEvent<PopoutMessage>) => void;
type WindowEventType =
  | "load"
  | "unload"
  | "focus"
  | "blur"
  | "resize"
  | "beforeunload";

interface PopoutWindowEvents {
  onReady?: (windowId: string) => void;
  onClose?: (windowId: string) => void;
  onError?: (error: Error, windowId?: string) => void;
  onFocus?: (windowId: string) => void;
  onBlur?: (windowId: string) => void;
  onStateChange?: (windowId: string, state: Partial<PopoutWindowState>) => void;
}

type StoredWindowPosition = {
  left: number;
  top: number;
  width: number;
  height: number;
};

// Constants
const DEFAULT_OPTIONS: Partial<ProcessedPopoutWindowOptions> = {
  modalType: "create" as const,
  width: 1280,
  height: 720,
  hideHeader: true,
  hideAside: false,
  resizable: true,
  scrollable: true,
  rememberPosition: true,
  features: [],
};

const DEFAULT_WINDOW_FEATURES: Record<
  WindowFeature | "scrollbars" | "resizable",
  string
> = {
  toolbar: "toolbar=no",
  location: "location=no",
  directories: "directories=no",
  status: "status=no",
  menubar: "menubar=no",
  copyhistory: "copyhistory=no",
  scrollbars: "scrollbars=yes",
  resizable: "resizable=yes",
};

const POSITION_STORAGE_KEY = "trenova_popout_positions";
const STALE_WINDOW_TIMEOUT = 8 * 60 * 60 * 1000; // 8 hours
const CLEANUP_INTERVAL = 60000; // 1 minute

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
  private cleanupInterval?: number;
  private readonly storedPositions: Map<string, StoredWindowPosition>;
  private focusQueue: string[];

  private constructor() {
    this.windows = new Map();
    this.messageHandlers = new Set();
    this.events = {};
    this.isInitialized = false;
    this.origin = window.location.origin;
    this.storedPositions = new Map();
    this.focusQueue = [];
    this.loadStoredPositions();
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
      window.addEventListener(
        "beforeunload",
        this.handleBeforeUnload.bind(this),
      );
      window.addEventListener("unload", this.cleanup.bind(this));
      window.addEventListener("focus", this.handleParentFocus.bind(this));
      window.addEventListener("blur", this.handleParentBlur.bind(this));

      // Periodic cleanup of stale windows
      this.cleanupInterval = setInterval(
        this.cleanupStaleWindows.bind(this),
        CLEANUP_INTERVAL,
      );

      this.isInitialized = true;

      if (APP_ENV === "development") {
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
      // Check if a window with the same path already exists
      const existingWindow = this.findWindowByPath(path);
      if (existingWindow && !existingWindow.window?.closed) {
        this.focusWindow(existingWindow.id);
        return existingWindow.id;
      }

      const id = uuidv4();
      const finalOptions = this.processOptions(options, path);
      const queryParams = this.sanitizeQueryParams({
        ...incomingQueryParams,
        ...this.createWindowParams(id, finalOptions),
      });

      const url = this.buildWindowUrl(path, queryParams);
      const windowFeatures = this.buildWindowFeatures(finalOptions);

      const popoutWindow = window.open(url, id, windowFeatures);

      if (!popoutWindow) {
        throw new Error(
          "Failed to open popup window - it may have been blocked by the browser. Please check your popup blocker settings.",
        );
      }

      this.registerWindow(id, popoutWindow, path, queryParams, finalOptions);
      this.attachWindowEvents(id, popoutWindow);
      this.focusQueue.push(id);

      // Save position after window is loaded
      if (finalOptions.rememberPosition) {
        setTimeout(() => this.saveWindowPosition(id), 100);
      }

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
      if (windowState) {
        // Save position before closing if remember position is enabled
        if (windowState.options.rememberPosition) {
          this.saveWindowPosition(id);
        }

        if (windowState.window && !windowState.window.closed) {
          windowState.window.close();
        }

        this.windows.delete(id);
        this.focusQueue = this.focusQueue.filter((wId) => wId !== id);
        this.events.onClose?.(id);
      }
    } catch (error) {
      console.error(`[Trenova] Error closing window ${id}:`, error);
      this.events.onError?.(error as Error, id);
    }
  }

  /**
   * Close all windows
   */
  closeAllWindows(): void {
    const windowIds = Array.from(this.windows.keys());
    windowIds.forEach((id) => this.closeWindow(id));
  }

  /**
   * Focus a specific window
   */
  focusWindow(id: string): void {
    try {
      const windowState = this.windows.get(id);
      if (windowState?.window && !windowState.window.closed) {
        windowState.window.focus();
        windowState.isFocused = true;
        windowState.lastActive = new Date();
        this.events.onFocus?.(id);
        this.events.onStateChange?.(id, { isFocused: true });
      }
    } catch (error) {
      console.error(`[Trenova] Error focusing window ${id}:`, error);
      this.events.onError?.(error as Error, id);
    }
  }

  /**
   * Find window by path
   */
  private findWindowByPath(path: string): PopoutWindowState | undefined {
    for (const [, state] of this.windows.entries()) {
      if (state.path === path) {
        return state;
      }
    }
    return undefined;
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
  sendMessage(id: string, type: MessageType, data?: unknown): void {
    try {
      const windowState = this.windows.get(id);
      if (windowState?.window && !windowState.window.closed) {
        const message: PopoutMessage = { type, popoutId: id, data };
        windowState.window.postMessage(message, this.origin);
      }
    } catch (error) {
      console.error(`[Trenova] Error sending message to window ${id}:`, error);
      this.events.onError?.(error as Error, id);
    }
  }

  /**
   * Broadcast a message to all windows
   */
  broadcastMessage(type: MessageType, data?: unknown): void {
    this.windows.forEach((_, id) => {
      this.sendMessage(id, type, data);
    });
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

  /**
   * Get window state by ID
   */
  getWindowState(id: string): PopoutWindowState | undefined {
    return this.windows.get(id);
  }

  /**
   * Check if any windows are open
   */
  hasOpenWindows(): boolean {
    return this.getActiveWindows().length > 0;
  }

  private processOptions(
    options: PopoutWindowOptions,
    path: string,
  ): ProcessedPopoutWindowOptions {
    const width = options.width ?? DEFAULT_OPTIONS.width!;
    const height = options.height ?? DEFAULT_OPTIONS.height!;

    // Try to get stored position if remember position is enabled
    let position: { left: number; top: number };
    const storedPosition =
      options.rememberPosition !== false
        ? this.storedPositions.get(path)
        : undefined;

    if (
      storedPosition &&
      options.left === undefined &&
      options.top === undefined
    ) {
      position = {
        left: storedPosition.left,
        top: storedPosition.top,
      };
    } else if (options.left !== undefined && options.top !== undefined) {
      position = {
        left: options.left,
        top: options.top,
      };
    } else {
      // Smart positioning: cascade windows if multiple are open
      const offset = this.windows.size * 30;
      position = {
        left: window.screen.width / 2 - width / 2 + offset,
        top: window.screen.height / 2 - height / 2 + offset,
      };
    }

    // Ensure window is within screen bounds
    position = this.constrainToScreen(position, width, height);

    return {
      ...(DEFAULT_OPTIONS as ProcessedPopoutWindowOptions),
      ...options,
      left: position.left,
      top: position.top,
      width,
      height,
      hideHeader: options.hideHeader ?? DEFAULT_OPTIONS.hideHeader!,
      hideAside: options.hideAside ?? DEFAULT_OPTIONS.hideAside!,
      modalType: options.modalType ?? DEFAULT_OPTIONS.modalType!,
      resizable: options.resizable ?? DEFAULT_OPTIONS.resizable!,
      scrollable: options.scrollable ?? DEFAULT_OPTIONS.scrollable!,
      rememberPosition:
        options.rememberPosition ?? DEFAULT_OPTIONS.rememberPosition!,
      features: options.features ?? DEFAULT_OPTIONS.features!,
    };
  }

  private constrainToScreen(
    position: { left: number; top: number },
    width: number,
    height: number,
  ): { left: number; top: number } {
    const screenWidth = window.screen.availWidth;
    const screenHeight = window.screen.availHeight;

    return {
      left: Math.max(0, Math.min(position.left, screenWidth - width)),
      top: Math.max(0, Math.min(position.top, screenHeight - height)),
    };
  }

  private createWindowParams(
    id: string,
    options: ProcessedPopoutWindowOptions,
  ) {
    return {
      modalType: options.modalType,
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
    const fullPath = `${path}?${searchParams.toString()}`;

    console.log("fullPath", fullPath);
    return fullPath;
  }

  private buildWindowFeatures(options: ProcessedPopoutWindowOptions): string {
    const features: string[] = [];

    // Add default features
    Object.entries(DEFAULT_WINDOW_FEATURES).forEach(([feature, value]) => {
      if (feature === "scrollbars") {
        features.push(options.scrollable ? "scrollbars=yes" : "scrollbars=no");
      } else if (feature === "resizable") {
        features.push(options.resizable ? "resizable=yes" : "resizable=no");
      } else if (!options.features.includes(feature as WindowFeature)) {
        features.push(value);
      }
    });

    // Add custom features
    options.features.forEach((feature) => {
      features.push(`${feature}=yes`);
    });

    // Add position and size
    features.push(
      `width=${options.width}`,
      `height=${options.height}`,
      `top=${options.top}`,
      `left=${options.left}`,
    );

    return features.join(", ");
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
    const handleEvent = (eventType: WindowEventType) => {
      const windowState = this.windows.get(id);
      if (!windowState) return;

      windowState.lastActive = new Date();

      switch (eventType) {
        case "load":
          this.sendMessage(id, "popout-ready");
          this.events.onReady?.(id);
          break;
        case "focus":
          windowState.isFocused = true;
          this.events.onFocus?.(id);
          this.events.onStateChange?.(id, { isFocused: true });
          break;
        case "blur":
          windowState.isFocused = false;
          this.events.onBlur?.(id);
          this.events.onStateChange?.(id, { isFocused: false });
          break;
        case "resize":
          if (windowState.options.rememberPosition) {
            this.saveWindowPosition(id);
          }
          break;
        case "beforeunload":
          this.saveWindowPosition(id);
          break;
      }
    };

    const events: WindowEventType[] = [
      "load",
      "unload",
      "focus",
      "blur",
      "resize",
      "beforeunload",
    ];
    events.forEach((eventType) => {
      window.addEventListener(eventType, () => handleEvent(eventType));
    });

    // Monitor window position changes
    if (window && !window.closed) {
      let lastX = window.screenX;
      let lastY = window.screenY;

      const checkPosition = setInterval(() => {
        if (window.closed) {
          clearInterval(checkPosition);
          return;
        }

        if (window.screenX !== lastX || window.screenY !== lastY) {
          lastX = window.screenX;
          lastY = window.screenY;
          const currentWindowState = this.windows.get(id);
          if (currentWindowState?.options.rememberPosition) {
            this.saveWindowPosition(id);
          }
        }
      }, 500);
    }
  }

  private handleMessage(event: MessageEvent<PopoutMessage>): void {
    try {
      if (event.origin !== this.origin) return;

      const message = event.data;
      if (!message?.type) return;

      const { type, popoutId, data, error } = message;

      switch (type) {
        case "popout-closed":
          if (popoutId) {
            this.closeWindow(popoutId);
          }
          break;
        case "popout-focus":
          if (popoutId) {
            const windowState = this.windows.get(popoutId);
            if (windowState) {
              windowState.isFocused = true;
              this.events.onFocus?.(popoutId);
            }
          }
          break;
        case "popout-blur":
          if (popoutId) {
            const windowState = this.windows.get(popoutId);
            if (windowState) {
              windowState.isFocused = false;
              this.events.onBlur?.(popoutId);
            }
          }
          break;
        case "popout-state-change":
          if (popoutId && data) {
            this.events.onStateChange?.(popoutId, data);
          }
          break;
        case "popout-error":
          if (error) {
            console.error(`[Trenova] Error from popout ${popoutId}:`, error);
            this.events.onError?.(new Error(error), popoutId);
          }
          break;
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
          now.getTime() - state.lastActive.getTime() > STALE_WINDOW_TIMEOUT)
      ) {
        this.closeWindow(id);
      }
    }
  }

  private handleBeforeUnload(): void {
    this.saveAllWindowPositions();
    this.closeAllWindows();
  }

  private handleParentFocus(): void {
    // When parent window gains focus, update states
    this.windows.forEach((state, id) => {
      if (state.window && !state.window.closed) {
        try {
          // Check if child window has focus - this is an approximation
          // since we can't directly check focus state of child windows
          const childHasFocus = false;
          if (state.isFocused !== childHasFocus) {
            state.isFocused = childHasFocus;
            this.events.onStateChange?.(id, { isFocused: childHasFocus });
          }
        } catch {
          // Cross-origin or other access issues
        }
      }
    });
  }

  private handleParentBlur(): void {
    // Parent window lost focus - check if any child window gained it
    setTimeout(() => {
      this.windows.forEach((state) => {
        if (state.window && !state.window.closed) {
          try {
            // Attempt to detect if child window has focus
            state.window.focus();
          } catch {
            // Unable to focus, likely cross-origin
          }
        }
      });
    }, 100);
  }

  private cleanup(): void {
    if (this.cleanupInterval) {
      clearInterval(this.cleanupInterval);
    }
    this.closeAllWindows();
    window.removeEventListener("message", this.handleMessage);
    window.removeEventListener("beforeunload", this.handleBeforeUnload);
    window.removeEventListener("unload", this.cleanup);
    window.removeEventListener("focus", this.handleParentFocus);
    window.removeEventListener("blur", this.handleParentBlur);
  }

  private loadStoredPositions(): void {
    try {
      const stored = localStorage.getItem(POSITION_STORAGE_KEY);
      if (stored) {
        const positions = JSON.parse(stored) as Record<
          string,
          StoredWindowPosition
        >;
        Object.entries(positions).forEach(([path, position]) => {
          this.storedPositions.set(path, position);
        });
      }
    } catch (error) {
      console.error("[Trenova] Error loading stored positions:", error);
    }
  }

  private saveWindowPosition(id: string): void {
    try {
      const state = this.windows.get(id);
      if (!state?.window || state.window.closed) return;

      const position: StoredWindowPosition = {
        left: state.window.screenX,
        top: state.window.screenY,
        width: state.window.outerWidth,
        height: state.window.outerHeight,
      };

      this.storedPositions.set(state.path, position);
      this.persistStoredPositions();
    } catch (error) {
      console.error(`[Trenova] Error saving window position for ${id}:`, error);
    }
  }

  private saveAllWindowPositions(): void {
    this.windows.forEach((_, id) => {
      this.saveWindowPosition(id);
    });
  }

  private persistStoredPositions(): void {
    try {
      const positions: Record<string, StoredWindowPosition> = {};
      this.storedPositions.forEach((position, path) => {
        positions[path] = position;
      });
      localStorage.setItem(POSITION_STORAGE_KEY, JSON.stringify(positions));
    } catch (error) {
      console.error("[Trenova] Error persisting stored positions:", error);
    }
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
