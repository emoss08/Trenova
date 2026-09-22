/**
 * The URL a provider posts a mailbox's mail to, and the credentials a
 * Postmark mailbox is protected by.
 *
 * The server returns only the path, because it cannot know the origin a
 * provider reaches it on. The API's base URL can: in development the API is on
 * its own port, in production it is served beside the page.
 */
export type BasicCredentials = { user: string; password: string };

export const POSTMARK_USER = "postmark";
export const POSTMARK_PASSWORD_LENGTH = 32;

const ALPHABET = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";

type RandomSource = (buffer: Uint8Array<ArrayBuffer>) => Uint8Array<ArrayBuffer>;

const cryptoRandom: RandomSource = (buffer) => crypto.getRandomValues(buffer);

export function webhookUrl(
  apiBaseUrl: string,
  pageOrigin: string,
  webhookPath: string,
  credentials?: BasicCredentials,
): string {
  const origin = new URL(apiBaseUrl, pageOrigin).origin;
  const url = new URL(webhookPath, origin);
  if (credentials !== undefined) {
    url.username = encodeURIComponent(credentials.user);
    url.password = encodeURIComponent(credentials.password);
  }

  return url.toString();
}

/**
 * A user and a password nobody could guess. Characters are drawn by
 * rejection sampling so each is equally likely: a plain modulo would favour
 * the first few letters of the alphabet.
 */
export function generatePostmarkCredentials(random: RandomSource = cryptoRandom): BasicCredentials {
  const limit = 256 - (256 % ALPHABET.length);
  let password = "";

  while (password.length < POSTMARK_PASSWORD_LENGTH) {
    const bytes = random(new Uint8Array(POSTMARK_PASSWORD_LENGTH * 2));
    for (const byte of bytes) {
      if (byte < limit && password.length < POSTMARK_PASSWORD_LENGTH) {
        password += ALPHABET[byte % ALPHABET.length];
      }
    }
  }

  return { user: POSTMARK_USER, password };
}

/** The pair as the server stores it. */
export function postmarkSecret(credentials: BasicCredentials): string {
  return `${credentials.user}:${credentials.password}`;
}
