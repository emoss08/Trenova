import { api } from "@trenova/shared/lib/api";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
import { readEventStream, type SSEMessage } from "@trenova/shared/lib/sse";

export type RealtimeConnectionState = "connecting" | "connected" | "disconnected";

export const REALTIME_USERS_SCOPE = "users";

export interface RealtimePresenceMember {
  userId: string;
  connectionId: string;
  name: string;
}

export interface RealtimeTypingEvent {
  scope: string;
  userId: string;
  name?: string;
  stop?: boolean;
}

export interface RealtimeConnectOptions {
  /** Who the stream is for. A change (sign-in as someone else, a tenant switch) reopens it. */
  identity: string;
  /** Join the tenant's online-user presence. The TMS does; the driver portal does not. */
  joinUsers: boolean;
}

type Listener<T> = (payload: T) => void;

interface ReadyEvent {
  connectionId: string;
  heartbeatIntervalMs: number;
  resumed: boolean;
  cursor?: string;
}

interface PresenceEvent {
  scope: string;
  action: "enter" | "leave";
  userId: string;
  connectionId: string;
  name?: string;
}

interface PresenceSnapshot {
  scope: string;
  members: RealtimePresenceMember[];
}

interface ScopeState {
  members: Map<string, RealtimePresenceMember>;
  listeners: Set<Listener<RealtimePresenceMember[]>>;
  /** How many callers hold a join on this scope; zero means watched but not joined. */
  joins: number;
  joinPath: string | null;
}

const STREAM_PATH = "/realtime/stream/";
const DEFAULT_HEARTBEAT_MS = 15_000;
/** A stream silent for this many heartbeats is treated as dead, even if the socket says otherwise. */
const WATCHDOG_HEARTBEATS = 2.5;
const BACKOFF_BASE_MS = 1_000;
const BACKOFF_MAX_MS = 30_000;
const MIN_RECONNECT_MS = 250;
/** A server that is recycling or draining streams asks everyone back; spread them out. */
const ROTATE_SPREAD_MS = 1_000;
const SHUTDOWN_SPREAD_MS = 5_000;

function jitter(maxMs: number): number {
  return Math.round(maxMs / 2 + Math.random() * (maxMs / 2));
}

function parseJSON(data: string): unknown {
  try {
    return JSON.parse(data) as unknown;
  } catch {
    return null;
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

/**
 * The browser's one live connection to the server: a server-sent event stream
 * carrying record invalidations, presence and typing for the signed-in user's
 * tenant.
 *
 * The stream is read with fetch rather than EventSource so reconnection is
 * ours: a resume cursor sent back as Last-Event-ID, backoff with jitter, a
 * watchdog on the server's heartbeat, and no retry at all once the session is
 * gone. Presence is tied to the stream's connection on the server, so closing
 * the tab withdraws it without a request of its own.
 */
export class RealtimeClient {
  private options: RealtimeConnectOptions | null = null;
  private state: RealtimeConnectionState = "disconnected";
  private controller: AbortController | null = null;
  private running = false;
  private generation = 0;
  private attempt = 0;
  private connectionId: string | null = null;
  private lastEventId: string | null = null;
  private closeReason: string | null = null;
  private heartbeatMs = DEFAULT_HEARTBEAT_MS;
  private watchdog: ReturnType<typeof setTimeout> | null = null;
  private retryTimer: ReturnType<typeof setTimeout> | null = null;
  private wakeRetry: (() => void) | null = null;

  private readonly stateListeners = new Set<Listener<RealtimeConnectionState>>();
  private readonly eventListeners = new Map<string, Set<Listener<unknown>>>();
  private readonly scopes = new Map<string, ScopeState>();

  constructor() {
    if (typeof window !== "undefined") {
      window.addEventListener("online", () => this.retryNow());
    }
  }

  /** Opens the stream, or keeps the open one when nothing about it has changed. */
  public connect(options: RealtimeConnectOptions): void {
    const current = this.options;
    const changed =
      current === null ||
      current.identity !== options.identity ||
      current.joinUsers !== options.joinUsers;

    if (!changed && this.running) {
      return;
    }

    if (changed && current !== null) {
      this.stop();
      this.resetSession();
    }

    this.options = options;
    this.running = true;
    const generation = ++this.generation;
    void this.run(generation);
  }

  /** Closes the stream and forgets the session: its cursor, its presence and its scopes. */
  public disconnect(): void {
    this.options = null;
    this.stop();
    this.resetSession();
  }

  public getState(): RealtimeConnectionState {
    return this.state;
  }

  public getConnectionId(): string | null {
    return this.connectionId;
  }

  public onStateChange(listener: Listener<RealtimeConnectionState>): () => void {
    this.stateListeners.add(listener);
    return () => this.stateListeners.delete(listener);
  }

  /** Listens for one event by name: "resource.invalidation", "typing", or "reset". */
  public on<T = unknown>(event: string, listener: Listener<T>): () => void {
    let listeners = this.eventListeners.get(event);
    if (!listeners) {
      listeners = new Set();
      this.eventListeners.set(event, listeners);
    }
    const wrapped = listener as Listener<unknown>;
    listeners.add(wrapped);
    return () => {
      listeners.delete(wrapped);
    };
  }

  /**
   * Follows who is present in a scope. The listener is called with the full
   * member list on every change, including right away when the list is known.
   */
  public subscribePresence(
    scope: string,
    listener: Listener<RealtimePresenceMember[]>,
  ): () => void {
    const state = this.scopeState(scope);
    state.listeners.add(listener);
    if (state.members.size > 0) {
      listener(Array.from(state.members.values()));
    }
    return () => {
      state.listeners.delete(listener);
      this.pruneScope(scope);
    };
  }

  /**
   * Joins a presence scope through the endpoint that authorizes it. Joins are
   * counted, so two components on one scope share one membership, and they are
   * replayed on every reconnect because a new stream is a new connection.
   */
  public joinScope(scope: string, joinPath: string): () => void {
    const state = this.scopeState(scope);
    state.joins += 1;
    state.joinPath = joinPath;
    if (state.joins === 1 && this.connectionId) {
      void this.sendJoin(scope, joinPath, this.connectionId);
    }

    let released = false;
    return () => {
      if (released) return;
      released = true;
      state.joins -= 1;
      if (state.joins > 0) return;

      const connectionId = this.connectionId;
      const path = state.joinPath;
      state.joinPath = null;
      state.members.clear();
      this.notifyScope(scope);
      this.pruneScope(scope);
      if (connectionId && path) {
        void api
          .delete(`${path}?connectionId=${encodeURIComponent(connectionId)}`)
          .catch(() => undefined);
      }
    };
  }

  /** Tells a joined scope the user is typing, or has stopped. Best effort by design. */
  public sendTyping(typingPath: string, stop: boolean): void {
    const connectionId = this.connectionId;
    if (!connectionId || this.state !== "connected") return;
    void api.post(typingPath, { connectionId, stop }).catch(() => undefined);
  }

  private async run(generation: number): Promise<void> {
    while (this.running && generation === this.generation) {
      const delay = await this.openOnce(generation);
      if (!this.running || generation !== this.generation) {
        return;
      }
      if (delay === null) {
        this.running = false;
        this.setState("disconnected");
        return;
      }
      this.setState("connecting");
      await this.wait(delay);
    }
  }

  /**
   * Holds one stream open until it ends. Returns how long to wait before the
   * next attempt, or null when there should not be one.
   */
  private async openOnce(generation: number): Promise<number | null> {
    const options = this.options;
    if (!options) return null;

    this.setState("connecting");
    this.closeReason = null;
    const controller = new AbortController();
    this.controller = controller;

    const headers = new Headers({ Accept: "text/event-stream" });
    if (this.lastEventId) {
      headers.set("Last-Event-ID", this.lastEventId);
    }
    const query = options.joinUsers ? `?presence=${REALTIME_USERS_SCOPE}` : "";

    let response: Response;
    try {
      response = await fetch(`${API_BASE_URL}${STREAM_PATH}${query}`, {
        credentials: "include",
        cache: "no-store",
        headers,
        signal: controller.signal,
      });
    } catch {
      return this.backoff();
    }

    if (response.status === 401 || response.status === 403) {
      return null;
    }
    if (response.status === 429 || response.status === 503) {
      const retryAfter = Number(response.headers.get("Retry-After"));
      const base = Number.isFinite(retryAfter) && retryAfter > 0 ? retryAfter * 1_000 : undefined;
      return base === undefined ? this.backoff() : base + jitter(BACKOFF_BASE_MS);
    }
    if (!response.ok || !response.body) {
      return this.backoff();
    }

    this.armWatchdog(controller);
    try {
      await readEventStream(
        response.body,
        (message) => {
          if (generation === this.generation) {
            this.handle(message, controller);
          }
        },
        controller.signal,
      );
    } catch {
      // The connection dropped mid-stream; the reason below decides the wait.
    } finally {
      this.clearWatchdog();
      this.connectionId = null;
      if (this.controller === controller) {
        this.controller = null;
      }
    }

    switch (this.lastCloseReason()) {
      case "rotate":
        return jitter(ROTATE_SPREAD_MS);
      case "shutdown":
        return jitter(SHUTDOWN_SPREAD_MS);
      case "overflow":
        return MIN_RECONNECT_MS;
      default:
        return this.backoff();
    }
  }

  private handle(message: SSEMessage, controller: AbortController): void {
    switch (message.event) {
      case "ready":
        this.onReady(parseJSON(message.data));
        break;
      case "heartbeat":
        break;
      case "close": {
        const payload = parseJSON(message.data);
        this.closeReason =
          isRecord(payload) && typeof payload.reason === "string" ? payload.reason : null;
        break;
      }
      case "presence.snapshot":
        this.onSnapshot(parseJSON(message.data));
        break;
      case "presence":
        this.onPresence(parseJSON(message.data));
        break;
      case "reset":
        this.emit("reset", null);
        break;
      default: {
        const payload = parseJSON(message.data);
        if (payload !== null) {
          this.emit(message.event, payload);
        }
      }
    }

    // The cursor moves only after the event is applied, so a reconnect resumes
    // from the last event this tab acted on rather than the last one received.
    if (message.id) {
      this.lastEventId = message.id;
    }

    // Re-armed after the event is applied, so the window a ready event
    // announces governs the very next silence.
    if (this.controller === controller) {
      this.armWatchdog(controller);
    }
  }

  /** Why the server ended the last stream, read after the stream has ended. */
  private lastCloseReason(): string | null {
    return this.closeReason;
  }

  private onReady(payload: unknown): void {
    if (!isRecord(payload) || typeof payload.connectionId !== "string") {
      return;
    }
    const ready = payload as unknown as ReadyEvent;
    this.connectionId = ready.connectionId;
    this.heartbeatMs =
      typeof ready.heartbeatIntervalMs === "number" && ready.heartbeatIntervalMs > 0
        ? ready.heartbeatIntervalMs
        : DEFAULT_HEARTBEAT_MS;
    if (!ready.resumed && ready.cursor) {
      this.lastEventId = ready.cursor;
    }
    this.attempt = 0;
    this.setState("connected");

    for (const [scope, state] of this.scopes) {
      if (state.joins > 0 && state.joinPath) {
        void this.sendJoin(scope, state.joinPath, ready.connectionId);
      }
    }
  }

  private onSnapshot(payload: unknown): void {
    if (!isRecord(payload) || typeof payload.scope !== "string") return;
    const snapshot = payload as unknown as PresenceSnapshot;
    // The tenant's user presence arrives with the stream, often before anything
    // on the page has asked for it, so its state is kept whether or not anyone
    // is listening yet.
    this.scopeState(snapshot.scope);
    this.replaceMembers(snapshot.scope, snapshot.members ?? []);
  }

  private onPresence(payload: unknown): void {
    if (!isRecord(payload) || typeof payload.scope !== "string") return;
    const event = payload as unknown as PresenceEvent;
    const state = this.scopes.get(event.scope);
    if (!state || !event.connectionId || !event.userId) return;

    if (event.action === "leave") {
      if (!state.members.delete(event.connectionId)) return;
    } else {
      state.members.set(event.connectionId, {
        userId: event.userId,
        connectionId: event.connectionId,
        name: event.name ?? "",
      });
    }
    this.notifyScope(event.scope);
  }

  private async sendJoin(scope: string, path: string, connectionId: string): Promise<void> {
    try {
      const snapshot = await api.post<PresenceSnapshot>(path, { connectionId });
      const state = this.scopes.get(scope);
      if (!state || state.joins === 0 || this.connectionId !== connectionId) {
        return;
      }
      this.replaceMembers(scope, snapshot.members ?? []);
    } catch {
      // A join that fails leaves the scope empty until the next reconnect;
      // presence is a courtesy, never a gate on the page working.
    }
  }

  private replaceMembers(scope: string, members: RealtimePresenceMember[]): void {
    const state = this.scopes.get(scope);
    if (!state) return;
    state.members = new Map(members.map((member) => [member.connectionId, member]));
    this.notifyScope(scope);
  }

  private notifyScope(scope: string): void {
    const state = this.scopes.get(scope);
    if (!state) return;
    const members = Array.from(state.members.values());
    for (const listener of state.listeners) {
      listener(members);
    }
  }

  private scopeState(scope: string): ScopeState {
    let state = this.scopes.get(scope);
    if (!state) {
      state = { members: new Map(), listeners: new Set(), joins: 0, joinPath: null };
      this.scopes.set(scope, state);
    }
    return state;
  }

  private pruneScope(scope: string): void {
    if (scope === REALTIME_USERS_SCOPE) return;
    const state = this.scopes.get(scope);
    if (state && state.joins === 0 && state.listeners.size === 0) {
      this.scopes.delete(scope);
    }
  }

  private emit(event: string, payload: unknown): void {
    const listeners = this.eventListeners.get(event);
    if (!listeners) return;
    for (const listener of listeners) {
      listener(payload);
    }
  }

  private setState(state: RealtimeConnectionState): void {
    if (this.state === state) return;
    this.state = state;
    for (const listener of this.stateListeners) {
      listener(state);
    }
  }

  private backoff(): number {
    const ceiling = Math.min(BACKOFF_MAX_MS, BACKOFF_BASE_MS * 2 ** this.attempt);
    this.attempt += 1;
    return Math.max(MIN_RECONNECT_MS, Math.round(Math.random() * ceiling));
  }

  private armWatchdog(controller: AbortController): void {
    this.clearWatchdog();
    this.watchdog = setTimeout(() => controller.abort(), this.heartbeatMs * WATCHDOG_HEARTBEATS);
  }

  private clearWatchdog(): void {
    if (this.watchdog !== null) {
      clearTimeout(this.watchdog);
      this.watchdog = null;
    }
  }

  private wait(ms: number): Promise<void> {
    return new Promise((resolve) => {
      this.wakeRetry = () => {
        if (this.retryTimer !== null) {
          clearTimeout(this.retryTimer);
          this.retryTimer = null;
        }
        this.wakeRetry = null;
        resolve();
      };
      this.retryTimer = setTimeout(() => this.wakeRetry?.(), ms);
    });
  }

  private retryNow(): void {
    if (this.running && this.wakeRetry) {
      this.attempt = 0;
      this.wakeRetry();
    }
  }

  private stop(): void {
    this.running = false;
    this.generation += 1;
    this.controller?.abort();
    this.controller = null;
    this.clearWatchdog();
    this.wakeRetry?.();
    this.connectionId = null;
    this.setState("disconnected");
  }

  private resetSession(): void {
    this.lastEventId = null;
    this.attempt = 0;
    for (const scope of this.scopes.values()) {
      scope.members.clear();
    }
    for (const scope of this.scopes.keys()) {
      this.notifyScope(scope);
    }
  }
}

export const realtimeClient = new RealtimeClient();
