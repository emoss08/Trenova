import { queries } from "@/lib/queries";
import { useDeskStore } from "@/stores/desk-store";
import type { AssistantArtifact } from "@/types/assistant";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { useMemo, useState } from "react";
import { ArtIcon } from "./desk-art-kinds";

/** One value read off the page, how sure the reading is, and where it was. */
export type ExtractedField = {
  key: string;
  label: string;
  value: string;
  confidence: number;
  needsLook: boolean;
  page: number;
  box: { x: number; y: number; w: number; h: number } | null;
};

export type Extraction = {
  documentId: string;
  fileName: string;
  kindLabel: string;
  confidence: number;
  pageCount: number;
  fields: ExtractedField[];
  attachedShipmentId: string;
};

function text(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function number(value: unknown): number {
  return typeof value === "number" && Number.isFinite(value) ? value : 0;
}

/** An extraction artifact's payload, read defensively. */
export function extractionFrom(artifact: AssistantArtifact): Extraction {
  const payload = artifact.payload;
  const fields = Array.isArray(payload.fields) ? payload.fields : [];
  return {
    documentId: text(payload.documentId),
    fileName: text(payload.fileName),
    kindLabel: text(payload.kindLabel) || text(payload.documentKind),
    confidence: number(payload.confidence),
    pageCount: Math.max(1, number(payload.pageCount)),
    attachedShipmentId: text(payload.attachedShipmentId),
    fields: fields
      .filter(
        (field): field is Record<string, unknown> => typeof field === "object" && field !== null,
      )
      .map((field) => {
        const box = field.box as Record<string, unknown> | undefined;
        return {
          key: text(field.key),
          label: text(field.label),
          value: text(field.value),
          confidence: number(field.confidence),
          needsLook: field.needsLook === true,
          page: number(field.page),
          box:
            box && typeof box === "object"
              ? { x: number(box.x), y: number(box.y), w: number(box.w), h: number(box.h) }
              : null,
        };
      })
      .filter((field) => field.value !== ""),
  };
}

/** Where a placeholder line sits on the page mock, as in the design. */
const LINE_WIDTHS = [72, 64, 80, 40, 76, 58, 69, 33];

/**
 * What document intelligence read off an uploaded file: a page with a box
 * over every value it found, the pages to step through, and each field with
 * how sure the reading is. Pointing at a field lights its box and the other
 * way round. The values that need a look come first in the eye: amber, and
 * counted above the list. From here the agent drafts the shipment, which
 * waits for approval like any other write.
 */
export function DeskExtractBody({ artifact }: { artifact: AssistantArtifact }) {
  const t = useT();
  const extraction = useMemo(() => extractionFrom(artifact), [artifact]);
  const [hot, setHot] = useState<string | null>(null);
  const [page, setPage] = useState(1);
  // Asked this session: the agent is drafting until its proposal arrives.
  const [requested, setRequested] = useState(false);
  const askAgent = useDeskStore((state) => state.askAgent);
  const asked = useDeskStore((state) => state.asks[artifact.threadId] ?? null);
  const proposalsQuery = useQuery(queries.assistant.proposals(artifact.threadId));
  const drafted = (proposalsQuery.data?.results ?? []).find(
    (proposal) =>
      proposal.toolName === "create_shipment" &&
      extraction.documentId !== "" &&
      proposal.arguments?.sourceDocumentId === extraction.documentId,
  );
  const low = extraction.fields.filter((field) => field.needsLook);
  const onPage = extraction.fields.filter((field) => field.box && field.page === page);

  const ask = (question: string) => askAgent(artifact.threadId, question);
  const fileName = extraction.fileName || t("this document");

  return (
    <div className="dk-ax-pad dk-ex">
      <div className="dk-ex-h">
        <span className="dk-ex-type">{extraction.kindLabel || t("Document")}</span>
        <span>
          {[
            extraction.fileName,
            t("{0, plural, one {# page} other {# pages}}", extraction.pageCount),
            extraction.confidence > 0
              ? t("classified at {0}%", Math.round(extraction.confidence * 100))
              : "",
          ]
            .filter(Boolean)
            .join(" · ")}
        </span>
      </div>
      <div className="dk-ex-grid">
        <div className="dk-ex-doc">
          <div className="dk-ex-page" aria-label={t("Page {0}", page)}>
            <div className="dk-ex-ph">
              <i style={{ width: "34%" }} />
              <i style={{ width: "22%", marginLeft: "auto" }} />
            </div>
            {Array.from({ length: 22 }, (_, index) => (
              <i
                key={index}
                className="dk-ex-ln"
                style={{
                  width: `${LINE_WIDTHS[index % LINE_WIDTHS.length]}%`,
                  marginTop: index % 6 === 0 ? 10 : 4,
                }}
              />
            ))}
            {onPage.map((field) => (
              <span
                key={field.key}
                className={cn(
                  "dk-ex-box",
                  field.needsLook && "dk-low",
                  hot === field.key && "dk-on",
                )}
                title={`${field.label}: ${field.value}`}
                style={{
                  left: `${(field.box?.x ?? 0) * 100}%`,
                  top: `${(field.box?.y ?? 0) * 100}%`,
                  width: `${(field.box?.w ?? 0) * 100}%`,
                  height: `${(field.box?.h ?? 0) * 100}%`,
                }}
                onMouseEnter={() => setHot(field.key)}
                onMouseLeave={() => setHot(null)}
              />
            ))}
          </div>
          {extraction.pageCount > 1 && (
            <div className="dk-ex-pg">
              {Array.from({ length: extraction.pageCount }, (_, index) => (
                <Button
                  key={index}
                  variant="bare"
                  size="bare"
                  className={cn(
                    "h-5.5 w-6 justify-center rounded-md font-plex-mono text-xs text-dsk-subtle hover:bg-dsk-hover",
                    page === index + 1 && "bg-dsk-ink text-dsk-ink-fg hover:bg-dsk-ink",
                  )}
                  aria-label={t("Page {0}", index + 1)}
                  aria-current={page === index + 1}
                  onClick={() => setPage(index + 1)}
                >
                  {index + 1}
                </Button>
              ))}
            </div>
          )}
        </div>
        <div className="dk-ex-fields">
          {low.length > 0 && (
            <div className="dk-ex-warn">
              <ArtIcon name="alert" size={12} stroke={2} />
              {t(
                "{0, plural, one {# field needs a look} other {# fields need a look}}",
                low.length,
              )}
            </div>
          )}
          {extraction.fields.map((field) => (
            // oxlint-disable-next-line jsx-a11y/no-static-element-interactions -- pointing links a field to its box; the value is read either way
            <div
              key={field.key}
              className={cn("dk-ex-f", field.needsLook && "dk-low", hot === field.key && "dk-on")}
              onMouseEnter={() => {
                setHot(field.key);
                if (field.page > 0) setPage(field.page);
              }}
              onMouseLeave={() => setHot(null)}
            >
              <span className="dk-ex-l">{field.label}</span>
              <span className="dk-ex-v" title={field.value}>
                {field.value}
              </span>
              <span
                className="dk-ex-c"
                title={t("{0}% confident", Math.round(field.confidence * 100))}
              >
                <i style={{ width: `${field.confidence * 100}%` }} />
              </span>
            </div>
          ))}
        </div>
      </div>
      <div className="dk-ax-acts">
        <Button
          variant="quiet"
          disabled={low.length === 0 || asked !== null}
          onClick={() =>
            ask(
              t(
                "Check the fields that need a look on {0} against the document and fix what it misread: {1}.",
                fileName,
                low.map((field) => field.label).join(", "),
              ),
            )
          }
        >
          {t("Fix fields")}
        </Button>
        <span className="flex-1" />
        {extraction.attachedShipmentId !== "" ? (
          <span className="dk-ax-sent">
            <ArtIcon name="check" size={13} stroke={2.4} />
            {t("Already a shipment")}
          </span>
        ) : drafted?.status === "Executed" ? (
          <span className="dk-ax-sent">
            <ArtIcon name="check" size={13} stroke={2.4} />
            {t("Shipment created")}
          </span>
        ) : drafted && drafted.status === "Pending" ? (
          <span className="dk-ax-sent">
            <ArtIcon name="check" size={13} stroke={2.4} />
            {t("Shipment drafted · waiting on your approval")}
          </span>
        ) : requested || asked !== null ? (
          <span className="dk-ax-sent dk-wait">{t("Drafting the shipment…")}</span>
        ) : (
          <Button
            disabled={extraction.documentId === ""}
            onClick={() => {
              setRequested(true);
              ask(
                t(
                  "Create a shipment from {0} (document {1}), using what was read from it.",
                  fileName,
                  extraction.documentId,
                ),
              );
            }}
          >
            {t("Create shipment from this")}
          </Button>
        )}
      </div>
    </div>
  );
}
