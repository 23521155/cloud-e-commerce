import { cookies } from "next/headers";
import { visitorHeaders } from "@/lib/gateway";
import { refreshSession, type SessionTokens } from "@/lib/user-api";

// The signed-in reader's tokens, in httpOnly cookies so page scripts never see them. The access
// token goes to the gateway as Authorization: Bearer and is short-lived; the refresh token gets a
// new one (src/proxy.ts does that before a page renders).
export const ACCESS_COOKIE = "session";
export const REFRESH_COOKIE = "refresh";
/** Between the password and the authenticator code of a sign-in: the code step's token. */
export const MFA_COOKIE = "mfa";

/** Who is signed in, read from the access token's claims. */
export type Session = {
  token: string;
  userId: string;
  email: string;
  name: string;
  roles: string[];
  /** Seconds since the epoch, as in the token. */
  expiresAt: number;
};

type Claims = { sub?: string; email?: string; name?: string; roles?: string[]; exp?: number };

/**
 * The session in an access token, or null when there is none or it has expired. The signature is
 * not checked here: the cookie is only written by this server, and every request the token is used
 * for goes through the gateway, which refuses a forged one. A forged cookie can only change the
 * name shown on this reader's own pages.
 */
export function readAccessToken(token: string | undefined): Session | null {
  if (!token) return null;

  let claims: Claims;
  try {
    claims = JSON.parse(Buffer.from(token.split(".")[1] ?? "", "base64url").toString("utf8")) as Claims;
  } catch {
    return null;
  }

  if (!claims.sub || !claims.exp || claims.exp * 1000 <= Date.now()) return null;

  return {
    token,
    userId: claims.sub,
    email: claims.email ?? "",
    name: claims.name ?? "",
    roles: claims.roles ?? [],
    expiresAt: claims.exp,
  };
}

export async function getSession(): Promise<Session | null> {
  return readAccessToken((await cookies()).get(ACCESS_COOKIE)?.value);
}

/** The cookies that hold a session, with their options. Proxy sets the same ones. */
export function sessionCookies(tokens: SessionTokens) {
  const options = {
    httpOnly: true,
    sameSite: "lax" as const,
    secure: process.env.NODE_ENV === "production",
    path: "/",
  };
  return [
    { name: ACCESS_COOKIE, value: tokens.accessToken, ...options, expires: new Date(tokens.expiresAt) },
    { name: REFRESH_COOKIE, value: tokens.refreshToken, ...options, expires: new Date(tokens.refreshExpiresAt) },
  ];
}

/** Keeps a new session. Only Server Functions, Route Handlers and Proxy can set cookies. */
export async function startSession(tokens: SessionTokens) {
  const store = await cookies();
  for (const cookie of sessionCookies(tokens)) store.set(cookie);
}

export async function endSession() {
  const store = await cookies();
  store.delete(ACCESS_COOKIE);
  store.delete(REFRESH_COOKIE);
}

export async function getRefreshToken(): Promise<string | undefined> {
  return (await cookies()).get(REFRESH_COOKIE)?.value;
}

/**
 * Gets a new access token now, e.g. after the reader changed their name so the header shows it.
 * For Server Functions; a refused refresh token ends the session.
 */
export async function renewSession(): Promise<void> {
  const refreshToken = await getRefreshToken();
  if (!refreshToken) return;

  const result = await refreshSession(refreshToken, await visitorHeaders());
  if (result.ok) await startSession(result.tokens);
  else if (result.reason === "signed-out") await endSession();
}

/** Authorization header for a gateway call on behalf of the signed-in reader; empty when signed out. */
export async function authHeader(): Promise<Record<string, string>> {
  const session = await getSession();
  return session ? { Authorization: `Bearer ${session.token}` } : {};
}

/** Keeps the code step of a sign-in until the reader types the code. */
export async function startCodeStep(mfaToken: string, expiresAt: string) {
  (await cookies()).set(MFA_COOKIE, mfaToken, {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    expires: new Date(expiresAt),
  });
}

export async function codeStepToken(): Promise<string | undefined> {
  return (await cookies()).get(MFA_COOKIE)?.value;
}

export async function endCodeStep() {
  (await cookies()).delete(MFA_COOKIE);
}
