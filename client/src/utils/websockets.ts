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
  /**
   * The underlying WebSocket object.
   */
  socket: WebSocket;

  /**
   * Creates a new WebSocketConnection instance.
   * @param url - The URL to which the WebSocket should connect.
   * @param eventHandlers - Event handlers for the WebSocket's lifecycle events.
   */
  constructor(url: string, private eventHandlers: Handlers) {
    this.socket = new WebSocket(url);

    this.socket.onopen = this.handleOpen.bind(this);
    this.socket.onmessage = this.handleMessage.bind(this);
    this.socket.onclose = this.handleClose.bind(this);
    this.socket.onerror = this.handleError.bind(this);
  }

  /**
   * Handles the WebSocket's open event.
   * @param event - The open event.
   */
  handleOpen(event: Event) {
    this.eventHandlers.onOpen && this.eventHandlers.onOpen(event);
  }

  /**
   * Handles the WebSocket's message event.
   * @param event - The message event.
   */
  handleMessage(event: MessageEvent) {
    this.eventHandlers.onMessage && this.eventHandlers.onMessage(event);
  }

  /**
   * Handles the WebSocket's close event.
   * @param event - The close event.
   */
  handleClose(event: CloseEvent) {
    this.eventHandlers.onClose && this.eventHandlers.onClose(event);
  }

  /**
   * Handles the WebSocket's error event.
   * @param event - The error event.
   */
  handleError(event: Event) {
    this.eventHandlers.onError && this.eventHandlers.onError(event);
  }

  /**
   * Closes the WebSocket connection.
   */
  close() {
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

interface WebSocketManager {
  connect: (id: string, url: string, handlers: Handlers) => WebSocketConnection;
  disconnect: (id: string) => void;
  disconnectAll: () => void;
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

    // return connection;
  }

  function disconnectAll() {
    for (const id of connections.keys()) {
      disconnect(id);
    }
  }

  function send(id: string, data: any) {
    const connection = connections.get(id);
    if (!connection) {
      throw new Error(`No connection with id ${id} found`);
    }

    connection.socket.send(data);
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

    connection.socket.onmessage = handler;
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
