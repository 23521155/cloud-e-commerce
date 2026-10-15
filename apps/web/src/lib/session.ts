import { cookies } from "next/headers";

// The signed-in reader's access token, in an httpOnly cookie so page scripts never see it. The
// web server sends it to the gateway as Authorization: Bearer; the gateway checks its signature.
const COOKIE = "session";

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
 * The current session, or null when signed out or the token has expired. The signature is not
 * checked here: the cookie is only written by this server, and every request the token is used
 * for goes through the gateway, which refuses a forged one. A forged cookie can only change the
 * name shown on this reader's own pages.
 */
export async function getSession(): Promise<Session | null> {
  const token = (await cookies()).get(COOKIE)?.value;
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

/** Keeps an access token until it expires. Only server actions and route handlers can set cookies. */
export async function startSession(token: string, expiresAt: string) {
  (await cookies()).set(COOKIE, token, {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    expires: new Date(expiresAt),
  });
}

export async function endSession() {
  (await cookies()).delete(COOKIE);
}

/** Authorization header for a gateway call on behalf of the signed-in reader; empty when signed out. */
export async function authHeader(): Promise<Record<string, string>> {
  const session = await getSession();
  return session ? { Authorization: `Bearer ${session.token}` } : {};
}
