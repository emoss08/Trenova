/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

// Example usage of the improved popout window manager
import { Button } from "@/components/ui/button";
import { usePopoutWindow } from "./use-popout-window";

export function PopoutWindowExample() {
  const {
    isPopout,
    activeWindows,
    hasOpenWindows,
    openPopout,
    closeAllPopouts,
    focusPopout,
    sendMessage,
  } = usePopoutWindow({
    onReady: (windowId) => {
      console.log(`Popout window ${windowId} is ready`);
    },
    onClose: (windowId) => {
      console.log(`Popout window ${windowId} was closed`);
    },
    onError: (error, windowId) => {
      console.error(`Error in popout window ${windowId}:`, error);
    },
    onFocus: (windowId) => {
      console.log(`Popout window ${windowId} gained focus`);
    },
    onBlur: (windowId) => {
      console.log(`Popout window ${windowId} lost focus`);
    },
  });

  // If we're in a popout window, show different content
  if (isPopout) {
    return (
      <div className="p-4">
        <h2>This is a popout window!</h2>
        <Button onClick={() => window.close()}>Close Window</Button>
      </div>
    );
  }

  // Main window content
  return (
    <div className="space-y-4">
      <h2>Popout Window Manager Example</h2>
      
      <div className="flex gap-2">
        <Button
          onClick={() => {
            // Open a create modal
            openPopout("/equipment/configurations/equipment-manufacturers", {
              modalType: "create",
            }, {
              width: 800,
              height: 600,
              title: "Create Equipment Manufacturer",
              rememberPosition: true,
            });
          }}
        >
          Open Create Modal
        </Button>

        <Button
          onClick={() => {
            // Open an edit modal
            openPopout("/equipment/configurations/equipment-manufacturers/123", {
              modalType: "edit",
            }, {
              width: 900,
              height: 700,
              title: "Edit Equipment Manufacturer",
              rememberPosition: true,
            });
          }}
        >
          Open Edit Modal
        </Button>

        <Button
          onClick={() => {
            // Open a full-screen popout
            openPopout("/shipments", {}, {
              width: 1400,
              height: 900,
              title: "Shipments",
              hideAside: true,
              rememberPosition: true,
            });
          }}
        >
          Open Shipments
        </Button>
      </div>

      {hasOpenWindows && (
        <div className="border rounded p-4">
          <h3>Active Windows ({activeWindows.length})</h3>
          <div className="space-y-2 mt-2">
            {activeWindows.map((windowId) => (
              <div key={windowId} className="flex justify-between items-center">
                <span className="text-sm">{windowId}</span>
                <div className="flex gap-2">
                  <Button
                    size="sm"
                    onClick={() => focusPopout(windowId)}
                  >
                    Focus
                  </Button>
                  <Button
                    size="sm"
                    onClick={() => sendMessage(windowId, "test-message", { hello: "world" })}
                  >
                    Send Message
                  </Button>
                </div>
              </div>
            ))}
            <Button
              onClick={closeAllPopouts}
              variant="destructive"
              className="mt-2"
            >
              Close All Windows
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}