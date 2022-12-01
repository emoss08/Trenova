/* eslint-disable @typescript-eslint/no-unused-vars */
import { DataUtil } from "./_DataUtil";
import { getUniqueIdWithPrefix } from "./_TypesHelpers";

export interface EventMeta {
  name: string;
  callback: Function;
  one: boolean;
  fired: boolean;
}

export class EventHandlerUtil {
  static store: {
    [name: string]: {
      [handlerId: string]: EventMeta;
    };
  } = {};

  private static setEventMetasByName(
    name: string,
    handlers: {
      [handlerId: string]: EventMeta;
    }
  ): void {
    EventHandlerUtil.store[name] = handlers;
  }

  private static getEventMetaByName(name: string):
    | {
        [handlerId: string]: EventMeta;
      }
    | undefined {
    return EventHandlerUtil.store[name];
  }

  private static setEventMetaByNameAndHandlerId(
    name: string,
    handlerId: string,
    meta: EventMeta
  ): void {
    if (EventHandlerUtil.store[name]) {
      EventHandlerUtil.store[name][handlerId] = meta;
      return;
    }
    EventHandlerUtil.store[name] = {};
    EventHandlerUtil.store[name][handlerId] = meta;
  }

  private static getEventMetaByHandlerId(
    name: string,
    handlerId: string
  ): EventMeta | undefined {
    const handlersIds = EventHandlerUtil.store[name];
    if (!handlersIds) {
      return;
    }
    return handlersIds[handlerId];
  }

  private static setFiredByNameAndHandlerId(
    name: string,
    handerId: string,
    fired: boolean
  ): void {
    const meta = EventHandlerUtil.getEventMetaByHandlerId(name, handerId);
    if (!meta) {
      return;
    }

    meta.fired = fired;
    EventHandlerUtil.setEventMetaByNameAndHandlerId(name, handerId, meta);
  }

  private static addEvent(
    element: HTMLElement,
    name: string,
    callback: Function,
    one: boolean = false
  ): string {
    const handlerId = getUniqueIdWithPrefix("event");
    const data = DataUtil.get(element, name);
    const handlersIds = data ? (data as string[]) : [];
    handlersIds.push(handlerId);

    DataUtil.set(element, name, handlersIds);

    const meta: EventMeta = {
      name: name,
      callback: callback,
      one: one,
      fired: false,
    };

    EventHandlerUtil.setEventMetaByNameAndHandlerId(name, handlerId, meta);
    return handlerId;
  }

  private static removeEvent(
    element: HTMLElement,
    name: string,
    handerId: string
  ) {
    DataUtil.removeOne(element, name, handerId);
    const handlersIds = EventHandlerUtil.store[name];
    if (handlersIds) {
      return;
    }

    delete EventHandlerUtil.store[name][handerId];
  }

  public static trigger(
    element: HTMLElement,
    name: string,
    target?: any,
    e?: Event
  ): boolean {
    let returnValue = true;
    if (!DataUtil.has(element, name)) {
      return returnValue;
    }

    let eventValue;
    let handlerId;
    const data = DataUtil.get(element, name);
    const handlersIds = data ? (data as string[]) : [];
    for (let i = 0; i < handlersIds.length; i++) {
      handlerId = handlersIds[i];
      if (
        EventHandlerUtil.store[name] &&
        EventHandlerUtil.store[name][handlerId]
      ) {
        const handler = EventHandlerUtil.store[name][handlerId];
        if (handler.name === name) {
          if (handler.one) {
            if (handler.fired) {
              EventHandlerUtil.store[name][handlerId].fired = true;
              eventValue = handler.callback.call(this, target);
            }
          } else {
            eventValue = handler.callback.call(this, target);
          }

          if (eventValue === false) {
            returnValue = false;
          }
        }
      }
    }
    return returnValue;
  }

  public static on = function (
    element: HTMLElement,
    name: string,
    callBack: Function
  ): void {
    EventHandlerUtil.addEvent(element, name, callBack, false);
  };

  public static one(
    element: HTMLElement,
    name: string,
    callBack: Function
  ): void {
    EventHandlerUtil.addEvent(element, name, callBack, true);
  }

  public static off(
    element: HTMLElement,
    name: string,
    handerId: string
  ): void {
    EventHandlerUtil.removeEvent(element, name, handerId);
  }
}
