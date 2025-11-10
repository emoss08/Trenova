import { CLIENT_NAME, CLIENT_VERSION } from "@/constants/env";
import { Check, Copy, Download } from "lucide-react";
import { useEffect, useState } from "react";
import { Button } from "./ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "./ui/dialog";
import { VisuallyHidden } from "./ui/visually-hidden";

interface ClientInfo {
  timestamp: string;
  browser: {
    userAgent: string;
    language: string;
    languages: readonly string[];
    cookieEnabled: boolean;
    onLine: boolean;
    platform: string;
    vendor: string;
    hardwareConcurrency: number;
    maxTouchPoints: number;
    doNotTrack: string | null;
  };
  screen: {
    width: number;
    height: number;
    availWidth: number;
    availHeight: number;
    colorDepth: number;
    pixelRatio: number;
    orientation: string;
  };
  viewport: {
    width: number;
    height: number;
  };
  timezone: {
    offset: number;
    name: string;
  };
  performance: {
    memory?: {
      jsHeapSizeLimit: number;
      totalJSHeapSize: number;
      usedJSHeapSize: number;
    };
    navigation?: {
      type: string;
      redirectCount: number;
    };
    timing?: {
      loadTime: number;
      domContentLoaded: number;
      domInteractive: number;
    };
  };
  storage: {
    localStorage: boolean;
    sessionStorage: boolean;
    indexedDB: boolean;
    quota?: {
      usage: number;
      quota: number;
      usagePercentage: number;
    };
  };
  network?: {
    effectiveType: string;
    downlink: number;
    rtt: number;
    saveData: boolean;
  };
  features: {
    webGL: boolean;
    webGL2: boolean;
    webRTC: boolean;
    serviceWorker: boolean;
    webAssembly: boolean;
    webWorker: boolean;
    geolocation: boolean;
    notifications: boolean;
    vibration: boolean;
    bluetooth: boolean;
    usb: boolean;
  };
  webGL?: {
    vendor: string;
    renderer: string;
    version: string;
    shadingLanguageVersion: string;
    maxTextureSize: number;
  };
  userPreferences: {
    prefersColorScheme: string;
    prefersReducedMotion: boolean;
    prefersReducedTransparency: boolean;
    prefersContrast: string;
  };
  serviceWorkers?: {
    registered: boolean;
    count: number;
  };
  application: {
    url: string;
    referrer: string;
    protocol: string;
    host: string;
    documentMode?: string;
    visibilityState: string;
  };
}

async function getClientInformation(): Promise<ClientInfo> {
  const nav = window.navigator;
  const perf = window.performance as Performance & {
    memory?: {
      jsHeapSizeLimit: number;
      totalJSHeapSize: number;
      usedJSHeapSize: number;
    };
  };

  // Get WebGL info
  const getWebGLInfo = () => {
    try {
      const canvas = document.createElement("canvas");
      const gl =
        canvas.getContext("webgl") || canvas.getContext("experimental-webgl");
      if (!gl || !(gl instanceof WebGLRenderingContext)) return undefined;

      const debugInfo = gl.getExtension("WEBGL_debug_renderer_info");
      return {
        vendor: debugInfo
          ? gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL)
          : gl.getParameter(gl.VENDOR),
        renderer: debugInfo
          ? gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL)
          : gl.getParameter(gl.RENDERER),
        version: gl.getParameter(gl.VERSION),
        shadingLanguageVersion: gl.getParameter(gl.SHADING_LANGUAGE_VERSION),
        maxTextureSize: gl.getParameter(gl.MAX_TEXTURE_SIZE),
      };
    } catch {
      return undefined;
    }
  };

  // Get storage quota
  const getStorageQuota = async () => {
    try {
      if ("storage" in navigator && "estimate" in navigator.storage) {
        const estimate = await navigator.storage.estimate();
        const usage = estimate.usage || 0;
        const quota = estimate.quota || 0;
        return {
          usage,
          quota,
          usagePercentage: quota > 0 ? Math.round((usage / quota) * 100) : 0,
        };
      }
    } catch {
      return undefined;
    }
    return undefined;
  };

  // Get network information
  const getNetworkInfo = () => {
    try {
      const conn =
        (nav as any).connection ||
        (nav as any).mozConnection ||
        (nav as any).webkitConnection;
      if (conn) {
        return {
          effectiveType: conn.effectiveType || "unknown",
          downlink: conn.downlink || 0,
          rtt: conn.rtt || 0,
          saveData: conn.saveData || false,
        };
      }
    } catch {
      return undefined;
    }
    return undefined;
  };

  // Get service worker info
  const getServiceWorkerInfo = async () => {
    try {
      if ("serviceWorker" in navigator) {
        const registrations = await navigator.serviceWorker.getRegistrations();
        return {
          registered: registrations.length > 0,
          count: registrations.length,
        };
      }
    } catch {
      return undefined;
    }
    return undefined;
  };

  // Get performance timing
  const getPerformanceTiming = () => {
    try {
      const timing = perf.timing;
      if (timing && timing.loadEventEnd && timing.navigationStart) {
        return {
          loadTime: timing.loadEventEnd - timing.navigationStart,
          domContentLoaded:
            timing.domContentLoadedEventEnd - timing.navigationStart,
          domInteractive: timing.domInteractive - timing.navigationStart,
        };
      }
    } catch {
      return undefined;
    }
    return undefined;
  };

  const [storageQuota, serviceWorkerInfo] = await Promise.all([
    getStorageQuota(),
    getServiceWorkerInfo(),
  ]);

  return {
    timestamp: new Date().toISOString(),
    browser: {
      userAgent: nav.userAgent,
      language: nav.language,
      languages: nav.languages,
      cookieEnabled: nav.cookieEnabled,
      onLine: nav.onLine,
      platform: nav.platform,
      vendor: nav.vendor,
      hardwareConcurrency: nav.hardwareConcurrency || 0,
      maxTouchPoints: nav.maxTouchPoints || 0,
      doNotTrack: nav.doNotTrack || null,
    },
    screen: {
      width: window.screen.width,
      height: window.screen.height,
      availWidth: window.screen.availWidth,
      availHeight: window.screen.availHeight,
      colorDepth: window.screen.colorDepth,
      pixelRatio: window.devicePixelRatio,
      orientation: window.screen.orientation?.type || "unknown",
    },
    viewport: {
      width: window.innerWidth,
      height: window.innerHeight,
    },
    timezone: {
      offset: new Date().getTimezoneOffset(),
      name: Intl.DateTimeFormat().resolvedOptions().timeZone,
    },
    performance: {
      memory: perf.memory
        ? {
            jsHeapSizeLimit: perf.memory.jsHeapSizeLimit,
            totalJSHeapSize: perf.memory.totalJSHeapSize,
            usedJSHeapSize: perf.memory.usedJSHeapSize,
          }
        : undefined,
      navigation: perf.navigation
        ? {
            type:
              ["navigate", "reload", "back_forward", "prerender"][
                perf.navigation.type
              ] || "unknown",
            redirectCount: perf.navigation.redirectCount,
          }
        : undefined,
      timing: getPerformanceTiming(),
    },
    storage: {
      localStorage: (() => {
        try {
          return typeof window.localStorage !== "undefined";
        } catch {
          return false;
        }
      })(),
      sessionStorage: (() => {
        try {
          return typeof window.sessionStorage !== "undefined";
        } catch {
          return false;
        }
      })(),
      indexedDB: typeof window.indexedDB !== "undefined",
      quota: storageQuota,
    },
    network: getNetworkInfo(),
    features: {
      webGL: (() => {
        try {
          const canvas = document.createElement("canvas");
          return !!(
            canvas.getContext("webgl") ||
            canvas.getContext("experimental-webgl")
          );
        } catch {
          return false;
        }
      })(),
      webGL2: (() => {
        try {
          const canvas = document.createElement("canvas");
          return !!canvas.getContext("webgl2");
        } catch {
          return false;
        }
      })(),
      webRTC: !!(
        (window as any).RTCPeerConnection ||
        (window as any).webkitRTCPeerConnection ||
        (window as any).mozRTCPeerConnection
      ),
      serviceWorker: "serviceWorker" in navigator,
      webAssembly: typeof WebAssembly !== "undefined",
      webWorker: typeof Worker !== "undefined",
      geolocation: "geolocation" in navigator,
      notifications: "Notification" in window,
      vibration: "vibrate" in navigator,
      bluetooth: "bluetooth" in navigator,
      usb: "usb" in navigator,
    },
    webGL: getWebGLInfo(),
    userPreferences: {
      prefersColorScheme: window.matchMedia("(prefers-color-scheme: dark)")
        .matches
        ? "dark"
        : window.matchMedia("(prefers-color-scheme: light)").matches
          ? "light"
          : "no-preference",
      prefersReducedMotion: window.matchMedia(
        "(prefers-reduced-motion: reduce)",
      ).matches,
      prefersReducedTransparency: window.matchMedia(
        "(prefers-reduced-transparency: reduce)",
      ).matches,
      prefersContrast: window.matchMedia("(prefers-contrast: more)").matches
        ? "more"
        : window.matchMedia("(prefers-contrast: less)").matches
          ? "less"
          : window.matchMedia("(prefers-contrast: no-preference)").matches
            ? "no-preference"
            : "unknown",
    },
    serviceWorkers: serviceWorkerInfo,
    application: {
      url: window.location.href,
      referrer: document.referrer,
      protocol: window.location.protocol,
      host: window.location.host,
      documentMode: (document as any).documentMode?.toString(),
      visibilityState: document.visibilityState,
    },
  };
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 Bytes";
  const k = 1024;
  const sizes = ["Bytes", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + " " + sizes[i];
}

export function ClientInformation() {
  const [open, setOpen] = useState(false);
  const [clientInfo, setClientInfo] = useState<ClientInfo | null>(null);
  const [copied, setCopied] = useState(false);

  // Keyboard shortcut: Ctrl+Shift+D to toggle debug modal
  useEffect(() => {
    const handleKeyDown = async (event: KeyboardEvent) => {
      if (event.ctrlKey && event.shiftKey && event.key === "D") {
        event.preventDefault();
        setOpen((prev) => !prev);

        // Fetch client info when opening
        if (!open) {
          const info = await getClientInformation();
          setClientInfo(info);
        }
      }
    };

    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [open]);

  const handleCopy = async () => {
    if (!clientInfo) return;
    const text = JSON.stringify(clientInfo, null, 2);
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  const handleDownload = () => {
    if (!clientInfo) return;
    const blob = new Blob([JSON.stringify(clientInfo, null, 2)], {
      type: "application/json",
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `client-debug-${Date.now()}.json`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-3xl" withClose={false}>
        <VisuallyHidden>
          <DialogHeader>
            <DialogTitle>Client Debug Information</DialogTitle>
            <DialogDescription>
              Technical information for debugging
            </DialogDescription>
          </DialogHeader>
        </VisuallyHidden>
        <DialogBody>
          {clientInfo ? (
            <div className="flex flex-col gap-6">
              <div className="flex items-start justify-between">
                <div className="flex flex-col gap-1">
                  <h2 className="text-2xl font-semibold">
                    Client Debug Information
                  </h2>
                  <p className="text-sm text-muted-foreground">
                    Technical details for troubleshooting and support
                  </p>
                  <p className="text-xs text-muted-foreground/70">
                    Press{" "}
                    <kbd className="rounded border border-border bg-muted px-1.5 py-0.5 font-mono text-[10px]">
                      Ctrl+Shift+D
                    </kbd>{" "}
                    or{" "}
                    <kbd className="rounded border border-border bg-muted px-1.5 py-0.5 font-mono text-[10px]">
                      Esc
                    </kbd>{" "}
                    to close
                  </p>
                </div>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleCopy}
                    className="gap-2"
                  >
                    {copied ? (
                      <Check className="size-4" />
                    ) : (
                      <Copy className="size-4" />
                    )}
                    {copied ? "Copied" : "Copy"}
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={handleDownload}
                    className="gap-2"
                  >
                    <Download className="size-4" />
                    Download
                  </Button>
                </div>
              </div>

              <InfoSection title="Session Information">
                <InfoRow label="Timestamp" value={clientInfo.timestamp} />
                <InfoRow
                  label="Timezone"
                  value={`${clientInfo.timezone.name} (UTC${clientInfo.timezone.offset > 0 ? "-" : "+"}${Math.abs(clientInfo.timezone.offset / 60)})`}
                />
              </InfoSection>

              <InfoSection title="Browser Information">
                <InfoRow
                  label="User Agent"
                  value={clientInfo.browser.userAgent}
                />
                <InfoRow label="Platform" value={clientInfo.browser.platform} />
                <InfoRow label="Vendor" value={clientInfo.browser.vendor} />
                <InfoRow label="Language" value={clientInfo.browser.language} />
                <InfoRow
                  label="Languages"
                  value={clientInfo.browser.languages.join(", ")}
                />
                <InfoRow
                  label="Cookies Enabled"
                  value={clientInfo.browser.cookieEnabled ? "Yes" : "No"}
                />
                <InfoRow
                  label="Online Status"
                  value={clientInfo.browser.onLine ? "Online" : "Offline"}
                />
                <InfoRow
                  label="Hardware Concurrency"
                  value={`${clientInfo.browser.hardwareConcurrency} cores`}
                />
                <InfoRow
                  label="Max Touch Points"
                  value={clientInfo.browser.maxTouchPoints.toString()}
                />
                {clientInfo.browser.doNotTrack && (
                  <InfoRow
                    label="Do Not Track"
                    value={clientInfo.browser.doNotTrack}
                  />
                )}
              </InfoSection>

              <InfoSection title="Display Information">
                <InfoRow
                  label="Screen Resolution"
                  value={`${clientInfo.screen.width} × ${clientInfo.screen.height}px`}
                />
                <InfoRow
                  label="Available Screen"
                  value={`${clientInfo.screen.availWidth} × ${clientInfo.screen.availHeight}px`}
                />
                <InfoRow
                  label="Viewport Size"
                  value={`${clientInfo.viewport.width} × ${clientInfo.viewport.height}px`}
                />
                <InfoRow
                  label="Color Depth"
                  value={`${clientInfo.screen.colorDepth}-bit`}
                />
                <InfoRow
                  label="Pixel Ratio"
                  value={clientInfo.screen.pixelRatio.toString()}
                />
                <InfoRow
                  label="Orientation"
                  value={clientInfo.screen.orientation}
                />
              </InfoSection>

              {clientInfo.performance.memory && (
                <InfoSection title="Memory Usage">
                  <InfoRow
                    label="JS Heap Size Limit"
                    value={formatBytes(
                      clientInfo.performance.memory.jsHeapSizeLimit,
                    )}
                  />
                  <InfoRow
                    label="Total JS Heap Size"
                    value={formatBytes(
                      clientInfo.performance.memory.totalJSHeapSize,
                    )}
                  />
                  <InfoRow
                    label="Used JS Heap Size"
                    value={formatBytes(
                      clientInfo.performance.memory.usedJSHeapSize,
                    )}
                  />
                </InfoSection>
              )}

              {clientInfo.performance.timing && (
                <InfoSection title="Performance Timing">
                  <InfoRow
                    label="Page Load Time"
                    value={`${clientInfo.performance.timing.loadTime}ms`}
                  />
                  <InfoRow
                    label="DOM Content Loaded"
                    value={`${clientInfo.performance.timing.domContentLoaded}ms`}
                  />
                  <InfoRow
                    label="DOM Interactive"
                    value={`${clientInfo.performance.timing.domInteractive}ms`}
                  />
                </InfoSection>
              )}

              {clientInfo.network && (
                <InfoSection title="Network Information">
                  <InfoRow
                    label="Effective Type"
                    value={clientInfo.network.effectiveType}
                  />
                  <InfoRow
                    label="Downlink Speed"
                    value={`${clientInfo.network.downlink} Mbps`}
                  />
                  <InfoRow
                    label="Round Trip Time"
                    value={`${clientInfo.network.rtt}ms`}
                  />
                  <InfoRow
                    label="Data Saver"
                    value={clientInfo.network.saveData ? "Enabled" : "Disabled"}
                  />
                </InfoSection>
              )}

              <InfoSection title="Storage Support">
                <InfoRow
                  label="Local Storage"
                  value={
                    clientInfo.storage.localStorage
                      ? "Available"
                      : "Unavailable"
                  }
                />
                <InfoRow
                  label="Session Storage"
                  value={
                    clientInfo.storage.sessionStorage
                      ? "Available"
                      : "Unavailable"
                  }
                />
                <InfoRow
                  label="IndexedDB"
                  value={
                    clientInfo.storage.indexedDB ? "Available" : "Unavailable"
                  }
                />
                {clientInfo.storage.quota && (
                  <>
                    <InfoRow
                      label="Storage Used"
                      value={formatBytes(clientInfo.storage.quota.usage)}
                    />
                    <InfoRow
                      label="Storage Quota"
                      value={formatBytes(clientInfo.storage.quota.quota)}
                    />
                    <InfoRow
                      label="Usage Percentage"
                      value={`${clientInfo.storage.quota.usagePercentage}%`}
                    />
                  </>
                )}
              </InfoSection>

              <InfoSection title="Feature Support">
                <InfoRow
                  label="WebGL"
                  value={
                    clientInfo.features.webGL ? "Supported" : "Not Supported"
                  }
                />
                <InfoRow
                  label="WebGL 2.0"
                  value={
                    clientInfo.features.webGL2 ? "Supported" : "Not Supported"
                  }
                />
                <InfoRow
                  label="WebRTC"
                  value={
                    clientInfo.features.webRTC ? "Supported" : "Not Supported"
                  }
                />
                <InfoRow
                  label="Service Worker"
                  value={
                    clientInfo.features.serviceWorker
                      ? "Supported"
                      : "Not Supported"
                  }
                />
                <InfoRow
                  label="WebAssembly"
                  value={
                    clientInfo.features.webAssembly
                      ? "Supported"
                      : "Not Supported"
                  }
                />
                <InfoRow
                  label="Web Worker"
                  value={
                    clientInfo.features.webWorker
                      ? "Supported"
                      : "Not Supported"
                  }
                />
                <InfoRow
                  label="Geolocation"
                  value={
                    clientInfo.features.geolocation
                      ? "Supported"
                      : "Not Supported"
                  }
                />
                <InfoRow
                  label="Notifications"
                  value={
                    clientInfo.features.notifications
                      ? "Supported"
                      : "Not Supported"
                  }
                />
                <InfoRow
                  label="Vibration"
                  value={
                    clientInfo.features.vibration
                      ? "Supported"
                      : "Not Supported"
                  }
                />
                <InfoRow
                  label="Bluetooth"
                  value={
                    clientInfo.features.bluetooth
                      ? "Supported"
                      : "Not Supported"
                  }
                />
                <InfoRow
                  label="USB"
                  value={
                    clientInfo.features.usb ? "Supported" : "Not Supported"
                  }
                />
              </InfoSection>

              {clientInfo.webGL && (
                <InfoSection title="WebGL Information">
                  <InfoRow label="Vendor" value={clientInfo.webGL.vendor} />
                  <InfoRow label="Renderer" value={clientInfo.webGL.renderer} />
                  <InfoRow label="Version" value={clientInfo.webGL.version} />
                  <InfoRow
                    label="Shading Language"
                    value={clientInfo.webGL.shadingLanguageVersion}
                  />
                  <InfoRow
                    label="Max Texture Size"
                    value={`${clientInfo.webGL.maxTextureSize}px`}
                  />
                </InfoSection>
              )}

              <InfoSection title="User Preferences">
                <InfoRow
                  label="Color Scheme"
                  value={clientInfo.userPreferences.prefersColorScheme}
                />
                <InfoRow
                  label="Reduced Motion"
                  value={
                    clientInfo.userPreferences.prefersReducedMotion
                      ? "Yes"
                      : "No"
                  }
                />
                <InfoRow
                  label="Reduced Transparency"
                  value={
                    clientInfo.userPreferences.prefersReducedTransparency
                      ? "Yes"
                      : "No"
                  }
                />
                <InfoRow
                  label="Contrast Preference"
                  value={clientInfo.userPreferences.prefersContrast}
                />
              </InfoSection>

              {clientInfo.serviceWorkers && (
                <InfoSection title="Service Workers">
                  <InfoRow
                    label="Registered"
                    value={clientInfo.serviceWorkers.registered ? "Yes" : "No"}
                  />
                  <InfoRow
                    label="Count"
                    value={clientInfo.serviceWorkers.count.toString()}
                  />
                </InfoSection>
              )}

              <InfoSection title="Application">
                <InfoRow label="Name" value={CLIENT_NAME} />
                <InfoRow label="Version" value={CLIENT_VERSION} />
                <InfoRow label="URL" value={clientInfo.application.url} />
                <InfoRow label="Host" value={clientInfo.application.host} />
                <InfoRow
                  label="Protocol"
                  value={clientInfo.application.protocol}
                />
                <InfoRow
                  label="Visibility State"
                  value={clientInfo.application.visibilityState}
                />
                {clientInfo.application.documentMode && (
                  <InfoRow
                    label="Document Mode"
                    value={clientInfo.application.documentMode}
                  />
                )}
                {clientInfo.application.referrer && (
                  <InfoRow
                    label="Referrer"
                    value={clientInfo.application.referrer}
                  />
                )}
              </InfoSection>

              {clientInfo.performance.navigation && (
                <InfoSection title="Navigation Information">
                  <InfoRow
                    label="Navigation Type"
                    value={clientInfo.performance.navigation.type}
                  />
                  <InfoRow
                    label="Redirect Count"
                    value={clientInfo.performance.navigation.redirectCount.toString()}
                  />
                </InfoSection>
              )}
            </div>
          ) : null}
        </DialogBody>
      </DialogContent>
    </Dialog>
  );
}

function InfoSection({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-3">
      <h3 className="text-sm font-semibold tracking-wide text-foreground/80 uppercase">
        {title}
      </h3>
      <div className="grid gap-2 rounded-lg border border-border bg-muted/30 p-4">
        {children}
      </div>
    </div>
  );
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid grid-cols-[200px_1fr] gap-4 text-sm">
      <span className="font-medium text-muted-foreground">{label}:</span>
      <span className="font-mono break-all text-foreground">{value}</span>
    </div>
  );
}
