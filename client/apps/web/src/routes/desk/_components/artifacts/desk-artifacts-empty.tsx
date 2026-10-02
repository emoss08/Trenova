import { useT } from "@trenova/shared/i18n/use-t";
import { DeskIcon } from "../desk-icons";

/**
 * The workspace before a conversation has made anything: a small stack of
 * blank cards, what will land here and what can be done with it, and the
 * shortcut that brings the panel back.
 */
export function DeskArtifactsEmpty({ onClose }: { onClose: () => void }) {
  const t = useT();

  return (
    <div className="dk-axe" data-slot="artifacts-empty">
      <button
        type="button"
        className="dk-axe-x"
        title={t("Hide artifacts")}
        aria-label={t("Hide artifacts")}
        onClick={onClose}
      >
        <DeskIcon name="x" size={12} stroke={2} />
      </button>
      <div className="dk-axe-in">
        <div className="dk-axe-art" aria-hidden>
          <span className="dk-axe-c dk-l" />
          <span className="dk-axe-c dk-r" />
          <span className="dk-axe-c dk-f">
            <span className="dk-axe-ic">
              <DeskIcon name="layout" size={13} stroke={1.8} />
            </span>
            <i />
            <i />
          </span>
        </div>
        <b className="dk-axe-t">{t("No artifacts yet")}</b>
        <p className="dk-axe-p">
          {t(
            "When the agent pulls a table, opens a record or drafts an email, it lands here so you can check it, pin it or export it.",
          )}
        </p>
        <span className="dk-axe-k">
          <span className="dk-kbd">⌘J</span> {t("opens this panel")}
        </span>
      </div>
    </div>
  );
}
