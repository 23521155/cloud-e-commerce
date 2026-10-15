// Account client for user-service, reached through the API gateway.
// Server-side only: the gateway URL is read on the server and never sent to the browser.
import { forwardedFor, GATEWAY_URL, visitorHeaders } from "@/lib/gateway";

const BASE = `${GATEWAY_URL}/api/users`;

export type NewAccount = { name: string; email: string; password: string };

/** Contact and delivery details kept on the account; empty strings are details not given. */
export type ProfileDetails = { phone: string; address: string; city: string; postalCode: string; country: string };

/** The account as the user service stored it. */
export type Account = {
  id: string;
  email: string;
  name: string;
  role: string;
  createdAt: string;
  /** When the reader confirmed the address; null until then. */
  emailVerifiedAt: string | null;
  /** When two-step sign-in was turned on; null while it is off. */
  totpEnabledAt: string | null;
} & ProfileDetails;

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

/** What signing in or refreshing gives: an access token, and the refresh token that gets the next one. */
export type SessionTokens = { accessToken: string; expiresAt: string; refreshToken: string; refreshExpiresAt: string };

type TokensBody = SessionTokens & { user: Account };

const tokensOf = ({ accessToken, expiresAt, refreshToken, refreshExpiresAt }: TokensBody): SessionTokens => ({
  accessToken,
  expiresAt,
  refreshToken,
  refreshExpiresAt,
});

export type LoginResult =
  | { ok: true; tokens: SessionTokens; account: Account }
  /** The password was right and the account asks for an authenticator code: send it with mfaToken. */
  | { ok: false; reason: "mfa"; mfaToken: string; mfaExpiresAt: string }
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
      headers: { "Content-Type": "application/json", ...(await visitorHeaders()) },
      body: JSON.stringify({ email, password }),
      cache: "no-store",
    });
  } catch {
    return { ok: false, reason: "failed" };
  }

  if (res.ok) {
    const body = (await res.json()) as TokensBody | { mfaRequired: true; mfaToken: string; mfaExpiresAt: string };
    if ("mfaRequired" in body) return { ok: false, reason: "mfa", mfaToken: body.mfaToken, mfaExpiresAt: body.mfaExpiresAt };
    return { ok: true, tokens: tokensOf(body), account: body.user };
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

export type RefreshResult =
  | { ok: true; tokens: SessionTokens }
  /** The refresh token was refused: expired, signed out, or reused. */
  | { ok: false; reason: "signed-out" }
  | { ok: false; reason: "failed" };

/**
 * POST /api/users/auth/refresh: new tokens for the same session. visitor are the browsing
 * reader's headers (see visitorHeadersFrom), passed in so Proxy can call this.
 */
export async function refreshSession(refreshToken: string, visitor: Record<string, string>): Promise<RefreshResult> {
  let res: Response;
  try {
    res = await fetch(`${BASE}/auth/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json", ...visitor },
      body: JSON.stringify({ refreshToken }),
      cache: "no-store",
    });
  } catch {
    return { ok: false, reason: "failed" };
  }

  if (res.ok) return { ok: true, tokens: tokensOf((await res.json()) as TokensBody) };
  if (res.status === 401) return { ok: false, reason: "signed-out" };

  return { ok: false, reason: "failed" };
}

/** POST /api/users/auth/logout: ends the session on the user service. Best effort. */
export async function logout(refreshToken: string): Promise<void> {
  try {
    await fetch(`${BASE}/auth/logout`, {
      method: "POST",
      headers: { "Content-Type": "application/json", ...(await forwardedFor()) },
      body: JSON.stringify({ refreshToken }),
      cache: "no-store",
    });
  } catch {
    // The cookies are cleared anyway; the session then just runs out.
  }
}

export type ChangePasswordResult =
  /** Every other session has ended; these tokens are the new session of this device. */
  | { ok: true; tokens: SessionTokens }
  | { ok: false; reason: "wrong-current" }
  /** The new password was refused; the message says why. */
  | { ok: false; reason: "invalid"; message: string }
  | { ok: false; reason: "signed-out" }
  | { ok: false; reason: "failed" };

/** POST /api/users/me/password. */
export async function changePassword(accessToken: string, current: string, password: string): Promise<ChangePasswordResult> {
  let res: Response;
  try {
    res = await fetch(`${BASE}/me/password`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${accessToken}`, ...(await visitorHeaders()) },
      body: JSON.stringify({ current, password }),
      cache: "no-store",
    });
  } catch {
    return { ok: false, reason: "failed" };
  }

  if (res.ok) return { ok: true, tokens: tokensOf((await res.json()) as TokensBody) };
  if (res.status === 403) return { ok: false, reason: "wrong-current" };
  if (res.status === 401) return { ok: false, reason: "signed-out" };
  if (res.status === 400) {
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    return { ok: false, reason: "invalid", message: (body.error ?? "").replace(/^invalid input: /, "") };
  }

  return { ok: false, reason: "failed" };
}

export type VerifyEmailResult = { ok: true } | { ok: false; reason: "invalid" } | { ok: false; reason: "failed" };

/** POST /api/users/auth/verify-email with the token from the link in the email. No sign-in needed. */
export async function verifyEmail(token: string): Promise<VerifyEmailResult> {
  let res: Response;
  try {
    res = await fetch(`${BASE}/auth/verify-email`, {
      method: "POST",
      headers: { "Content-Type": "application/json", ...(await forwardedFor()) },
      body: JSON.stringify({ token }),
      cache: "no-store",
    });
  } catch {
    return { ok: false, reason: "failed" };
  }

  if (res.ok) return { ok: true };
  if (res.status === 400) return { ok: false, reason: "invalid" };

  return { ok: false, reason: "failed" };
}

export type ResendResult =
  | { ok: true }
  | { ok: false; reason: "already-verified" | "too-soon" | "signed-out" | "failed" };

/** POST /api/users/me/verify-email: a new confirmation link to the reader's address. */
export async function resendVerification(accessToken: string): Promise<ResendResult> {
  let res: Response;
  try {
    res = await fetch(`${BASE}/me/verify-email`, {
      method: "POST",
      headers: { Authorization: `Bearer ${accessToken}`, ...(await forwardedFor()) },
      cache: "no-store",
    });
  } catch {
    return { ok: false, reason: "failed" };
  }

  if (res.status === 202) return { ok: true };
  if (res.status === 409) return { ok: false, reason: "already-verified" };
  if (res.status === 429) return { ok: false, reason: "too-soon" };
  if (res.status === 401) return { ok: false, reason: "signed-out" };

  return { ok: false, reason: "failed" };
}

/** POST /api/users/auth/forgot-password. The answer is the same whether or not the address has an account. */
export async function requestPasswordReset(email: string): Promise<{ ok: boolean }> {
  try {
    const res = await fetch(`${BASE}/auth/forgot-password`, {
      method: "POST",
      headers: { "Content-Type": "application/json", ...(await forwardedFor()) },
      body: JSON.stringify({ email }),
      cache: "no-store",
    });
    return { ok: res.status === 202 };
  } catch {
    return { ok: false };
  }
}

export type ResetPasswordResult =
  | { ok: true }
  /** The link was used already or has expired. */
  | { ok: false; reason: "invalid-link" }
  /** The new password was refused. */
  | { ok: false; reason: "weak" }
  | { ok: false; reason: "failed" };

/** POST /api/users/auth/reset-password with the token from the emailed link. */
export async function resetPassword(token: string, password: string): Promise<ResetPasswordResult> {
  let res: Response;
  try {
    res = await fetch(`${BASE}/auth/reset-password`, {
      method: "POST",
      headers: { "Content-Type": "application/json", ...(await forwardedFor()) },
      body: JSON.stringify({ token, password }),
      cache: "no-store",
    });
  } catch {
    return { ok: false, reason: "failed" };
  }

  if (res.status === 204) return { ok: true };
  if (res.status === 400) {
    const body = (await res.json().catch(() => ({}))) as { error?: string };
    return { ok: false, reason: body.error?.startsWith("invalid input:") ? "weak" : "invalid-link" };
  }

  return { ok: false, reason: "failed" };
}

export type VerifyCodeResult =
  | { ok: true; tokens: SessionTokens }
  | { ok: false; reason: "wrong-code" }
  /** The code step ran out of time or of tries: sign in again. */
  | { ok: false; reason: "expired" }
  | { ok: false; reason: "failed" };

/** POST /api/users/auth/mfa/verify: finishes a sign-in with an authenticator code. */
export async function verifyMfa(mfaToken: string, code: string): Promise<VerifyCodeResult> {
  let res: Response;
  try {
    res = await fetch(`${BASE}/auth/mfa/verify`, {
      method: "POST",
      headers: { "Content-Type": "application/json", ...(await forwardedFor()) },
      body: JSON.stringify({ mfaToken, code }),
      cache: "no-store",
    });
  } catch {
    return { ok: false, reason: "failed" };
  }

  if (res.ok) return { ok: true, tokens: tokensOf((await res.json()) as TokensBody) };
  if (res.status === 400) return { ok: false, reason: "wrong-code" };
  if (res.status === 401) return { ok: false, reason: "expired" };

  return { ok: false, reason: "failed" };
}

export type TotpSetupResult =
  | { ok: true; secret: string; otpauthUrl: string }
  | { ok: false; reason: "already-on" | "unavailable" | "signed-out" | "failed" };

/** POST /api/users/me/mfa/totp/setup: a new secret for the reader's authenticator app. */
export async function setupTotp(accessToken: string): Promise<TotpSetupResult> {
  let res: Response;
  try {
    res = await fetch(`${BASE}/me/mfa/totp/setup`, {
      method: "POST",
      headers: { Authorization: `Bearer ${accessToken}`, ...(await forwardedFor()) },
      cache: "no-store",
    });
  } catch {
    return { ok: false, reason: "failed" };
  }

  if (res.ok) return { ok: true, ...((await res.json()) as { secret: string; otpauthUrl: string }) };
  if (res.status === 409) return { ok: false, reason: "already-on" };
  if (res.status === 503) return { ok: false, reason: "unavailable" };
  if (res.status === 401) return { ok: false, reason: "signed-out" };

  return { ok: false, reason: "failed" };
}

export type TotpConfirmResult =
  /** Recovery codes are shown this once. */
  | { ok: true; recoveryCodes: string[] }
  | { ok: false; reason: "wrong-code" | "conflict" | "signed-out" | "failed" };

/** POST /api/users/me/mfa/totp/confirm: turns two-step sign-in on with a first code. */
export async function confirmTotp(accessToken: string, code: string): Promise<TotpConfirmResult> {
  let res: Response;
  try {
    res = await fetch(`${BASE}/me/mfa/totp/confirm`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${accessToken}`, ...(await forwardedFor()) },
      body: JSON.stringify({ code }),
      cache: "no-store",
    });
  } catch {
    return { ok: false, reason: "failed" };
  }

  if (res.ok) return { ok: true, ...((await res.json()) as { recoveryCodes: string[] }) };
  if (res.status === 400) return { ok: false, reason: "wrong-code" };
  if (res.status === 409) return { ok: false, reason: "conflict" };
  if (res.status === 401) return { ok: false, reason: "signed-out" };

  return { ok: false, reason: "failed" };
}

export type TotpDisableResult = { ok: true } | { ok: false; reason: "wrong-password" | "wrong-code" | "signed-out" | "failed" };

/** POST /api/users/me/mfa/totp/disable: needs the password and a code (or recovery code). */
export async function disableTotp(accessToken: string, password: string, code: string): Promise<TotpDisableResult> {
  let res: Response;
  try {
    res = await fetch(`${BASE}/me/mfa/totp/disable`, {
      method: "POST",
      headers: { "Content-Type": "application/json", Authorization: `Bearer ${accessToken}`, ...(await forwardedFor()) },
      body: JSON.stringify({ password, code }),
      cache: "no-store",
    });
  } catch {
    return { ok: false, reason: "failed" };
  }

  if (res.status === 204) return { ok: true };
  if (res.status === 403) return { ok: false, reason: "wrong-password" };
  if (res.status === 400) return { ok: false, reason: "wrong-code" };
  if (res.status === 401) return { ok: false, reason: "signed-out" };

  return { ok: false, reason: "failed" };
}
