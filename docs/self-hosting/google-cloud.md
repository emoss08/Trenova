# Google Cloud APIs

Everything in Trenova that calls Google, which Google Cloud APIs each piece needs enabled, and how
to set up its key or identity. Enable an API in the Google Cloud console under **APIs & Services →
Library**, or with `gcloud services enable <service> --project <project>`.

| Feature | Who sets it up | Enable | Credential |
|---|---|---|---|
| [Maps in the browser](#google-maps) | Each organization, in **Integrations** | Maps JavaScript API (`maps-backend.googleapis.com`) | The Google Maps integration's API key |
| [Address search](#google-maps) | Each organization, in **Integrations** | Places API (New) (`places.googleapis.com`) | The same API key |
| [Map styles](#map-id-light-and-dark-styles) | Each organization, optional | Nothing extra | A Map ID from the key's project |
| [Claude on Vertex AI](#claude-on-vertex-ai) | Each organization, in **AI control** | Vertex AI API (`aiplatform.googleapis.com`) | Service account key, or the server's identity |
| [Claude Platform on Google Cloud](#claude-platform-on-google-cloud) | Each organization, in **AI control** | Claude Platform on Google Cloud (`claude.googleapis.com`) | Service account key, or the server's identity |
| [Encryption keys](#cloud-kms) | The server's operator | Cloud Key Management Service (KMS) API (`cloudkms.googleapis.com`) | The server's identity |

Nothing else in Trenova calls Google. Mileage and routing come from PC*Miler and the distance
engine, not Google's Routes, Directions, Distance Matrix or Geocoding APIs, so leave those off.
Browser push notifications in Chrome go through `fcm.googleapis.com`, but the browser handles
that with Trenova's own VAPID keys; there is nothing to enable or configure in Google Cloud.

## Google Maps

One API key per organization, saved on the **Google Maps** card in
**Admin → Integrations**. Trenova uses it in two places:

- **The browser** loads the Maps JavaScript API with it for every map: the shipment map with its
  geofences, traffic and weather layers, location geofence editing, report maps, the freight
  outlook reach map, the Desk route map and the command palette's shipment preview.
- **The server** calls Places API (New) with it whenever someone searches for an address
  (address fields, the location form, a worker's home address) and when an agent places a
  location. Each search makes one Autocomplete (New) request and up to five Place Details (New)
  requests.

Enable both APIs in the key's project:

```sh
gcloud services enable maps-backend.googleapis.com places.googleapis.com --project <project>
```

Enable **Places API (New)**, not the legacy **Places API**. Trenova does not call the legacy
API, and Google no longer enables it for new projects. If only the legacy one is on, address
search fails with "Google Maps rejected the API key".

### Restricting the key

Because the same key is used from the browser and from the server:

- **Restrict it by API**, to Maps JavaScript API and Places API (New). This is the restriction
  that matters.
- **Do not add an HTTP referrer (website) restriction.** The server's Places requests carry no
  referrer, so Google rejects them and address search stops working while the maps still load.
- **Do not add an IP address restriction** either: the browser's map requests come from each
  user's own address.

The key is visible to anyone signed in to the organization, which is why the API restriction
and a billing budget or per-API quota in Google Cloud are the controls to rely on.

### Map ID: light and dark styles

Optional. Without a Map ID, maps use Google's default style in both themes.

1. In the same project as the key, open **Google Maps Platform → Map management** and create a
   Map ID of type **JavaScript**.
2. Give it a light and a dark map style (**Map styles**, then associate both with the Map ID).
3. Paste the Map ID into **Map ID** on the Google Maps card and save.

A Map ID from another project does not work with the key. Without a Map ID, Trenova uses
Google's demo Map ID, so markers still draw, but in Google's default style.

## Claude on Vertex AI

Set up per organization in **Admin → AI control → Providers**, with **Claude on Google Vertex
AI**.

1. Enable the Vertex AI API: `gcloud services enable aiplatform.googleapis.com --project <project>`.
2. In **Vertex AI → Model Garden**, enable each Claude model you will use, in the region you
   will use (for example `us-east5`, or `global`).
3. Choose how Trenova signs in:
   - **Service account key**: create a service account with the **Vertex AI User** role
     (`roles/aiplatform.user`), create a JSON key for it, and paste the whole file into
     **Service account key (JSON)**.
   - **This server's credentials**: only offered when the operator sets
     `ai.serverCloudCredentials: true`. Trenova then uses the server's Application Default
     Credentials (see [The server's identity](#the-servers-identity)), and that identity needs
     **Vertex AI User**.
4. Enter the **Region** and **Project ID**, and type the **Model ID** from Model Garden. Vertex
   AI does not list models to Trenova, so **Fetch models** is not offered.

## Claude Platform on Google Cloud

The Claude API run by Anthropic and billed through Google Cloud. Set up per organization in
**Admin → AI control → Providers**, with **Claude Platform on Google Cloud**.

1. Set up Claude Platform on Google Cloud for the project, following Anthropic's and Google's
   onboarding for it, so that its API (`claude.googleapis.com`) is enabled and an Anthropic
   workspace is linked to the project.
2. Sign in with a **Service account key** (a service account with access to the platform, its
   JSON key pasted into **Service account key (JSON)**), or with **This server's credentials**
   under the same `ai.serverCloudCredentials` rule as Vertex AI.
3. Enter the **Region** (the Google Cloud location, usually `global`), **Project ID** and the
   Anthropic **Workspace ID**. **Fetch models** works here.

## Cloud KMS

For the server's operator. Production and staging refuse to start without a Cloud KMS key:
Trenova wraps the keys that encrypt stored secrets (integration credentials, provider keys)
with it.

1. Enable the API: `gcloud services enable cloudkms.googleapis.com --project <project>`.
2. Create a symmetric encrypt/decrypt key, by hand or with Cloud KMS Autokey.
3. Grant the server's identity **Cloud KMS CryptoKey Encrypter/Decrypter**
   (`roles/cloudkms.cryptoKeyEncrypterDecrypter`) on that key.
4. Configure it:

   ```yaml
   security:
     encryption:
       keyManager: gcp-autokey
       gcpKms:
         cryptoKey: projects/<project>/locations/<location>/keyRings/<ring>/cryptoKeys/<key>
         credentialsFile: "" # a service account JSON key; empty uses the server's identity
   ```

## The server's identity

Cloud KMS, and Vertex AI or Claude Platform with **This server's credentials**, sign in with
Google's Application Default Credentials (`cloud-platform` scope), found in this order:

1. The file named by `GOOGLE_APPLICATION_CREDENTIALS` (or, for KMS only,
   `security.encryption.gcpKms.credentialsFile`).
2. On Google Cloud (GKE with Workload Identity, Compute Engine, Cloud Run), the attached service
   account, with no file at all.

Leave `ai.serverCloudCredentials` false when several organizations share one server: with it on,
any organization's administrator could spend the operator's Google Cloud account.

## When something is wrong

| Symptom | Cause |
|---|---|
| Address search says "Google Maps rejected the API key" | Places API (New) is not enabled in the key's project, the key is restricted to other APIs, or it has an HTTP referrer restriction |
| Maps show "This page can't load Google Maps correctly" | Maps JavaScript API is not enabled, billing is off on the project, or the key's restrictions block it |
| Maps load but ignore the light/dark theme | No Map ID, a Map ID from another project, or no style associated with it |
| A Vertex AI provider's test fails with permission denied | The Vertex AI API is off, the model is not enabled in Model Garden in that region, or the identity lacks **Vertex AI User** |
| The server will not start: "production and staging require a GCP KMS crypto key resource" | `security.encryption.gcpKms.cryptoKey` is empty |
| The server will not start: "the GCP KMS client could not be created" | No Application Default Credentials found, or the credentials file is unreadable |
