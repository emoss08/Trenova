import { brandMarkFor } from "@trenova/shared/components/ui/logos/registry";
import { cn } from "@trenova/shared/lib/utils";

/** The vendors the Desk draws itself, flat and in one colour, at any size. */
const PATHS: Record<string, string> = {
  anthropic:
    "M17.3041 3.541h-3.6718l6.696 16.918H24Zm-10.6082 0L0 20.459h3.7442l1.3693-3.5527h7.0052l1.3693 3.5528h3.7442L10.5363 3.5409Zm-.3712 10.2232 2.2914-5.9456 2.2914 5.9456Z",
  openai:
    "M22.2819 9.8211a5.9847 5.9847 0 0 0-.5157-4.9108 6.0462 6.0462 0 0 0-6.5098-2.9A6.0651 6.0651 0 0 0 4.9807 4.1818a5.9847 5.9847 0 0 0-3.9977 2.9 6.0462 6.0462 0 0 0 .7427 7.0966 5.98 5.98 0 0 0 .511 4.9107 6.051 6.051 0 0 0 6.5146 2.9001A5.9847 5.9847 0 0 0 13.2599 24a6.0557 6.0557 0 0 0 5.7718-4.2058 5.9894 5.9894 0 0 0 3.9977-2.9001 6.0557 6.0557 0 0 0-.7475-7.0729zm-9.022 12.6081a4.4755 4.4755 0 0 1-2.8764-1.0408l.1419-.0804 4.7783-2.7582a.7948.7948 0 0 0 .3927-.6813v-6.7369l2.02 1.1686a.071.071 0 0 1 .038.052v5.5826a4.504 4.504 0 0 1-4.4945 4.4944zm-9.6607-4.1254a4.4708 4.4708 0 0 1-.5346-3.0137l.142.0852 4.783 2.7582a.7712.7712 0 0 0 .7806 0l5.8428-3.3685v2.3324a.0804.0804 0 0 1-.0332.0615L9.74 19.9502a4.4992 4.4992 0 0 1-6.1408-1.6464zM2.3408 7.8956a4.485 4.485 0 0 1 2.3655-1.9728V11.6a.7664.7664 0 0 0 .3879.6765l5.8144 3.3543-2.0201 1.1685a.0757.0757 0 0 1-.071 0l-4.8303-2.7865A4.504 4.504 0 0 1 2.3408 7.872zm16.5963 3.8558L13.1038 8.364 15.1192 7.2a.0757.0757 0 0 1 .071 0l4.8303 2.7913a4.4944 4.4944 0 0 1-.6765 8.1042v-5.6772a.79.79 0 0 0-.407-.667zm2.0107-3.0231l-.142-.0852-4.7735-2.7818a.7759.7759 0 0 0-.7854 0L9.409 9.2297V6.8974a.0662.0662 0 0 1 .0284-.0615l4.8303-2.7866a4.4992 4.4992 0 0 1 6.6802 4.66zM8.3065 12.863l-2.02-1.1638a.0804.0804 0 0 1-.038-.0567V6.0742a4.4992 4.4992 0 0 1 7.3757-3.4537l-.142.0805L8.704 5.459a.7948.7948 0 0 0-.3927.6813zm1.0976-2.3654l2.602-1.4998 2.6069 1.4998v2.9994l-2.5974 1.4997-2.6067-1.4997Z",
  gemini:
    "M11.04 19.32Q12 21.51 12 24q0-2.49.93-4.68.96-2.19 2.58-3.81t3.81-2.55Q21.51 12 24 12q-2.49 0-4.68-.93a12.3 12.3 0 0 1-3.81-2.58 12.3 12.3 0 0 1-2.58-3.81Q12 2.49 12 0q0 2.49-.96 4.68-.93 2.19-2.55 3.81a12.3 12.3 0 0 1-3.81 2.58Q2.49 12 0 12q2.49 0 4.68.96 2.19.93 3.81 2.55t2.55 3.81",
  mistral:
    "M17.143 3.429v3.428h-3.429v3.429h-3.428V6.857H6.857V3.43H3.43v13.714H0v3.428h10.286v-3.428H6.857v-3.429h3.429v3.429h3.429v-3.429h3.428v3.429h-3.428v3.428H24v-3.428h-3.43V3.429z",
  groq: "m18.445 4.406-9.468 13.74 7.341.665-1.69 9.578 9.469-13.74-7.342-.664 1.69-9.579Z",
};

const VIEW_BOX: Record<string, string> = { groq: "0.54 0.39 32 32" };

/** What a vendor is called where a person reads it. */
export const VENDOR_NAMES: Record<string, string> = {
  anthropic: "Anthropic",
  openai: "OpenAI",
  gemini: "Google",
  groq: "Groq",
  mistral: "Mistral",
  openrouter: "OpenRouter",
  together: "Together",
  fireworks: "Fireworks",
  deepseek: "DeepSeek",
  bedrock: "AWS Bedrock",
  huggingface: "Hugging Face",
  ollama: "Ollama",
};

/** The shared logo set's marks for the vendors the Desk does not draw itself. */
const SHARED_MARKS = new Map(
  ["openrouter", "together", "fireworks", "bedrock", "ollama", "vllm", "lmstudio"].flatMap((key) => {
    const Mark = brandMarkFor({ presetKey: key });
    return Mark ? [[key, <Mark key={key} className="size-full" />] as const] : [];
  }),
);

/** The vendors the Auto reel and rain draw, in the order they turn up. */
export const REEL_VENDORS = ["anthropic", "openai", "gemini", "groq", "mistral"] as const;

/**
 * The gradient Gemini's mark is filled with. Drawn once per picker; the
 * stylesheet points the mark's path at it.
 */
export function GeminiGradient() {
  return (
    <svg width="0" height="0" className="absolute" aria-hidden>
      <defs>
        <linearGradient id="gemGrad" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" style={{ stopColor: "var(--dsk-gem-1)" }} />
          <stop offset=".55" style={{ stopColor: "var(--dsk-gem-2)" }} />
          <stop offset="1" style={{ stopColor: "var(--dsk-gem-3)" }} />
        </linearGradient>
      </defs>
    </svg>
  );
}

/** Auto's own mark: a diamond with a point in it, the router's choice. */
function AutoMark({ size }: { size: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinejoin="round"
      className="dk-mk-auto"
      aria-hidden
    >
      <path d="M2.7 10.3a2.41 2.41 0 0 0 0 3.41l7.59 7.59a2.41 2.41 0 0 0 3.41 0l7.59-7.59a2.41 2.41 0 0 0 0-3.41l-7.59-7.59a2.41 2.41 0 0 0-3.41 0Z" />
      <circle cx="12" cy="12" r="2" fill="currentColor" stroke="none" />
    </svg>
  );
}

/** A plain chip for an endpoint nobody publishes, such as a self-hosted model. */
function GenericMark({ size }: { size: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
    >
      <rect x="5" y="5" width="14" height="14" rx="2" />
      <path d="M9 2v3M15 2v3M9 19v3M15 19v3M2 9h3M2 15h3M19 9h3M19 15h3" />
    </svg>
  );
}

/**
 * The mark of the company behind a model: the Desk's own drawing for the
 * vendors it shows most, the shared logo set for the rest, and a plain chip
 * for an endpoint no vendor publishes.
 */
export function DeskBrandMark({ vendor, size = 14 }: { vendor: string; size?: number }) {
  if (vendor === "auto") {
    return <AutoMark size={size} />;
  }
  const path = PATHS[vendor];
  if (path) {
    return (
      <svg
        width={size}
        height={size}
        viewBox={VIEW_BOX[vendor] ?? "0 0 24 24"}
        fill="currentColor"
        className={cn("dk-mk", `dk-mk-${vendor}`)}
        aria-hidden
      >
        <path d={path} />
      </svg>
    );
  }
  const shared = SHARED_MARKS.get(vendor);
  if (shared) {
    return (
      <span className="grid place-items-center" style={{ width: size, height: size }}>
        {shared}
      </span>
    );
  }
  return <GenericMark size={size} />;
}
