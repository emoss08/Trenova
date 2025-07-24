<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# Popout Window Manager

An enterprise-grade popout window management system for React applications with advanced features like window position memory, focus management, and state synchronization.

## Features

- **Smart Window Management**: Prevents duplicate windows for the same path
- **Position Memory**: Remembers window positions across sessions
- **Focus Management**: Track and control window focus states
- **State Synchronization**: Real-time state updates between parent and child windows
- **Error Recovery**: Robust error handling and automatic cleanup
- **Event System**: Comprehensive event callbacks for window lifecycle
- **Message Passing**: Secure cross-window communication
- **TypeScript Support**: Full type safety and IntelliSense

## Usage

### Basic Example

```tsx
import { usePopoutWindow } from '@/hooks/popout-window/use-popout-window';

function MyComponent() {
  const { openPopout, closePopout, isPopout } = usePopoutWindow();

  if (isPopout) {
    return <div>This is displayed in a popout window!</div>;
  }

  return (
    <button onClick={() => openPopout('/my-route', { id: '123' })}>
      Open Popout
    </button>
  );
}
```

### Advanced Example with Events

```tsx
const {
  activeWindows,
  openPopout,
  focusPopout,
  sendMessage,
  broadcastMessage,
} = usePopoutWindow({
  onReady: (windowId) => console.log('Window ready:', windowId),
  onClose: (windowId) => console.log('Window closed:', windowId),
  onFocus: (windowId) => console.log('Window focused:', windowId),
  onBlur: (windowId) => console.log('Window blurred:', windowId),
  onError: (error, windowId) => console.error('Window error:', error),
});
```

## API Reference

### usePopoutWindow Hook

#### Options

- `onReady?: (windowId: string) => void` - Called when window is loaded
- `onClose?: (windowId: string) => void` - Called when window is closed
- `onError?: (error: Error, windowId?: string) => void` - Called on errors
- `onFocus?: (windowId: string) => void` - Called when window gains focus
- `onBlur?: (windowId: string) => void` - Called when window loses focus
- `onStateChange?: (windowId: string, state: any) => void` - Called on state changes

#### Returns

- `isPopout: boolean` - Whether current window is a popout
- `popoutId: string | null` - ID of current popout window
- `activeWindows: string[]` - Array of active window IDs
- `hasOpenWindows: boolean` - Whether any windows are open
- `openPopout(path, queryParams?, options?)` - Open a new popout window
- `closePopout(windowId?)` - Close a specific window or current popout
- `closeAllPopouts()` - Close all open windows
- `focusPopout(windowId)` - Focus a specific window
- `sendMessage(windowId, type, data?)` - Send message to a window
- `broadcastMessage(type, data?)` - Send message to all windows
- `getWindowState(windowId)` - Get window state information

### PopoutWindowOptions

```typescript
type PopoutWindowOptions = {
  modalType?: "create" | "edit";
  width?: number;
  height?: number;
  left?: number;
  top?: number;
  hideHeader?: boolean;
  hideAside?: boolean;
  resizable?: boolean;
  scrollable?: boolean;
  title?: string;
  features?: WindowFeature[];
  rememberPosition?: boolean;
};
```

## Window Position Memory

The manager automatically saves window positions when `rememberPosition: true` is set. Positions are stored in localStorage and restored when opening the same path again.

## Message Passing

Send messages between windows:

```tsx
// Send to specific window
sendMessage(windowId, 'update-data', { value: 42 });

// Broadcast to all windows
broadcastMessage('theme-changed', { theme: 'dark' });

// Listen for messages (in popout window)
window.addEventListener('message', (event) => {
  if (event.data.type === 'update-data') {
    console.log('Received data:', event.data.data);
  }
});
```

## Best Practices

1. Always check `isPopout` to render appropriate content
2. Use `rememberPosition: true` for frequently used windows
3. Handle errors with the `onError` callback
4. Clean up resources when component unmounts
5. Use unique paths to prevent window conflicts
6. Set appropriate window titles for user clarity

## Security

- Messages are only accepted from the same origin
- Window features are sanitized to prevent XSS
- Query parameters are properly encoded
- Automatic cleanup prevents memory leaks