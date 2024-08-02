/**
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

export type TWebsocketStatuses =
  | "SUCCESS"
  | "FAILURE"
  | "WARNING"
  | "PROCESSING"
  | "INFO";

export type WebSocketMessageProps = {
  action?: string | null;
  step?: number | null;
  status?: TWebsocketStatuses | null;
  message: string;
};

interface WebSocketOptions {
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
}

interface Handlers {
  onOpen?: (event: Event) => void;
  onMessage?: (event: MessageEvent) => void;
  onError?: (event: Event) => void;
  onClose?: (event: CloseEvent) => void;
}

interface WebSocketEventMap {
  close: CloseEvent;
  error: Event;
  message: MessageEvent;
  open: Event;
}

export type WebSocketEvent = WebSocketEventMap[keyof WebSocketEventMap];

/**
 * Represents a WebSocket connection, providing lifecycle event handling.
 */
export class WebSocketConnection {
  private socket: WebSocket;
  private readonly url: string;
  private readonly reconnectInterval: number;
  private reconnectAttempts: number = 0;
  private readonly maxReconnectAttempts: number;

  constructor(
    url: string,
    private eventHandlers: Handlers,
    options: WebSocketOptions = {},
  ) {
    this.url = url;
    this.reconnectInterval = options.reconnectInterval || 5000;
    this.maxReconnectAttempts = options.maxReconnectAttempts || 5;
    this.socket = this.createWebSocket();
  }

  private createWebSocket(): WebSocket {
    const socket = new WebSocket(this.url);
    socket.onopen = this.handleOpen.bind(this);
    socket.onmessage = this.handleMessage.bind(this);
    socket.onclose = this.handleClose.bind(this);
    socket.onerror = this.handleError.bind(this);
    return socket;
  }

  private handleOpen(event: Event) {
    this.reconnectAttempts = 0; // Reset reconnect attempts
    this.eventHandlers.onOpen?.(event);
  }

  private handleMessage(event: MessageEvent) {
    this.eventHandlers.onMessage?.(event);
  }

  private handleClose(event: CloseEvent) {
    if (this.reconnectAttempts < this.maxReconnectAttempts) {
      setTimeout(() => {
        this.reconnectAttempts++;
        this.socket = this.createWebSocket();
      }, this.reconnectInterval);
    }
    this.eventHandlers.onClose?.(event);
  }

  private handleError(event: Event) {
    console.error("WebSocket error:", event);
    this.eventHandlers.onError?.(event);
  }

  public send<T>(data: T) {
    this.socket.send(JSON.stringify(data));
  }

  public receive<T>(handler: (data: T) => void) {
    this.socket.onmessage = (event: MessageEvent) => {
      const data = JSON.parse(event.data) as T;
      handler(data);
    };
  }

  public close() {
    this.socket.close();
  }

  public isConnected(): boolean {
    return this.socket.readyState === WebSocket.OPEN;
  }

  public isConnecting(): boolean {
    return this.socket.readyState === WebSocket.CONNECTING;
  }

  public isClosed(): boolean {
    return this.socket.readyState === WebSocket.CLOSED;
  }
}

export interface WebSocketManager {
  connect: (
    id: string,
    url: string,
    handlers: Handlers,
    options?: WebSocketOptions,
  ) => WebSocketConnection | boolean;
  disconnect: (id: string) => void;
  disconnectAll: () => void;
  get: (id: string) => WebSocketConnection;
  send: <T>(id: string, data: T) => void;
  sendJson: (id: string, data: any) => void;
  receive: <T>(id: string, handler: (data: T) => void) => void;
  has: (id: string) => boolean;
}

export function createWebsocketManager(): WebSocketManager {
  const connections = new Map<string, WebSocketConnection>();

  function connect(
    id: string,
    url: string,
    handlers: Handlers,
    options: WebSocketOptions = {},
  ): WebSocketConnection | boolean {
    if (connections.has(id)) {
      return connections.get(id) as WebSocketConnection;
    }

    const connection = new WebSocketConnection(url, handlers, options);
    connections.set(id, connection);

    return connection;
  }

  function disconnect(id: string) {
    const connection = connections.get(id);
    if (!connection) {
      throw new Error(`WebSocket connection with id "${id}" not found`);
    }

    connection.close();
    connections.delete(id);
  }

  function disconnectAll() {
    for (const id of connections.keys()) {
      disconnect(id);
    }
  }

  function send<T>(id: string, data: T) {
    const connection = connections.get(id);
    if (!connection) {
      throw new Error(`No connection with id ${id} found`);
    }

    connection.send(data);
  }

  function sendJson(id: string, data: any) {
    send(id, JSON.stringify(data));
  }

  function get(id: string) {
    const connection = connections.get(id);
    if (!connection) {
      throw new Error(`WebSocket connection with id "${id}" not found`);
    }

    return connection;
  }

  function receive<T>(id: string, handler: (data: T) => void) {
    const connection = connections.get(id);
    if (!connection) {
      throw new Error(`No connection with id ${id} found`);
    }

    connection.receive(handler);
  }

  function has(id: string) {
    return connections.has(id);
  }

  return {
    connect,
    disconnect,
    disconnectAll,
    send,
    sendJson,
    receive,
    get,
    has,
  };
}
