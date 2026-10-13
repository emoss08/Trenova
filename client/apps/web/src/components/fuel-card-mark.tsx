import { CreditCard01Icon } from "@trenova/shared/components/icons";
import { cn } from "@trenova/shared/lib/utils";
import {
  FUEL_CARD_PROVIDER_LABELS,
  type FuelCardProvider,
} from "@trenova/shared/types/fuel-ifta-enums";

type Box = { x: number; y: number; width: number; height: number };

type NetworkFace = {
  src: string;
  /** The file's own coordinate space: its viewBox for a vector, its pixel size for a raster. */
  source: Pick<Box, "width" | "height">;
  /** The part of the file that is the network's mark, in the same coordinates. */
  mark: Box;
  /** The colour the network prints its cards in. */
  fill: string;
};

/**
 * Each network's mark is the Brandfetch file, used unmodified and cropped at
 * render time to the mark alone: Comdata's file carries an "A Corpay Company"
 * line beneath the wordmark, and EFS's is the WEX | EFS | Fleet One banner. The
 * card colours are Brandfetch's (Comdata's dark colour) or, where Brandfetch
 * lists none, the fill of the brand's own logo (WEX's red). EFS draws its mark
 * in grey and red, so its card is white, the way the network prints it.
 * "Other" is not a network and wears no brand.
 */
const NETWORK_FACES: Partial<Record<FuelCardProvider, NetworkFace>> = {
  WEX: {
    src: "/integrations/logos/wex-logo-dark.svg",
    source: { width: 249.65, height: 73.63 },
    mark: { x: 0, y: 0, width: 249.65, height: 73.63 },
    fill: "#D00A2C",
  },
  Comdata: {
    src: "/integrations/logos/comdata-logo-dark.png",
    source: { width: 323, height: 150 },
    mark: { x: 19, y: 26, width: 286, height: 58 },
    fill: "#0B2A45",
  },
  EFS: {
    src: "/integrations/logos/efs-logo-light.svg",
    source: { width: 237.07, height: 37.08 },
    mark: { x: 84.2, y: 9.3, width: 71.6, height: 17.6 },
    fill: "#FFFFFF",
  },
};

const SIZE_CLASSES = {
  md: "h-6 w-9",
  sm: "h-5 w-8",
} as const;

type FuelCardMarkProps = {
  provider: FuelCardProvider;
  /** `sm` fits inside a form control; `md` sits in a table row or a list. */
  size?: keyof typeof SIZE_CLASSES;
  className?: string;
};

/**
 * A card network drawn the way checkout pages draw Visa and Mastercard: a small
 * card in the network's colour carrying its mark. The colours are the brands'
 * own and stay the same in either theme.
 */
export function FuelCardMark({ provider, size = "md", className }: FuelCardMarkProps) {
  const face = NETWORK_FACES[provider];

  return (
    <span
      role="img"
      aria-label={FUEL_CARD_PROVIDER_LABELS[provider]}
      data-provider={provider}
      className={cn(
        "ring-foreground/10 inline-flex shrink-0 items-center justify-center overflow-hidden rounded-sm px-1 ring-1 ring-inset",
        SIZE_CLASSES[size],
        !face && "bg-muted text-muted-foreground",
        className,
      )}
      style={face ? { backgroundColor: face.fill } : undefined}
    >
      {face ? (
        // The svg takes the crop's proportions, so nothing outside the crop
        // shows above or below it. It is sized inline because the rows and
        // buttons it sits in resize any svg that carries no size class.
        <svg
          aria-hidden
          viewBox={`${face.mark.x} ${face.mark.y} ${face.mark.width} ${face.mark.height}`}
          className="block"
          style={{
            width: "100%",
            height: "auto",
            aspectRatio: `${face.mark.width} / ${face.mark.height}`,
          }}
        >
          <image href={face.src} width={face.source.width} height={face.source.height} />
        </svg>
      ) : (
        <CreditCard01Icon aria-hidden className={size === "sm" ? "size-3.5" : "size-4"} />
      )}
    </span>
  );
}
