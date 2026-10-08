import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatUnixTime } from "@trenova/shared/lib/date";
import { cn, getNameInitials } from "@trenova/shared/lib/utils";
import { useReducedMotion } from "motion/react";
import { useEffect, useState, type CSSProperties } from "react";
import { Ic } from "../kit/ic";

/** One step of the loop, in 140ms beats. */
const BEAT_MS = 140;
/** The beat the answer starts on, and the beat by which everything has been shown. */
const ANSWER_BEAT = 22;
const LAST_BEAT = 48;

type ExtensionDemoProps = {
  /** The agent the demo shows answering: a real Desk agent of the organization's. */
  agentName: string;
};

/**
 * How an agent uses web search in Desk: a person asks, the agent searches and reads two
 * pages, then answers with its sources. It plays through once and holds the answer, since
 * nothing loops on a working screen; under reduced motion it shows the answer at once.
 */
export function ExtensionDemo({ agentName }: ExtensionDemoProps) {
  const t = useT();
  const reduced = useReducedMotion();
  const user = useAuthStore((state) => state.user);
  const [beat, setBeat] = useState(0);
  const [askedAt] = useState(() => Math.floor(Date.now() / 1000));

  useEffect(() => {
    if (reduced) return;
    if (beat >= LAST_BEAT) return;
    const timer = window.setTimeout(() => setBeat((current) => current + 1), BEAT_MS);
    return () => window.clearTimeout(timer);
  }, [beat, reduced]);

  const answer = t(
    "Yes. Their liability policy with Great West Casualty runs to Mar 31, 2027, and FMCSA shows the authority as active.",
  ).split(" ");
  const steps = [
    {
      label: t("Searching the web"),
      done: t("Searched the web"),
      detail: t("Ridgeline Freight MC 884213 insurance"),
      elapsed: "0.8s",
    },
    {
      label: t("Reading 2 pages"),
      done: t("Read 2 pages"),
      detail: "safer.fmcsa.dot.gov · ridgelinefreight.com",
      elapsed: "1.3s",
    },
  ];

  const at = reduced ? LAST_BEAT : beat;
  const asked = at >= 1;
  const working = at >= 5;
  const shown = [at >= 8, at >= 15];
  const answered = at >= ANSWER_BEAT;
  const words = answered ? Math.min(answer.length, at - ANSWER_BEAT) : 0;
  const sourced = at >= ANSWER_BEAT + answer.length;
  const name = user?.name ?? t("You");

  return (
    <div className="demo" aria-label={t("How agents use it")}>
      <span className="demo-k">
        <Ic n="chat" s={11} />
        {t("How agents use it · Desk")}
      </span>
      <div className={cn("dq", asked && "in")}>
        <span className="dq-w">
          <span className="me sm">{getNameInitials(name, "U", { maxLength: 2 })}</span>
          <b>{name}</b>
          <span className="mono">{formatUnixTime(askedAt)}</span>
        </span>
        <p>{t("Is Ridgeline Freight's insurance current?")}</p>
      </div>
      <div className={cn("da", working && "in")}>
        <span className="who">
          <span className={cn("dm", working && !answered && "busy")} />
          <b>{agentName}</b>
          <span className="dtg">
            <Ic n="search" s={10} />
            {t("Web search")}
          </span>
        </span>
        <div className="dn">
          {steps.map((step, index) => {
            const on = shown[index];
            const past = index === 0 ? shown[1] : answered;
            return (
              <div
                key={`${index}-${on ? "on" : "off"}`}
                className={cn("dnl", !on ? "hid" : past ? "past" : "cur")}
              >
                <span className="ck">
                  {past ? <Ic n="check" s={12} w={2.4} /> : <i className="spn" />}
                </span>
                <span className="tx">
                  {past ? step.done : step.label}
                  <em>{step.detail}</em>
                </span>
                <span className="el mono">{past ? step.elapsed : ""}</span>
              </div>
            );
          })}
        </div>
        <p className="dp">
          {answer.slice(0, words).map((word, index) => (
            <span key={index} className="dw">
              {word}{" "}
            </span>
          ))}
          {answered &&
            (words < answer.length ? (
              <span className="car" />
            ) : (
              <span className="wc w">
                <SourceMark letter="F" hue={230} size={12} />
                fmcsa.dot.gov
                <em>+1</em>
              </span>
            ))}
        </p>
        <div className={cn("dsx", sourced && "in")}>
          <span className="ws-b">
            <span className="ws-st">
              <SourceMark letter="F" hue={230} size={16} />
              <SourceMark letter="R" hue={25} size={16} />
            </span>
            {t("2 sources")}
            <Ic n="chevR" s={11} />
          </span>
        </div>
      </div>
    </div>
  );
}

function SourceMark({ letter, hue, size }: { letter: string; hue: number; size: number }) {
  return (
    <span
      className="wf"
      style={
        { "--wh": hue, width: size, height: size, fontSize: size * 0.58 } as CSSProperties
      }
    >
      {letter}
    </span>
  );
}
