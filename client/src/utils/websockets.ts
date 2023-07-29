/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
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
  message:
    | string
    | Array<
        Array<{
          invoice_number: string;
          missing_documents: Array<string>;
        }>
      >;
};

/**
 * Represents a WebSocket connection, providing lifecycle event handling.
 */
export class WebSocketConnection {
  private socket: WebSocket;

  constructor(url: string, private eventHandlers: Handlers) {
    this.socket = new WebSocket(url);

    this.socket.onopen = this.handleOpen.bind(this);
    this.socket.onmessage = this.handleMessage.bind(this);
    this.socket.onclose = this.handleClose.bind(this);
    this.socket.onerror = this.handleError.bind(this);
  }

  private handleOpen(event: Event) {
    this.eventHandlers.onOpen?.(event);
  }

  private handleMessage(event: MessageEvent) {
    this.eventHandlers.onMessage?.(event);
  }

  private handleClose(event: CloseEvent) {
    this.eventHandlers.onClose?.(event);
  }

  private handleError(event: Event) {
    this.eventHandlers.onError?.(event);
  }

  public send(data: any) {
    this.socket.send(data);
  }

  public receive(handler: (event: WebSocketEvent) => void) {
    this.socket.onmessage = handler;
  }

  public close() {
    this.socket.close();
  }
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

export interface WebSocketManager {
  connect: (id: string, url: string, handlers: Handlers) => WebSocketConnection;
  disconnect: (id: string) => void;
  get: (id: string) => WebSocketConnection;
  send: (id: string, data: any) => void;
  sendJson: (id: string, data: any) => void;
  receive: (id: string, handler: (event: WebSocketEvent) => void) => void;
  has: (id: string) => boolean;
}

export function createWebsocketManager(): WebSocketManager {
  const connections = new Map<string, WebSocketConnection>();

  function connect(id: string, url: string, handlers: Handlers) {
    if (connections.has(id)) {
      throw new Error(`WebSocket connection with id "${id}" already exists`);
    }

    const connection = new WebSocketConnection(url, handlers);
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

    return connection;
  }

  function send(id: string, data: any) {
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

  function receive(id: string, handler: (event: WebSocketEvent) => void) {
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
    send,
    sendJson,
    receive,
    get,
    has,
  };
}
