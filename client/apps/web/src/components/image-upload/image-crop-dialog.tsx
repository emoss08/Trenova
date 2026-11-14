import {
  AlertCircleIcon,
  ImagePlusIcon,
  ImageXIcon,
  Moon01Icon,
  RefreshCw01Icon,
  SunIcon,
  ZoomInIcon,
  ZoomOutIcon,
} from "@trenova/shared/components/icons";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Slider } from "@trenova/shared/components/ui/slider";
import { Tooltip, TooltipContent, TooltipTrigger } from "@trenova/shared/components/ui/tooltip";
import { useElementSize } from "@trenova/shared/hooks/use-element-size";
import { useObjectUrl } from "@trenova/shared/hooks/use-object-url";
import { useT } from "@trenova/shared/i18n/use-t";
import { describeError } from "@trenova/shared/lib/error-presentation";
import { cn } from "@trenova/shared/lib/utils";
import {
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type ReactNode,
  type RefObject,
} from "react";
import Cropper from "react-easy-crop";
import "react-easy-crop/react-easy-crop.css";
import { CropDrawingProvider, type CropDrawing } from "./crop-result";
import {
  cropArea,
  cropLayout,
  croppedFile,
  outputSize,
  reducedSource,
  stepToZoom,
  turned,
  withinSource,
  zoomToStep,
  type Area,
  type CropSource,
  type Point,
  type Size,
} from "./image-crop";
import { SVG_TYPE } from "./image-file";
import type { ImageSpec } from "./image-spec";
import { sizedSvg } from "./svg-source";

/** The longest side of the copy the previews are drawn from. */
const PREVIEW_SOURCE_PIXELS = 1024;

/** Below this many pixels on its longest side a result looks soft wherever it is enlarged. */
const SOFT_BELOW_PIXELS = 128;

const ZOOM_BUTTON_STEP = 10;

/**
 * The cropper adds a fixed amount of zoom per wheel notch, which is a leap for a picture
 * zoomed far out and nothing for one zoomed far in. Scaling its speed by the zoom makes a
 * notch a quarter more or less at any zoom.
 */
const WHEEL_ZOOM_RATIO = 0.5;
const KEYBOARD_STEP_PIXELS = 8;
const ORIGIN: Point = { x: 0, y: 0 };

/** How round a tile's corners are, as a share of its side: the sidebar's own tile. */
const TILE_RADIUS = "22%";

export type ImageCropDialogProps = {
  open: boolean;
  /** The picture being positioned. It stays while the dialog closes, so the leave has something to draw. */
  file: File | null;
  spec: ImageSpec;
  /**
   * The places the result is drawn, shown beside the stage as the picture moves, with a
   * `CropResult` wherever one draws it.
   */
  preview?: ReactNode;
  title: string;
  confirmLabel: string;
  /** Why the replacement last offered was turned away. */
  problem?: string | null;
  /** Asks for another picture in place of this one. */
  onReplace: () => void;
  onClose: () => void;
  /** Uploads the result. The dialog holds until it settles and stays open when it fails. */
  onConfirm: (file: File) => Promise<void>;
};

export function ImageCropDialog({
  open,
  file,
  title,
  onClose,
  ...workspace
}: ImageCropDialogProps) {
  const t = useT();
  const [busy, setBusy] = useState(false);
  const [pick, setPick] = useState({ file, count: 0 });
  const stageRef = useRef<HTMLDivElement>(null);

  // Every picture offered starts from a clean workspace, the same file picked twice included.
  if (pick.file !== file) {
    setPick({ file, count: pick.count + 1 });
  }

  return (
    <Dialog open={open} onOpenChange={(next) => !next && !busy && onClose()}>
      {/* Focus starts on the stage, where the arrow keys move the picture. Left to itself it
          lands on the first control, the background switch, which then wears a ring the
          person did nothing to ask for. */}
      <DialogContent size={workspace.preview ? "xl" : "md"} initialFocus={stageRef}>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>
            {t("Drag to move the picture. Scroll or use the slider to zoom.")}
          </DialogDescription>
        </DialogHeader>
        {file ? (
          <CropWorkspace
            key={pick.count}
            file={file}
            busy={busy}
            stageRef={stageRef}
            onBusyChange={setBusy}
            onClose={onClose}
            {...workspace}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

type Loaded = {
  /** The picture's own size in pixels. */
  size: Size;
  /** The picture at full size, which the upload is drawn from. */
  full: CropSource;
  /** A reduced copy the previews are drawn from, and the scale it was reduced by. */
  reduced: { source: CropSource; scale: number };
};

type View = {
  quarterTurns: number;
  position: Point;
  /** Null until the person zooms: the frame's own starting zoom. */
  zoom: number | null;
};

const RESTING: View = { quarterTurns: 0, position: ORIGIN, zoom: null };

type Ground = "light" | "dark";

function CropWorkspace({
  file,
  spec,
  preview,
  confirmLabel,
  problem,
  busy,
  stageRef: focusRef,
  onBusyChange,
  onReplace,
  onClose,
  onConfirm,
}: Omit<ImageCropDialogProps, "open" | "file" | "title"> & {
  file: File;
  busy: boolean;
  /** Takes the dialog's first focus. */
  stageRef: RefObject<HTMLDivElement | null>;
  onBusyChange: (busy: boolean) => void;
}) {
  const t = useT();
  const [stageRef, stage] = useElementSize<HTMLDivElement>();
  const cropperRef = useRef<Cropper>(null);
  const sending = useRef(false);
  const [drawable, setDrawable] = useState<File | null>(file.type === SVG_TYPE ? null : file);
  const [loaded, setLoaded] = useState<Loaded | null>(null);
  const [unreadable, setUnreadable] = useState(false);
  const [failure, setFailure] = useState<string | null>(null);
  const [view, setView] = useState<View>(RESTING);
  const [ground, setGround] = useState<Ground>(() =>
    window.document.documentElement.classList.contains("dark") ? "dark" : "light",
  );
  const url = useObjectUrl(drawable);

  useEffect(() => {
    if (file.type !== SVG_TYPE) return;
    let stale = false;
    sizedSvg(file).then(
      (sized) => {
        if (!stale) setDrawable(sized);
      },
      () => {
        if (!stale) setUnreadable(true);
      },
    );
    return () => {
      stale = true;
    };
  }, [file]);

  useEffect(() => {
    if (!url) return;
    let stale = false;
    const image = new Image();
    image.decoding = "async";
    image.src = url;
    image.decode().then(
      () => {
        if (stale) return;
        const size = { width: image.naturalWidth, height: image.naturalHeight };
        if (size.width === 0 || size.height === 0) {
          setUnreadable(true);
          return;
        }
        setLoaded({
          size,
          full: { image, size },
          reduced: reducedSource(image, size, PREVIEW_SOURCE_PIXELS),
        });
      },
      () => {
        if (!stale) setUnreadable(true);
      },
    );
    return () => {
      stale = true;
    };
  }, [url]);

  const layout =
    loaded && stage.width > 0 && stage.height > 0
      ? cropLayout(loaded.size, stage, spec.frame, view.quarterTurns, spec.fit)
      : null;
  const range = layout?.range ?? null;
  const zoom = range ? Math.min(Math.max(view.zoom ?? range.initial, range.min), range.max) : 1;
  const frameWidth = layout?.frame.width;
  const frameHeight = layout?.frame.height;

  // The cropper acts on one changed prop per update, the zoom ahead of the frame's size, so
  // a frame that changes size together with its zoom would keep its old box. Measuring again
  // whenever the frame changes takes the new one.
  useLayoutEffect(() => {
    const cropper = cropperRef.current;
    if (cropper?.state.cropSize) {
      cropper.computeSizes();
    }
  }, [frameWidth, frameHeight]);

  let area: Area | null = null;
  if (loaded && layout) {
    const inFrame = cropArea({
      position: view.position,
      zoom,
      quarterTurns: view.quarterTurns,
      source: loaded.size,
      media: layout.media,
      frame: layout.frame,
    });
    area =
      spec.fit === "cover"
        ? withinSource(inFrame, turned(loaded.size, view.quarterTurns))
        : inFrame;
  }

  const drawing: CropDrawing | null =
    loaded && area
      ? {
          source: loaded.reduced.source,
          scale: loaded.reduced.scale,
          area,
          quarterTurns: view.quarterTurns,
        }
      : null;
  const output = area ? outputSize(area, spec.maxDimension) : null;
  const soft = output ? Math.max(output.width, output.height) < SOFT_BELOW_PIXELS : false;
  const moved =
    view.zoom !== null || view.quarterTurns !== 0 || view.position.x !== 0 || view.position.y !== 0;
  const idle = !range || busy;
  const grounded = spec.fit === "contain";

  const zoomBy = (steps: number) => {
    if (!range) return;
    const step = Math.min(Math.max(zoomToStep(zoom, range) + steps, 0), 100);
    setView((current) => ({ ...current, zoom: stepToZoom(step, range) }));
  };

  const confirm = async () => {
    if (!loaded || !area || sending.current) return;
    sending.current = true;
    onBusyChange(true);
    setFailure(null);
    try {
      let result: File;
      try {
        result = await croppedFile(
          loaded.full,
          area,
          view.quarterTurns,
          spec.maxDimension,
          file.name,
        );
      } catch {
        setFailure(t("That picture could not be prepared. Choose another one."));
        return;
      }
      try {
        await onConfirm(result);
      } catch (error) {
        setFailure(describeError(error).detail ?? t("The upload did not go through. Try again."));
        return;
      }
      onClose();
    } finally {
      sending.current = false;
      onBusyChange(false);
    }
  };

  const notice = failure ?? problem ?? null;

  return (
    <>
      <div className={cn("-mx-4 grid border-t", preview && "sm:grid-cols-[minmax(0,1fr)_24rem]")}>
        <div className="flex min-w-0 flex-col">
          <div
            ref={(node) => {
              focusRef.current = node;
              return stageRef(node);
            }}
            tabIndex={-1}
            onKeyDown={(event) => {
              // Focus starts on the stage, without a ring. The first arrow key hands it to the
              // frame, which is what moves the picture, and is passed on so that press counts.
              const cropper = cropperRef.current;
              if (
                event.target !== event.currentTarget ||
                !cropper ||
                !event.key.startsWith("Arrow")
              ) {
                return;
              }
              cropper.cropperRef.current?.focus({ preventScroll: true });
              cropper.onKeyDown(event);
            }}
            className={cn(
              "relative h-80 overflow-hidden outline-none",
              !grounded && "bg-sunken",
              grounded && ground === "light" && "bg-doc-paper",
              grounded && ground === "dark" && "bg-doc-ink",
            )}
          >
            {unreadable ? (
              <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 bg-sunken px-6 text-center text-sm text-foreground-muted">
                <ImageXIcon className="size-5 text-foreground-subtle" />
                {t("That file could not be opened as a picture. Choose another one.")}
              </div>
            ) : url && layout ? (
              <div
                className={cn(
                  "absolute inset-0 animate-in fade-in-0",
                  busy && "pointer-events-none",
                )}
              >
                <Cropper
                  ref={cropperRef}
                  image={url}
                  crop={view.position}
                  zoom={zoom}
                  rotation={view.quarterTurns * 90}
                  aspect={layout.frame.width / layout.frame.height}
                  cropSize={layout.frame}
                  minZoom={layout.range.min}
                  maxZoom={layout.range.max}
                  cropShape={spec.mask === "round" ? "round" : "rect"}
                  showGrid={false}
                  zoomSpeed={zoom * WHEEL_ZOOM_RATIO}
                  keyboardStep={KEYBOARD_STEP_PIXELS}
                  disableAutomaticStylesInjection
                  onCropChange={(position) => setView((current) => ({ ...current, position }))}
                  onZoomChange={(next) => setView((current) => ({ ...current, zoom: next }))}
                  mediaProps={{ draggable: false, onError: () => setUnreadable(true) }}
                  cropperProps={{
                    role: "group",
                    "aria-label": t("Picture position. The arrow keys move the picture."),
                  }}
                  style={{
                    cropAreaStyle: spec.mask === "tile" ? { borderRadius: TILE_RADIUS } : undefined,
                  }}
                  classes={{
                    // The frame's own box shadow is what dims the stage around it, so the shared
                    // focus ring, a box shadow too, would take the dimming away.
                    // design-tokens-ignore: the ring is an outline here, in the ring's own colour
                    cropAreaClassName: "outline-ring outline-offset-2 focus-visible:outline-2",
                  }}
                />
              </div>
            ) : null}
            {grounded && !unreadable ? (
              <SegmentedControl
                aria-label={t("Background")}
                className="absolute top-2 right-2"
                labelClassName="sr-only"
                items={[
                  { value: "light", label: t("Light"), icon: SunIcon },
                  { value: "dark", label: t("Dark"), icon: Moon01Icon },
                ]}
                value={ground}
                onValueChange={setGround}
              />
            ) : null}
          </div>

          {unreadable ? null : (
            <div className="flex items-center gap-1 border-t px-4 py-2.5">
              <IconControl
                label={t("Zoom out")}
                disabled={idle}
                onClick={() => zoomBy(-ZOOM_BUTTON_STEP)}
              >
                <ZoomOutIcon />
              </IconControl>
              <Slider
                aria-label={t("Zoom")}
                min={0}
                max={100}
                step={1}
                disabled={idle}
                value={range ? zoomToStep(zoom, range) : 0}
                onValueChange={(step) => {
                  if (!range) return;
                  setView((current) => ({ ...current, zoom: stepToZoom(step, range) }));
                }}
                className="min-w-0 flex-1 px-1.5"
              />
              <IconControl
                label={t("Zoom in")}
                disabled={idle}
                onClick={() => zoomBy(ZOOM_BUTTON_STEP)}
              >
                <ZoomInIcon />
              </IconControl>
              <span aria-hidden className="mx-2 h-4 w-px bg-border" />
              <IconControl
                label={t("Rotate")}
                disabled={idle}
                onClick={() =>
                  setView((current) => ({
                    quarterTurns: (current.quarterTurns + 1) % 4,
                    position: ORIGIN,
                    zoom: null,
                  }))
                }
              >
                <RefreshCw01Icon />
              </IconControl>
              <Button
                type="button"
                variant="ghost"
                size="sm"
                disabled={idle || !moved}
                onClick={() => setView(RESTING)}
              >
                {t("Reset")}
              </Button>
            </div>
          )}
        </div>
        {preview && !unreadable ? (
          <div className="relative min-h-72 border-t bg-card sm:border-t-0 sm:border-l">
            {/* The preview takes the height the stage and its controls set, whatever its own
                content would ask for: a document drawn small still measures full size. */}
            <div className="absolute inset-4 flex">
              <CropDrawingProvider drawing={drawing}>{preview}</CropDrawingProvider>
            </div>
          </div>
        ) : null}
        {notice ? (
          <div className="col-span-full border-t px-4 py-3">
            <Alert variant="destructive" size="sm" role="alert">
              <AlertCircleIcon />
              <AlertDescription>{notice}</AlertDescription>
            </Alert>
          </div>
        ) : soft ? (
          <div className="col-span-full border-t px-4 py-3">
            <Alert variant="warning" size="sm">
              <AlertCircleIcon />
              <AlertDescription>
                {t("This selection is small, so it may look soft where it is shown larger.")}
              </AlertDescription>
            </Alert>
          </div>
        ) : null}
      </div>

      <DialogFooter className="-mt-4">
        <Button
          type="button"
          variant="ghost"
          className="sm:mr-auto"
          disabled={busy}
          onClick={onReplace}
        >
          <ImagePlusIcon />
          {t("Choose another")}
        </Button>
        <Button type="button" variant="outline" disabled={busy} onClick={onClose}>
          {t("Cancel")}
        </Button>
        <Button
          type="button"
          isLoading={busy}
          disabled={!area || unreadable}
          onClick={() => void confirm()}
        >
          {confirmLabel}
        </Button>
      </DialogFooter>
    </>
  );
}

function IconControl({
  label,
  disabled,
  onClick,
  children,
}: {
  label: string;
  disabled: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger
        render={
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label={label}
            disabled={disabled}
            onClick={onClick}
          />
        }
      >
        {children}
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}
