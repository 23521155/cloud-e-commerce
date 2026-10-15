// Account client for user-service, reached through the API gateway.
// Server-side only: the gateway URL is read on the server and never sent to the browser.
import { forwardedFor, GATEWAY_URL } from "@/lib/gateway";

const BASE = `${GATEWAY_URL}/api/users`;

export type NewAccount = { name: string; email: string; password: string };

/** The account as the user service stored it. */
export type Account = { id: string; email: string; name: string; role: string; createdAt: string };

export type RegisterResult =
  | { ok: true; account: Account }
  /** Another account already uses the email. */
  | { ok: false; reason: "taken" }
  /** A field was refused; the message names it. */
  | { ok: false; reason: "invalid"; message: string }
  /** Too many attempts from this visitor; the gateway's rate limit. */
  | { ok: false; reason: "busy" }
  /** The account could not be created now (service down, or an unexpected answer). */
  | { ok: false; reason: "failed" };

/** POST /api/users/auth/register. */
export async function registerAccount(account: NewAccount): Promise<RegisterResult> {
  let res: Response;
  try {
    res = await fetch(`${BASE}/auth/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json", ...(await forwardedFor()) },
      body: JSON.stringify(account),
      cache: "no-store",
    });
  } catch {
    return { ok: false, reason: "failed" };
  }

  if (res.status === 201) return { ok: true, account: (await res.json()) as Account };
  if (res.status === 409) return { ok: false, reason: "taken" };
  if (res.status === 429) return { ok: false, reason: "busy" };
  if (res.status === 400) {
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    return { ok: false, reason: "invalid", message: (body.error ?? "").replace(/^invalid input: /, "") };
  }

  return { ok: false, reason: "failed" };
}
