import LetterGlitch from "./ui/letter-glitch";

type UnsupportedReason = "mobile" | "safari" | "ie" | "unknown";

interface DeviceInfo {
  isMobile: boolean;
  isSafari: boolean;
  isIE: boolean;
}

interface RequirementsDetails {
  title: string;
  description: string;
  requirements: string[];
}

function getUnsupportedReason(device: DeviceInfo): UnsupportedReason {
  if (device.isMobile) return "mobile";
  if (device.isSafari) return "safari";
  if (device.isIE) return "ie";
  return "unknown";
}

function getReasonDetails(reason: UnsupportedReason): RequirementsDetails {
  switch (reason) {
    case "mobile":
      return {
        title: "Desktop Required",
        description: "Trenova is a web-based system designed for desktop use.",
        requirements: [
          "Desktop or laptop computer",
          "Minimum 1280x720 resolution",
          "Chrome, Edge, or Firefox browser",
        ],
      };
    case "safari":
      return {
        title: "Browser Not Supported",
        description: "Please use a Chromium-based browser",
        requirements: [
          "Chrome (Recommended)",
          "Microsoft Edge",
          "Firefox",
          "Brave or Arc",
        ],
      };
    case "ie":
      return {
        title: "Browser Outdated",
        description: "Internet Explorer is no longer supported",
        requirements: [
          "Chrome (Recommended)",
          "Microsoft Edge",
          "Firefox",
          "Any modern browser",
        ],
      };
    default:
      return {
        title: "Unsupported Configuration",
        description: "Current browser or device is not supported",
        requirements: [
          "Chrome, Edge, or Firefox",
          "Desktop environment",
          "JavaScript enabled",
        ],
      };
  }
}

export function UnsupportedDevice({ device }: { device: DeviceInfo }) {
  const reason = getUnsupportedReason(device);
  const details = getReasonDetails(reason);

  return (
    <div className="flex min-h-screen items-center justify-center bg-background">
      <div className="flex flex-col gap-4">
        {/* Main glitch container */}
        <div className="relative h-screen w-screen rounded-md border border-border">
          <LetterGlitch
            glitchColors={["#9c9c9c", "#696969", "#424242"]}
            glitchSpeed={50}
            centerVignette={true}
            outerVignette={true}
            smooth={true}
          />
          <div className="pointer-events-none absolute inset-0 flex flex-col items-center justify-center gap-2">
            <div className="flex flex-col items-center gap-1">
              <p className="bg-amber-300 px-1 py-0.5 text-center font-table text-sm/none font-medium text-amber-950 uppercase select-none dark:bg-amber-400 dark:text-neutral-900">
                {details.title}
              </p>
              <p className="bg-neutral-900 px-1 py-0.5 text-center font-table text-sm/none font-medium text-white uppercase select-none dark:bg-neutral-500 dark:text-neutral-900">
                {details.description}
              </p>
            </div>
            <RequirementsDetails details={details} />
          </div>
        </div>
      </div>
    </div>
  );
}

function RequirementsDetails({ details }: { details: RequirementsDetails }) {
  return (
    <div className="flex flex-col gap-1">
      <div className="rounded-md border border-border bg-card/5 p-2 backdrop-blur-sm">
        <p className="font-table text-xs font-medium tracking-wide text-muted-foreground uppercase">
          Requirements
        </p>
        <div className="flex flex-wrap gap-2">
          {details.requirements.map((req, index) => (
            <span
              key={index}
              className="inline-flex items-center rounded-sm bg-muted px-2 py-0.5 font-table text-xs text-foreground"
            >
              {req}
            </span>
          ))}
          <div className="flex items-center gap-2 border-t pt-3 sm:border-t-0 sm:border-l sm:pt-0 sm:pl-4">
            <a
              href="mailto:support@trenova.app"
              className="font-table text-xs text-muted-foreground transition-colors hover:text-foreground"
            >
              support@trenova.app
            </a>
          </div>
        </div>
      </div>
    </div>
  );
}
