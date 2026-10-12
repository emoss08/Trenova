import { useRootTheme } from "@/hooks/use-root-theme";
import { oklchToSrgb, parseCssColor, srgbToHex } from "@/lib/oklch";
import { useReducedMotion } from "motion/react";
import {
  createContext,
  use,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
  type RefObject,
} from "react";
import { AuthStageContext, type AuthStageControls } from "./auth-stage-context";
import { GlyphField, type GlyphFieldOptions } from "./glyph-field";
import { readSkyTokens, skyAt, type SkyTheme } from "./sky";
import { useLocalHour } from "./use-local-hour";

const GLYPH_CELL_PX = 12;
const GLYPH_SPEED = 1;
const GLYPH_DOTS = 1;
const FONT_PROPERTY = "--font-mono";
const BACKGROUND_PROPERTY = "--auth-canvas";
const FALLBACK_FONT = "monospace";

type StageState = { field: GlyphField | null; done: boolean };

const StageStateContext = createContext<RefObject<StageState> | null>(null);

function readStageOptions(theme: SkyTheme, hour: number, motion: boolean): GlyphFieldOptions | null {
  const style = getComputedStyle(document.documentElement);
  const tokens = readSkyTokens(style);
  const background = parseCssColor(style.getPropertyValue(BACKGROUND_PROPERTY));
  if (!tokens || !background) {
    return null;
  }

  return {
    cell: GLYPH_CELL_PX,
    speed: GLYPH_SPEED,
    dots: GLYPH_DOTS,
    motion,
    palette: skyAt(hour, theme, tokens).stops,
    background: srgbToHex(oklchToSrgb(background)),
    fontFamily: style.getPropertyValue(FONT_PROPERTY).trim() || FALLBACK_FONT,
  };
}

/**
 * Provides the stage's controls to the screens under it. The controls are stable for
 * the stage's lifetime and act on whichever field is mounted, so a screen that bursts
 * before the field exists — or where none ever will — costs nothing.
 */
export function AuthStage({ children }: { children: ReactNode }) {
  const state = useRef<StageState>({ field: null, done: false });

  const controls = useMemo<AuthStageControls>(
    () => ({
      burst: () => state.current.field?.burst(),
      setDone: (done) => {
        state.current.done = done;
        state.current.field?.setDone(done);
      },
    }),
    [],
  );

  return (
    <StageStateContext value={state}>
      <AuthStageContext value={controls}>{children}</AuthStageContext>
    </StageStateContext>
  );
}

/**
 * The shader canvas. It owns the field's whole life: created once the glyph font has
 * loaded, paused while the tab is hidden, destroyed on unmount. It maps the app's state
 * onto it — the theme and the local hour pick the palette, reduced motion stills it —
 * and the field eases between palettes itself, so nothing here animates.
 *
 * Without WebGL2, or when the shader does not build, a radial wash stands in.
 */
export function AuthStageCanvas() {
  const state = use(StageStateContext);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const optionsRef = useRef<GlyphFieldOptions | null>(null);
  const [fallback, setFallback] = useState(false);
  const theme = useRootTheme();
  const hour = useLocalHour();
  const motion = !useReducedMotion();

  useEffect(() => {
    const options = readStageOptions(theme, hour, motion);
    optionsRef.current = options;
    if (options) {
      state?.current.field?.set(options);
    }
  }, [state, theme, hour, motion]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || !state) {
      return;
    }
    const stageState = state.current;

    let cancelled = false;
    let field: GlyphField | null = null;

    const onVisibilityChange = () => {
      if (document.hidden) {
        field?.pause();
      } else {
        field?.resume();
      }
    };

    const mount = () => {
      if (cancelled) {
        return;
      }
      const options = optionsRef.current;
      field = options ? GlyphField.create(canvas, options) : null;
      if (!field) {
        setFallback(true);
        return;
      }
      field.setDone(stageState.done);
      if (document.hidden) {
        field.pause();
      }
      stageState.field = field;
      document.addEventListener("visibilitychange", onVisibilityChange);
    };

    const font = optionsRef.current?.fontFamily ?? FALLBACK_FONT;
    Promise.resolve(document.fonts?.load(`500 20px ${font}`)).then(mount, mount);

    return () => {
      cancelled = true;
      document.removeEventListener("visibilitychange", onVisibilityChange);
      field?.destroy();
      if (stageState.field === field) {
        stageState.field = null;
      }
    };
  }, [state]);

  return (
    <>
      <canvas ref={canvasRef} aria-hidden="true" className="absolute inset-0 block size-full" />
      {fallback ? <div aria-hidden="true" className="auth-stage-fallback absolute inset-0" /> : null}
    </>
  );
}
