# Email Template Manager UI

This is a Svelte-based UI for the Email Template Manager microservice. It provides a modern interface for managing email templates and their sample data.

## Features

- Monaco Editor integration for HTML and JSON editing
- Live preview of email templates
- Real-time updates via WebSockets
- Sample data management
- Dark mode UI

## Development

### Prerequisites

- Node.js (version 16 or higher)
- npm (version 7 or higher)

### Installation

```bash
# Install dependencies
npm install
```

### Running in Development Mode

```bash
# Start the development server
npm run dev
```

This will start the Svelte development server at http://localhost:3000. Note that you will also need to run the Go backend server to handle API requests.

### Building for Production

```bash
# Build the application
npm run build
```

The built files will be in the `build` directory, which the Go server will serve.

## Integration with Go Backend

The UI communicates with the Go backend through the following endpoints:

- `/api/templates` - List, get, update, and preview templates
- `/api/samples` - List, get, and update sample data
- `/ws` - WebSocket endpoint for real-time updates

## Folder Structure

- `src/` - Source code
  - `lib/` - Reusable components
  - `routes/` - SvelteKit routes
- `static/` - Static assets
- `build/` - Build output (generated)
