import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import { ArtIcon, DeskArtKindIcon, type DeskArtKind } from "./desk-art-kinds";

const STACK: readonly DeskArtKind[] = ["report", "record", "table"];

/**
 * The workspace before a conversation has made anything: three blank cards
 * fanned out, each marked with a kind of artifact, what will land here and
 * what can be done with it, and the shortcut that brings the panel back.
 */
export function DeskArtifactsEmpty({ onClose }: { onClose: () => void }) {
  const t = useT();

  return (
    <div className="dk-axe" data-slot="artifacts-empty">
      <div className="dk-axe-top">
        <button
          type="button"
          className="dk-ax-ib"
          title={t("Close")}
          aria-label={t("Close")}
          onClick={onClose}
        >
          <ArtIcon name="x" size={13} stroke={2.2} />
        </button>
      </div>
      <div className="dk-axe-c">
        <div className="dk-axe-stack" aria-hidden>
          {STACK.map((kind) => (
            <span key={kind}>
              <span className={cn("dk-ax-ki", `dk-k-${kind}`)}>
                <DeskArtKindIcon kind={kind} size={14} />
              </span>
              <i />
              <i />
            </span>
          ))}
        </div>
        <b>{t("No artifacts yet")}</b>
        <p>
          {t(
            "When the agent pulls a table, opens a record or drafts an email, it lands here so you can check it, pin it or export it.",
          )}
        </p>
        <span className="dk-axe-k">
          <span className="dk-kbd">⌘J</span>
          {t("opens this panel")}
        </span>
      </div>
    </div>
  );
}
