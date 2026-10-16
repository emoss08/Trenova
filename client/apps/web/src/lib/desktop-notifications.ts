type DeliveryDecision = { urgent: boolean; desktop: boolean; sound: boolean };

type IncomingNotification = {
  id?: string;
  title?: string;
  message?: string;
  data?: Record<string, unknown> | null;
};

/**
 * What the server decided for this person's desktop and sound, stamped on the notification
 * when it was created. A notification without one predates the preferences and is treated
 * as bell-only, the same as the server would.
 */
export function deliveryDecision(notification: IncomingNotification): DeliveryDecision {
  const delivery = notification.data?.delivery;
  if (!delivery || typeof delivery !== "object") {
    return { urgent: false, desktop: false, sound: false };
  }
  const record = delivery as Record<string, unknown>;
  return {
    urgent: record.urgent === true,
    desktop: record.desktop === true,
    sound: record.sound === true,
  };
}

/**
 * A desktop notification is shown only when the server allowed it, the browser was
 * given permission, and the person is not already looking at Trenova: a visible tab has
 * the notification bell, and a toast for what `shouldToastNotification` lets through.
 */
export function shouldShowDesktop(
  notification: IncomingNotification,
  environment: { permission: NotificationPermission | "unsupported"; visible: boolean },
): boolean {
  return (
    deliveryDecision(notification).desktop &&
    environment.permission === "granted" &&
    !environment.visible &&
    Boolean(notification.title)
  );
}

let audioContext: AudioContext | null = null;

function playChime() {
  if (typeof window === "undefined" || !("AudioContext" in window)) return;
  audioContext ??= new AudioContext();
  const context = audioContext;
  const start = context.currentTime;
  const gain = context.createGain();
  gain.gain.setValueAtTime(0.0001, start);
  gain.gain.exponentialRampToValueAtTime(0.08, start + 0.02);
  gain.gain.exponentialRampToValueAtTime(0.0001, start + 0.6);
  gain.connect(context.destination);
  for (const [frequency, offset] of [
    [880, 0],
    [1320, 0.09],
  ] as const) {
    const oscillator = context.createOscillator();
    oscillator.type = "sine";
    oscillator.frequency.setValueAtTime(frequency, start + offset);
    oscillator.connect(gain);
    oscillator.start(start + offset);
    oscillator.stop(start + 0.6);
  }
}

export function deliverToDesktop(notification: IncomingNotification, onClick?: () => void) {
  const permission =
    typeof window !== "undefined" && "Notification" in window
      ? window.Notification.permission
      : "unsupported";
  const visible = typeof document !== "undefined" && document.visibilityState === "visible";
  if (!shouldShowDesktop(notification, { permission, visible })) {
    return;
  }

  const shown = new window.Notification(notification.title ?? "", {
    body: notification.message ?? "",
    tag: notification.id,
  });
  shown.onclick = () => {
    window.focus();
    onClick?.();
    shown.close();
  };
  if (deliveryDecision(notification).sound) {
    playChime();
  }
}
