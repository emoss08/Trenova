# Plugin Management

## Overview

Monta offers a streamlined plugin system, facilitating swift integration of new applications and modules without undue
complexity. Through the plugin manager, developers can readily install zip files from the Monta Plugin repository on
GIT, which are then seamlessly integrated into the Monta application. Moreover, the plugin manager empowers developers
to easily activate or deactivate plugins as needed, enabling swift and efficient testing without the inconvenience of
uninstallation.

### API Endpoints

GET /api/plugin_list/ - Returns a list of all plugins installed on the system.
POST /api/plugin_install/{plugin_name} - Installs a plugin from the Monta Plugin repository on GIT.

#### Example

1. /api/plugin_install/billing/ - Installs the billing plugin from the Monta Plugin repository on GIT.
2. Run `makemigrations` and `migrate` to update the database.
3. Turn off application server & Run `runserver` to start the server.