// Account client for user-service, reached through the API gateway.
// Server-side only: the gateway URL is read on the server and never sent to the browser.
import { forwardedFor, GATEWAY_URL } from "@/lib/gateway";

const BASE = `${GATEWAY_URL}/api/users`;

export type NewAccount = { name: string; email: string; password: string };

/** Contact and delivery details kept on the account; empty strings are details not given. */
export type ProfileDetails = { phone: string; address: string; city: string; postalCode: string; country: string };

/** The account as the user service stored it. */
export type Account = { id: string; email: string; name: string; role: string; createdAt: string } & ProfileDetails;

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

export type LoginResult =
  | { ok: true; accessToken: string; expiresAt: string; account: Account }
  /** The email or the password is wrong; which one is not told. */
  | { ok: false; reason: "wrong" }
  | { ok: false; reason: "busy" }
  | { ok: false; reason: "failed" };

/** POST /api/users/auth/login. */
export async function login(email: string, password: string): Promise<LoginResult> {
  let res: Response;
  try {
    res = await fetch(`${BASE}/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json", ...(await forwardedFor()) },
      body: JSON.stringify({ email, password }),
      cache: "no-store",
    });
  } catch {
    return { ok: false, reason: "failed" };
  }

  if (res.ok) {
    const body = (await res.json()) as { accessToken: string; expiresAt: string; user: Account };
    return { ok: true, accessToken: body.accessToken, expiresAt: body.expiresAt, account: body.user };
  }
  if (res.status === 401) return { ok: false, reason: "wrong" };
  if (res.status === 429) return { ok: false, reason: "busy" };

  return { ok: false, reason: "failed" };
}

export type MeResult =
  | { ok: true; account: Account }
  /** The token was refused: expired, or the account is gone. */
  | { ok: false; reason: "signed-out" }
  | { ok: false; reason: "failed" };

/** GET /api/users/me with the reader's access token. */
export async function me(accessToken: string): Promise<MeResult> {
  let res: Response;
  try {
    res = await fetch(`${BASE}/me`, {
      headers: { Authorization: `Bearer ${accessToken}`, ...(await forwardedFor()) },
      cache: "no-store",
    });
  } catch {
    return { ok: false, reason: "failed" };
  }

  if (res.ok) return { ok: true, account: (await res.json()) as Account };
  if (res.status === 401) return { ok: false, reason: "signed-out" };

  return { ok: false, reason: "failed" };
}

export type UpdateProfileResult =
  | { ok: true; account: Account }
  | { ok: false; reason: "signed-out" }
  /** A field was refused; the message names it. */
  | { ok: false; reason: "invalid"; message: string }
  | { ok: false; reason: "failed" };

/** PUT /api/users/me: the name and the whole profile. The email cannot be changed here. */
export async function updateProfile(accessToken: string, profile: { name: string } & ProfileDetails): Promise<UpdateProfileResult> {
  let res: Response;
  try {
    res = await fetch(`${BASE}/me`, {
      method: "PUT",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${accessToken}`, ...(await forwardedFor()) },
      body: JSON.stringify(profile),
      cache: "no-store",
    });
  } catch {
    return { ok: false, reason: "failed" };
  }

  if (res.ok) return { ok: true, account: (await res.json()) as Account };
  if (res.status === 401) return { ok: false, reason: "signed-out" };
  if (res.status === 400) {
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    return { ok: false, reason: "invalid", message: (body.error ?? "").replace(/^invalid input: /, "") };
  }

  return { ok: false, reason: "failed" };
}
