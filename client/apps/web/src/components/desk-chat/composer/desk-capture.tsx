import { useDeskSetting } from "@/stores/desk-settings-store";
import {
  cancelCaptureRequest,
  createCaptureRequest,
  type CaptureBatchDetail,
  type CaptureDevice,
  type CaptureRequest,
} from "@/lib/graphql/capture";
import { queries } from "@/lib/queries";
import { api } from "@trenova/shared/lib/api";
import { useQueries, useQuery, useQueryClient } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useCallback, useMemo, useState } from "react";
import { Link } from "react-router";
import { DeskIcon } from "../desk-icons";
import type { DeskAttachment } from "./desk-attachments";

/** What a scan into a conversation is filed against. */
const THREAD_TARGET = "assistant_thread";

/** How often an unfinished scan is checked, on top of the realtime nudges. */
const SCAN_POLL_MS = 1500;

type StartedScan = { requestId: string; device: string; startedAt: number };

type ScanNameParams = {
  startedAt: number;
  pages: number;
  index: number;
  of: number;
  t: ReturnType<typeof useT>;
};

function scanName({ startedAt, pages, index, of, t }: ScanNameParams): string {
  const time = new Date(startedAt)
    .toLocaleTimeString([], { hour: "numeric", minute: "2-digit" })
    .replace(/\s/g, "");
  return of > 1
    ? t(
        "Scan {0} ({1} of {2}) · {3, plural, one {# page} other {# pages}}.pdf",
        time,
        index + 1,
        of,
        pages,
      )
    : t("Scan {0} · {1, plural, one {# page} other {# pages}}.pdf", time, pages);
}

/** Turns one scan's request and batch into the chips it shows on the message. */
function scanAttachments(
  scan: StartedScan,
  request: CaptureRequest | undefined,
  batch: CaptureBatchDetail | undefined,
  t: ReturnType<typeof useT>,
): DeskAttachment[] {
  const base = { id: scan.requestId, name: scan.device, size: 0, progress: 0 };
  const failed =
    request?.status === "Failed" || request?.status === "Expired" || request?.status === "Canceled";
  if (failed) {
    return [
      {
        ...base,
        status: "error",
        refused: true,
        error:
          request?.status === "Expired"
            ? t("{0} didn't pick up the scan", scan.device)
            : request?.failureMessage || t("No pages came through"),
      },
    ];
  }
  const items = (batch?.items ?? []).filter((item) => item.status !== "Discarded");
  const filed = items.filter((item) => item.status === "Filed" && item.documentId);
  const settled =
    batch !== undefined &&
    batch.sealedAt != null &&
    items.length > 0 &&
    items.every((item) => item.status === "Filed" || item.status === "Failed");
  if (!settled) {
    return [
      {
        ...base,
        status: "uploading",
        scan: { device: scan.device, pages: batch?.receivedPageCount ?? 0 },
      },
    ];
  }
  if (filed.length === 0) {
    return [{ ...base, status: "error", refused: true, error: t("No pages came through") }];
  }
  return filed.map((item, index) => ({
    id: `${scan.requestId}:${item.id}`,
    name: scanName({
      startedAt: scan.startedAt,
      pages: item.pageCount,
      index,
      of: filed.length,
      t,
    }),
    size: item.pageCount * 180_000,
    status: "ready",
    progress: 1,
    documentId: item.documentId ?? undefined,
    contentType: "application/pdf",
  }));
}

export type DeskScans = ReturnType<typeof useDeskScans>;

/**
 * Scans started from this conversation's composer. Each is a capture request
 * aimed at the conversation: the person's own computer scans the pages, the
 * pages are filed onto the conversation as a document, and that document
 * rides with the message like any uploaded file. Until the pages are filed
 * the scan shows as one chip counting the pages as they arrive.
 */
export function useDeskScans(threadId: string | null) {
  const t = useT();
  const queryClient = useQueryClient();
  const [scans, setScans] = useState<StartedScan[]>([]);
  const [dropped, setDropped] = useState<ReadonlySet<string>>(() => new Set());
  const active = threadId !== null && scans.length > 0;

  const requests = useQuery({
    ...queries.capture.requests(THREAD_TARGET, threadId ?? ""),
    enabled: active,
    refetchInterval: active ? SCAN_POLL_MS : false,
  });
  const byId = useMemo(
    () => new Map((requests.data ?? []).map((request) => [request.id, request])),
    [requests.data],
  );
  const batchIds = scans
    .map((scan) => byId.get(scan.requestId)?.batchId)
    .filter((id): id is string => Boolean(id));
  const batches = useQueries({
    queries: batchIds.map((id) => ({
      ...queries.capture.batch(id),
      refetchInterval: SCAN_POLL_MS,
    })),
  });
  const batchById = new Map(
    batches.flatMap((result) => (result.data ? [[result.data.id, result.data] as const] : [])),
  );

  const items = scans
    .flatMap((scan) => {
      const request = byId.get(scan.requestId);
      const batch = request?.batchId ? batchById.get(request.batchId) : undefined;
      return scanAttachments(scan, request, batch, t);
    })
    .filter((item) => !dropped.has(item.id));

  const start = useCallback(
    async (input: {
      device: CaptureDevice;
      sourceName: string;
      profileId: string | null;
      documentTypeId: string | null;
    }) => {
      if (!threadId) {
        return;
      }
      const request = await createCaptureRequest({
        deviceId: input.device.id,
        mode: "Scan",
        targetType: THREAD_TARGET,
        targetId: threadId,
        profileId: input.profileId,
        documentTypeId: input.documentTypeId,
        sourceName: input.sourceName || null,
      });
      setScans((current) => [
        ...current,
        { requestId: request.id, device: input.device.name, startedAt: Date.now() },
      ]);
      void queryClient.invalidateQueries({
        queryKey: queries.capture.requests(THREAD_TARGET, threadId).queryKey,
      });
    },
    [queryClient, threadId],
  );

  const remove = useCallback(
    (id: string) => {
      const scan = scans.find((candidate) => candidate.requestId === id);
      const request = scan ? byId.get(scan.requestId) : undefined;
      if (scan && request?.isOpen) {
        void cancelCaptureRequest(request.id);
      }
      setDropped((current) => new Set(current).add(id));
    },
    [byId, scans],
  );

  const clear = useCallback(() => {
    setScans([]);
    setDropped(new Set());
  }, []);

  return {
    items,
    start,
    remove,
    clear,
    owns: (id: string) => items.some((item) => item.id === id),
  };
}

type DocumentTypeOption = { id: string; name: string };

function useDocumentTypeOptions(enabled: boolean) {
  return useQuery({
    queryKey: ["desk", "capture", "document-types"],
    queryFn: async () => {
      const response = await api.get<{ results: DocumentTypeOption[] }>(
        "/document-types/select-options/?limit=50",
      );
      return response.results ?? [];
    },
    enabled,
    staleTime: 5 * 60_000,
  });
}

/**
 * Choosing where a scan comes from: the person's computers running Trenova
 * Capture, which scanner on it, the scan settings and what kind of paper it
 * is. A computer that is not connected can still be picked; the scan waits a
 * few minutes for it to come online.
 */
export function DeskCapturePanel({
  onBack,
  onStarted,
  scans,
}: {
  onBack: () => void;
  onStarted: () => void;
  scans: DeskScans;
}) {
  const t = useT();
  const devicesQuery = useQuery(queries.capture.myDevices("Active"));
  const profilesQuery = useQuery(queries.capture.availableProfiles());
  const devices = devicesQuery.data ?? [];
  const typesQuery = useDocumentTypeOptions(devices.length > 0);
  const preferredDevice = useDeskSetting("scanDevice");
  const preferredProfile = useDeskSetting("scanProfile");
  const [deviceId, setDeviceId] = useState<string | null>(preferredDevice || null);
  const [source, setSource] = useState("");
  const [profileId, setProfileId] = useState<string | null>(preferredProfile || null);
  const [documentTypeId, setDocumentTypeId] = useState("");
  const [starting, setStarting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const device =
    devices.find((candidate) => candidate.id === deviceId) ??
    devices.find((candidate) => candidate.isOnline) ??
    devices[0];
  const profiles = profilesQuery.data ?? [];
  const profile = profiles.find((candidate) => candidate.id === profileId) ?? profiles[0];

  const start = async () => {
    if (!device) {
      return;
    }
    setStarting(true);
    setError(null);
    try {
      await scans.start({
        device,
        sourceName: source,
        profileId: profile?.id ?? null,
        documentTypeId: documentTypeId || null,
      });
      onStarted();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : t("The scan couldn't be started"));
    } finally {
      setStarting(false);
    }
  };

  return (
    <div className="dk-cap">
      <div className="dk-cap-h">
        <button
          type="button"
          className="dk-ib"
          title={t("Back")}
          aria-label={t("Back")}
          onClick={onBack}
        >
          <DeskIcon name="chevL" size={13} />
        </button>
        <span>
          <b>{t("Scan from Capture")}</b>
          <em>{t("Pages scan on your computer and attach to this message")}</em>
        </span>
      </div>
      {devicesQuery.isPending ? (
        <div className="dk-cap-empty">
          <span>{t("Looking for your computers…")}</span>
        </div>
      ) : devices.length === 0 ? (
        <div className="dk-cap-empty">
          <span className="dk-cap-eic">
            <DeskIcon name="scanner" size={18} stroke={1.8} />
          </span>
          <b>{t("No computer is set up to scan for you")}</b>
          <span>
            {t(
              "Install Trenova Capture, sign in from its tray icon, and approve the code it shows.",
            )}
          </span>
          <div className="dk-cap-acts">
            <Link className="dk-ec-btn dk-ink" to="/capture/devices">
              {t("Download Trenova Capture")}
            </Link>
            <Link className="dk-ec-btn" to="/capture/devices">
              {t("My scanners")}
            </Link>
          </div>
        </div>
      ) : (
        <>
          <div className="dk-cap-f">
            <div className="dk-cap-l">{t("Computer")}</div>
            <div className="dk-cap-devs">
              {devices.map((candidate) => (
                <button
                  key={candidate.id}
                  type="button"
                  className={cn("dk-cap-dev", device?.id === candidate.id && "dk-on")}
                  onClick={() => {
                    setDeviceId(candidate.id);
                    setSource("");
                  }}
                >
                  <span className={cn("dk-cap-dot", candidate.isOnline && "dk-ok")} />
                  <b>{candidate.name}</b>
                  <em>{candidate.isOnline ? t("Connected") : t("Not connected")}</em>
                  {device?.id === candidate.id && <DeskIcon name="check" size={12} stroke={2.4} />}
                </button>
              ))}
            </div>
            {device && !device.isOnline && (
              <div className="dk-cap-warn">
                <DeskIcon name="alert" size={12} stroke={2} />
                {t(
                  "{0} isn't connected. The scan waits a few minutes for it to come online.",
                  device.name,
                )}
              </div>
            )}
          </div>
          <div className="dk-cap-grid">
            <label className="dk-cap-sel">
              <span>{t("Scanner")}</span>
              <select value={source} onChange={(event) => setSource(event.target.value)}>
                <option value="">{t("Computer's default")}</option>
                {(device?.sources ?? []).map((candidate) => (
                  <option key={candidate.name} value={candidate.name}>
                    {candidate.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="dk-cap-sel">
              <span>{t("Scan settings")}</span>
              <select
                value={profile?.id ?? ""}
                onChange={(event) => setProfileId(event.target.value || null)}
              >
                {profiles.length === 0 && <option value="">{t("Organization default")}</option>}
                {profiles.map((candidate) => (
                  <option key={candidate.id} value={candidate.id}>
                    {candidate.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="dk-cap-sel dk-wide">
              <span>{t("Document type")}</span>
              <select
                value={documentTypeId}
                onChange={(event) => setDocumentTypeId(event.target.value)}
              >
                <option value="">{t("Let Desk decide")}</option>
                {(typesQuery.data ?? []).map((candidate) => (
                  <option key={candidate.id} value={candidate.id}>
                    {candidate.name}
                  </option>
                ))}
              </select>
            </label>
          </div>
          <div className="dk-cap-foot">
            <span>
              {error ??
                (device
                  ? device.isOnline
                    ? t("Put the pages in the scanner after you start.")
                    : t("Starts when {0} connects.", device.name)
                  : "")}
            </span>
            <button
              type="button"
              className="dk-ec-btn dk-ink"
              disabled={!device || starting}
              onClick={() => void start()}
            >
              {starting ? t("Starting…") : t("Start scan")}
            </button>
          </div>
        </>
      )}
    </div>
  );
}
