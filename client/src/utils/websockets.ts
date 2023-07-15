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

type WebSocketEventHandlers = {
  onOpen?: (event: Event) => void;
  onMessage?: (event: MessageEvent) => void;
  onClose?: (event: CloseEvent) => void;
  onError?: (event: Event) => void;
};

export type TWebsocketStatuses =
  | "SUCCESS"
  | "FAILURE"
  | "WARNING"
  | "PROCESSING"
  | "INFO";

export type WebsocketMessageProps = {
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
class WebSocketConnection {
  /**
   * The underlying WebSocket object.
   */
  socket: WebSocket;

  /**
   * Creates a new WebSocketConnection instance.
   * @param url - The URL to which the WebSocket should connect.
   * @param eventHandlers - Event handlers for the WebSocket's lifecycle events.
   */
  constructor(url: string, private eventHandlers: WebSocketEventHandlers = {}) {
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

/**
 * Manages WebSocket connections by providing functionality
 * to connect, disconnect, send messages, and check the existence of connections.
 */
export class WebSocketManager {
  /**
   * A map to keep track of WebSocket connections.
   * @private
   */
  private connections = new Map<string, WebSocketConnection>();

  /**
   * Establishes a WebSocket connection with the specified ID.
   * @param id - Unique identifier for the WebSocket connection.
   * @param url - URL to which the WebSocket connection should be established.
   * @param eventHandlers - Optional event handlers for the WebSocket connection.
   * @throws Will throw an error if a connection with the same ID already exists.
   */
  connect(id: string, url: string, eventHandlers: WebSocketEventHandlers = {}) {
    if (this.connections.has(id)) {
      throw new Error(`WebSocket connection with id "${id}" already exists`);
    }

    const connection = new WebSocketConnection(url, eventHandlers);
    this.connections.set(id, connection);
  }

  /**
   * Closes a WebSocket connection with the specified ID.
   * @param id - The ID of the WebSocket connection to close.
   * @throws Will throw an error if no connection with the specified ID is found.
   */
  disconnect(id: string) {
    const connection = this.connections.get(id);
    if (!connection) {
      throw new Error(`WebSocket connection with id "${id}" not found`);
    }

    connection.close();
    this.connections.delete(id);
  }

  /**
   * Closes all WebSocket connections managed by the instance.
   */
  disconnectAll() {
    for (const id of this.connections.keys()) {
      this.disconnect(id);
    }
  }

  /**
   * Sends a JSON message to the WebSocket connection with the specified ID.
   * @param id - The ID of the WebSocket connection to which the message should be sent.
   * @param data - The data to be sent.
   * @throws Will throw an error if no connection with the specified ID is found.
   */
  sendJsonMessage(id: string, data: any) {
    const connection = this.connections.get(id);
    if (!connection) {
      throw new Error(`WebSocket connection with id "${id}" not found`);
    }

    connection.socket.send(JSON.stringify(data));
  }

  /**
   * Sends a message to the WebSocket connection with the specified ID.
   * @param id - The ID of the WebSocket connection to which the message should be sent.
   * @param data - The data to be sent.
   * @throws Will throw an error if no connection with the specified ID is found.
   */
  sendMessage(id: string, data: any) {
    const connection = this.connections.get(id);
    if (!connection) {
      throw new Error(`WebSocket connection with id "${id}" not found`);
    }

    connection.socket.send(JSON.stringify(data));
  }

  /**
   * Retrieves the WebSocketConnection object for the specified ID.
   * @param id - The ID of the WebSocket connection to retrieve.
   * @returns The WebSocketConnection object associated with the specified ID.
   * @throws Will throw an error if no connection with the specified ID is found.
   */
  get(id: string) {
    const connection = this.connections.get(id);
    if (!connection) {
      throw new Error(`WebSocket connection with id "${id}" not found`);
    }

    return connection;
  }

  /**
   * Checks if a WebSocket connection with the specified ID exists.
   * @param id - The ID of the WebSocket connection to check.
   * @returns True if a connection with the specified ID exists, false otherwise.
   */
  has(id: string) {
    return this.connections.has(id);
  }
}
