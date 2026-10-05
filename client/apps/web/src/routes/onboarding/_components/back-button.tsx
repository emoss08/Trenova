import logoRainbow from "@/assets/logo.webp";
import { ArrowLeftIcon } from "@trenova/shared/components/icons";
import { useT } from "@trenova/shared/i18n/use-t";

/** Icon-only back: an arrow at rest that trades places with the Trenova mark on hover. */
export function BackButton({ hidden, onBack }: { hidden: boolean; onBack: () => void }) {
  const t = useT();
  return (
    <button
      type="button"
      className="nv-bk"
      data-hidden={hidden}
      onClick={onBack}
      disabled={hidden}
      aria-hidden={hidden || undefined}
      tabIndex={hidden ? -1 : undefined}
      aria-label={t("Back")}
      title={t("Back")}
    >
      <span className="nv-bk-ic">
        <img className="nv-bk-l" src={logoRainbow} alt="" />
        <span className="nv-bk-a">
          <ArrowLeftIcon size={15} strokeWidth={1.8} aria-hidden="true" />
        </span>
      </span>
    </button>
  );
}
