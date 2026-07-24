import { api } from "@trenova/shared/lib/api";

const SERVICE_WORKER_URL = `${import.meta.env.BASE_URL}dash-sw.js`;

type PushPublicKeyResponse = {
  enabled: boolean;
  publicKey: string;
};

export function pushSupported(): boolean {
  return (
    typeof window !== "undefined" &&
    "serviceWorker" in navigator &&
    "PushManager" in window &&
    "Notification" in window
  );
}

function urlBase64ToUint8Array(base64String: string): Uint8Array {
  const padding = "=".repeat((4 - (base64String.length % 4)) % 4);
  const base64 = (base64String + padding).replace(/-/g, "+").replace(/_/g, "/");
  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);
  for (let i = 0; i < rawData.length; i += 1) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

export async function fetchPushConfig(): Promise<PushPublicKeyResponse> {
  return api.get<PushPublicKeyResponse>("/push/public-key/");
}

export async function getPushSubscription(): Promise<PushSubscription | null> {
  if (!pushSupported()) {
    return null;
  }
  const registration = await navigator.serviceWorker.getRegistration(SERVICE_WORKER_URL);
  if (!registration) {
    return null;
  }
  return registration.pushManager.getSubscription();
}

function subscriptionKeys(subscription: PushSubscription) {
  const json = subscription.toJSON();
  return {
    endpoint: subscription.endpoint,
    p256dh: json.keys?.p256dh ?? "",
    auth: json.keys?.auth ?? "",
  };
}

export async function enablePush(): Promise<void> {
  if (!pushSupported()) {
    throw new Error("This browser doesn't support push notifications.");
  }

  const config = await fetchPushConfig();
  if (!config.enabled || !config.publicKey) {
    throw new Error("Push notifications aren't configured for this carrier yet.");
  }

  const permission = await Notification.requestPermission();
  if (permission !== "granted") {
    throw new Error("Notifications were blocked. Enable them in your browser settings.");
  }

  const registration = await navigator.serviceWorker.register(SERVICE_WORKER_URL);
  await navigator.serviceWorker.ready;

  const subscription =
    (await registration.pushManager.getSubscription()) ??
    (await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(config.publicKey).buffer as ArrayBuffer,
    }));

  await api.post("/push/subscriptions/", subscriptionKeys(subscription));
}

export async function disablePush(): Promise<void> {
  const subscription = await getPushSubscription();
  if (!subscription) {
    return;
  }
  try {
    await api.delete("/push/subscriptions/", {
      body: JSON.stringify({ endpoint: subscription.endpoint }),
    });
  } finally {
    await subscription.unsubscribe();
  }
}
