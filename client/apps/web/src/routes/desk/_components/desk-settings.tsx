import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { queries } from "@/lib/queries";
import { useDeskSettingsStore, type DeskSettings } from "@/stores/desk-settings-store";
import { useDeskStore } from "@/stores/desk-store";
import { useQuery } from "@tanstack/react-query";
import { useTheme } from "@trenova/shared/components/theme-provider";
import { Button } from "@trenova/shared/components/ui/button";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import {
  useCallback,
  useEffect,
  useId,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { DeskIcon, type DeskIconName } from "@/components/desk-chat/desk-icons";
import { deskIconClass } from "@/components/desk-chat/desk-button-styles";
import { usePermissions } from "@/hooks/use-permission";
import { Resource } from "@trenova/shared/types/permission";
import { CaseChecklistsSection } from "./settings/case-checklists/case-checklists-section";
import { Seg } from "./settings/desk-seg";
import { onRadioArrows, useModalFocus } from "./use-modal-focus";

type PersonalSection = "appearance" | "conversation" | "composer" | "files" | "artifacts" | "agent";

/**
 * A section of the Desk's settings: the six that are the person's own, then
 * the organization's, which only someone who may read billing control sees.
 */
export type DeskSettingsSection = PersonalSection | "checklists";

type Section = DeskSettingsSection;

const SECTION_KEYS: readonly DeskSettingsSection[] = [
  "appearance",
  "conversation",
  "composer",
  "files",
  "artifacts",
  "agent",
  "checklists",
];

/** Whether a value names a settings section, as a link to the Desk may. */
export function isDeskSettingsSection(value: string): value is DeskSettingsSection {
  return (SECTION_KEYS as readonly string[]).includes(value);
}

const SECTIONS: Array<[PersonalSection, DeskIconName]> = [
  ["appearance", "eye"],
  ["conversation", "chat"],
  ["composer", "plus"],
  ["files", "copy"],
  ["artifacts", "table"],
  ["agent", "shield"],
];

function sectionLabel(section: Section, t: TranslateFn): string {
  switch (section) {
    case "appearance":
      return t("Appearance");
    case "conversation":
      return t("Conversation");
    case "composer":
      return t("Composer");
    case "files":
      return t("Files & scanning");
    case "artifacts":
      return t("Artifacts");
    case "agent":
      return t("Agent & approvals");
    case "checklists":
      return t("Case checklists");
  }
}

function Row({
  title,
  detail,
  wide,
  children,
}: {
  title: string;
  detail: string;
  wide?: boolean;
  children: ReactNode;
}) {
  return (
    <div className={cn("dk-sx-row", wide && "dk-wide")}>
      <div className="dk-sx-l">
        <b>{title}</b>
        <span>{detail}</span>
      </div>
      <div className="dk-sx-c">{children}</div>
    </div>
  );
}

const WIDTHS = ["narrow", "default", "wide"] as const;

const WIDTH_PREVIEW: Record<DeskSettings["width"], number> = { narrow: 46, default: 62, wide: 84 };

function WidthPreview({ width }: { width: DeskSettings["width"] }) {
  return (
    <div className="dk-sx-wp" aria-hidden>
      <span className="dk-sx-wp-sb" />
      <span className="dk-sx-wp-main">
        <span className="dk-sx-wp-col" style={{ width: `${WIDTH_PREVIEW[width]}%` }}>
          <i style={{ width: "58%", alignSelf: "flex-end" }} />
          <i />
          <i style={{ width: "82%" }} />
          <i style={{ width: "64%" }} />
          <b />
        </span>
      </span>
    </div>
  );
}

/**
 * The Desk's settings: how it looks, how a message is sent and read, what the
 * composer offers, where files and scans go, how artifacts arrive and how an
 * approval feels. Each change applies at once and stays with this browser.
 */
export function DeskSettingsDialog({
  agents,
  initialSection = "appearance",
  onClose,
}: {
  agents: readonly AgentChoice[];
  /** The section it opens on, such as Case checklists from a link. */
  initialSection?: DeskSettingsSection;
  onClose: () => void;
}) {
  const t = useT();
  const { theme, setTheme } = useTheme();
  const settings = useDeskSettingsStore((state) => state.settings);
  const set = useDeskSettingsStore((state) => state.set);
  const reset = useDeskSettingsStore((state) => state.reset);
  const setSharePage = useDeskStore((state) => state.setSharePage);
  const checklistAccess = usePermissions(Resource.BillingControl);
  const [chosen, setSection] = useState<Section>(initialSection);
  // The organization's section is only there for someone who may read
  // billing control; asked for without it, the dialog opens on the first.
  const section: Section =
    chosen === "checklists" && !checklistAccess.canRead ? "appearance" : chosen;
  const organizational = section === "checklists";
  const [closing, setClosing] = useState(false);
  const panelRef = useRef<HTMLDivElement>(null);
  const titleId = useId();
  useModalFocus(panelRef);

  const devicesQuery = useQuery({
    ...queries.capture.myDevices("Active"),
    enabled: section === "files",
  });
  const profilesQuery = useQuery({
    ...queries.capture.availableProfiles(),
    enabled: section === "files",
  });

  const close = useCallback(() => {
    setClosing(true);
    window.setTimeout(onClose, 160);
  }, [onClose]);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.stopPropagation();
        close();
      }
    };
    window.addEventListener("keydown", onKey, true);
    panelRef.current?.focus();
    return () => window.removeEventListener("keydown", onKey, true);
  }, [close]);

  return (
    <div
      className={cn("dk-srch-wrap dk-sx-wrap", closing && "dk-out")}
      onMouseDown={(event) => event.target === event.currentTarget && close()}
    >
      <div
        className={cn("dk-sx", organizational && "dk-xl")}
        role="dialog"
        aria-modal
        aria-labelledby={titleId}
        ref={panelRef}
        tabIndex={-1}
      >
        <div className="dk-sx-top">
          <h2 id={titleId}>{t("Settings")}</h2>
          <Button
            variant="quiet"
            size="icon-sm"
            className={deskIconClass}
            onClick={close}
            title={t("Close")}
            aria-label={t("Close")}
          >
            <DeskIcon name="x" size={14} />
          </Button>
        </div>
        <nav className="dk-sx-nav">
          {SECTIONS.map(([key, icon]) => (
            <Button
              key={key}
              variant="bare"
              size="bare"
              className={cn(
                "flex h-8 gap-2.5 rounded-lg px-2.5 text-left text-sm text-dsk-muted transition-colors duration-150 hover:bg-dsk-hover hover:text-dsk-fg [&_svg]:text-dsk-subtle",
                section === key && "bg-dsk-hover font-medium text-dsk-fg [&_svg]:text-dsk-fg",
              )}
              aria-current={section === key ? "page" : undefined}
              onClick={() => setSection(key)}
            >
              <DeskIcon name={icon} size={14} />
              {sectionLabel(key, t)}
            </Button>
          ))}
          {checklistAccess.canRead && (
            <>
              <div className="dk-sx-ng">{t("Organization")}</div>
              <Button
                variant="bare"
                size="bare"
                className={cn(
                  "flex h-8 gap-2.5 rounded-lg px-2.5 text-left text-sm whitespace-nowrap text-dsk-muted transition-colors duration-150 hover:bg-dsk-hover hover:text-dsk-fg [&_svg]:text-dsk-subtle",
                  organizational && "bg-dsk-hover font-medium text-dsk-fg [&_svg]:text-dsk-fg",
                )}
                aria-current={organizational ? "page" : undefined}
                onClick={() => setSection("checklists")}
              >
                <DeskIcon name="check" size={14} />
                {sectionLabel("checklists", t)}
              </Button>
            </>
          )}
          <span className="flex-1" />
          {!organizational && (
          <Button
            variant="bare"
            size="bare"
            className="flex h-7 rounded-lg px-2.5 text-left text-xs font-normal text-dsk-subtle transition-colors duration-150 hover:text-dsk-fg"
            onClick={() => {
              reset();
              setSharePage(true);
              setTheme("system");
            }}
          >
            {t("Reset to defaults")}
          </Button>
          )}
        </nav>
        {organizational ? (
          <div className="dk-sx-body dk-ck-body" key={section}>
            <CaseChecklistsSection readOnly={!checklistAccess.canUpdate} />
          </div>
        ) : (
        <div className="dk-sx-body" key={section}>
          <div className="dk-sx-head">{sectionLabel(section, t)}</div>
          {section === "appearance" && (
            <>
              <Row title={t("Theme")} detail={t("Match your system, or pick light or dark.")}>
                <Seg
                  label={t("Theme")}
                  value={theme}
                  onChange={setTheme}
                  options={[
                    ["system", t("System")],
                    ["light", t("Light")],
                    ["dark", t("Dark")],
                  ]}
                />
              </Row>
              <Row
                title={t("Conversation width")}
                detail={t(
                  "How wide messages run. Wide fits more of a table or long reply per line.",
                )}
                wide
              >
                <div className="dk-sx-wopts" role="radiogroup" aria-label={t("Conversation width")}>
                  {WIDTHS.map((width) => (
                    <Button
                      key={width}
                      variant="bare"
                      size="bare"
                      role="radio"
                      aria-checked={settings.width === width}
                      tabIndex={settings.width === width ? 0 : -1}
                      className={cn(
                        "dk-sx-wo flex flex-col items-stretch gap-2 text-left text-sm text-dsk-muted transition-colors duration-150 hover:text-dsk-fg aria-checked:font-medium aria-checked:text-dsk-fg",
                        settings.width === width && "dk-on",
                      )}
                      onClick={() => set("width", width)}
                      onKeyDown={(event) =>
                        onRadioArrows(event, WIDTHS, settings.width, (next) => set("width", next))
                      }
                    >
                      <WidthPreview width={width} />
                      <span>
                        {width === "narrow"
                          ? t("Narrow")
                          : width === "wide"
                            ? t("Wide")
                            : t("Default")}
                      </span>
                    </Button>
                  ))}
                </div>
              </Row>
              <Row
                title={t("Motion")}
                detail={t(
                  "Reduced turns off confetti, the working border, typing effects and shimmer.",
                )}
              >
                <Seg
                  label={t("Motion")}
                  value={settings.motion}
                  onChange={(value) => set("motion", value)}
                  options={[
                    ["full", t("Full")],
                    ["reduced", t("Reduced")],
                  ]}
                />
              </Row>
              <Row title={t("Text size")} detail={t("Size of message text in the conversation.")}>
                <Seg
                  label={t("Text size")}
                  value={settings.text}
                  onChange={(value) => set("text", value)}
                  options={[
                    ["small", t("Small")],
                    ["medium", t("Medium")],
                    ["large", t("Large")],
                  ]}
                />
              </Row>
            </>
          )}
          {section === "conversation" && (
            <>
              <Row
                title={t("Send with")}
                detail={t("Which keys send a message. The other inserts a new line.")}
              >
                <Seg
                  label={t("Send with")}
                  value={settings.send}
                  onChange={(value) => set("send", value)}
                  options={[
                    ["enter", t("Enter")],
                    ["mod", t("⌘ Enter")],
                  ]}
                />
              </Row>
              <Row
                title={t("Source numbers")}
                detail={t(
                  "Small numbers in replies that show which tool call each fact came from.",
                )}
              >
                <Seg
                  label={t("Source numbers")}
                  value={settings.refs}
                  onChange={(value) => set("refs", value)}
                  options={[
                    ["on", t("Show")],
                    ["off", t("Hide")],
                  ]}
                />
              </Row>
              <Row title={t("Open Desk to")} detail={t("What you see when you open Desk.")}>
                <Seg
                  label={t("Open Desk to")}
                  value={settings.start}
                  onChange={(value) => set("start", value)}
                  options={[
                    ["today", t("Today")],
                    ["last", t("Last conversation")],
                  ]}
                />
              </Row>
            </>
          )}
          {section === "composer" && (
            <>
              <Row
                title={t("Start new conversations with")}
                detail={t(
                  "The agent the composer picks when you start fresh. You can still switch per message.",
                )}
              >
                <select
                  className="dk-sx-select"
                  value={settings.agent}
                  onChange={(event) => set("agent", event.target.value)}
                  aria-label={t("Start new conversations with")}
                >
                  <option value="">{t("The last one I used")}</option>
                  {agents.map((agent) => (
                    <option key={agent.id} value={agent.id}>
                      {agent.name}
                    </option>
                  ))}
                </select>
              </Row>
              <Row
                title={t("Share the page you came from")}
                detail={t(
                  "Send the page you came to Desk from with each message so the agent knows what you were looking at.",
                )}
              >
                <Seg
                  label={t("Share the page you came from")}
                  value={settings.sharePage}
                  onChange={(value) => {
                    set("sharePage", value);
                    setSharePage(value === "on");
                  }}
                  options={[
                    ["on", t("By default")],
                    ["off", t("Ask me")],
                  ]}
                />
              </Row>
              <Row
                title={t("@ mentions")}
                detail={t("Type @ to reference a shipment, customer, invoice, driver or carrier.")}
              >
                <Seg
                  label={t("@ mentions")}
                  value={settings.mentions}
                  onChange={(value) => set("mentions", value)}
                  options={[
                    ["on", t("On")],
                    ["off", t("Off")],
                  ]}
                />
              </Row>
              <Row
                title={t("Slash commands")}
                detail={t("Type / for /status, /quote, /report and /explain.")}
              >
                <Seg
                  label={t("Slash commands")}
                  value={settings.slash}
                  onChange={(value) => set("slash", value)}
                  options={[
                    ["on", t("On")],
                    ["off", t("Off")],
                  ]}
                />
              </Row>
              <Row title={t("Dictation")} detail={t("Show the microphone in the composer.")}>
                <Seg
                  label={t("Dictation")}
                  value={settings.mic}
                  onChange={(value) => set("mic", value)}
                  options={[
                    ["on", t("Show")],
                    ["off", t("Hide")],
                  ]}
                />
              </Row>
              <Row
                title={t("Suggested questions")}
                detail={t("Type out example questions in the empty composer on Today.")}
              >
                <Seg
                  label={t("Suggested questions")}
                  value={settings.presets}
                  onChange={(value) => set("presets", value)}
                  options={[
                    ["on", t("On")],
                    ["off", t("Off")],
                  ]}
                />
              </Row>
            </>
          )}
          {section === "files" && (
            <>
              <Row
                title={t("Drop files")}
                detail={t("Where dropping a file attaches it to your message.")}
              >
                <Seg
                  label={t("Drop files")}
                  value={settings.drop}
                  onChange={(value) => set("drop", value)}
                  options={[
                    ["page", t("Anywhere")],
                    ["composer", t("On the composer")],
                  ]}
                />
              </Row>
              <Row
                title={t("Scan with")}
                detail={t(
                  "The computer Scan from Capture starts on. Only your own paired computers are listed.",
                )}
              >
                <select
                  className="dk-sx-select"
                  value={settings.scanDevice}
                  onChange={(event) => set("scanDevice", event.target.value)}
                  aria-label={t("Scan with")}
                >
                  <option value="">{t("Ask each time")}</option>
                  {(devicesQuery.data ?? []).map((device) => (
                    <option key={device.id} value={device.id}>
                      {device.name}
                    </option>
                  ))}
                </select>
              </Row>
              <Row
                title={t("Scan settings")}
                detail={t(
                  "The scan profile to start from. Your admin manages profiles in Scanning and printing.",
                )}
              >
                <select
                  className="dk-sx-select"
                  value={settings.scanProfile}
                  onChange={(event) => set("scanProfile", event.target.value)}
                  aria-label={t("Scan settings")}
                >
                  <option value="">{t("Organization default")}</option>
                  {(profilesQuery.data ?? []).map((profile) => (
                    <option key={profile.id} value={profile.id}>
                      {profile.name}
                    </option>
                  ))}
                </select>
              </Row>
            </>
          )}
          {section === "artifacts" && (
            <>
              <Row
                title={t("Open new artifacts")}
                detail={t("When an agent makes a table, record or draft.")}
              >
                <Seg
                  label={t("Open new artifacts")}
                  value={settings.autoOpen}
                  onChange={(value) => set("autoOpen", value)}
                  options={[
                    ["on", t("Right away")],
                    ["off", t("When I click")],
                  ]}
                />
              </Row>
              <Row
                title={t("New artifact dot")}
                detail={t(
                  "Mark Workspace in the top bar when something new arrives while it's closed.",
                )}
              >
                <Seg
                  label={t("New artifact dot")}
                  value={settings.artNotify}
                  onChange={(value) => set("artNotify", value)}
                  options={[
                    ["on", t("Show")],
                    ["off", t("Hide")],
                  ]}
                />
              </Row>
            </>
          )}
          {section === "agent" && (
            <>
              <Row
                title={t("Approval celebration")}
                detail={t("What plays when you approve a proposed change.")}
              >
                <Seg
                  label={t("Approval celebration")}
                  value={settings.celebrate}
                  onChange={(value) => set("celebrate", value)}
                  options={[
                    ["confetti", t("Confetti")],
                    ["subtle", t("Subtle")],
                    ["off", t("Off")],
                  ]}
                />
              </Row>
              <Row
                title={t("Working border")}
                detail={t("The light that travels around the composer while an agent works.")}
              >
                <Seg
                  label={t("Working border")}
                  value={settings.ring}
                  onChange={(value) => set("ring", value)}
                  options={[
                    ["on", t("Animated")],
                    ["off", t("Off")],
                  ]}
                />
              </Row>
            </>
          )}
        </div>
        )}
      </div>
    </div>
  );
}

/** The classes on the Desk's root that carry a person's settings into its styles. */
export function deskSettingsClasses(settings: DeskSettings): string {
  return cn(
    `dk-w-${settings.width}`,
    `dk-t-${settings.text}`,
    (settings.ring === "off" || settings.motion === "reduced") && "dk-no-ring",
    settings.motion === "reduced" && "dk-calm",
    settings.drop === "composer" && "dk-drop-cmp",
    settings.refs === "off" && "dk-no-refs",
    settings.celebrate !== "confetti" && "dk-calm-ok",
    settings.celebrate === "off" && "dk-no-ok-fx",
  );
}
