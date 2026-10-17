// A Google sign-in in progress: the state, nonce and PKCE verifier the user service made, and where
// to go afterwards, kept in an httpOnly cookie between /auth/google and its callback.
import { cookies } from "next/headers";

const FLOW_COOKIE = "google_flow";
/** Both route handlers live under it, and nothing else needs the cookie. */
const FLOW_PATH = "/auth/google";
/** Time to choose an account at Google and come back. */
const FLOW_SECONDS = 10 * 60;

export type GoogleFlow = { state: string; nonce: string; verifier: string; next: string };

/** Only a path on this site, so the sign-in cannot be used to send readers elsewhere. */
export function localPath(value: string | null | undefined): string {
  const path = value ?? "";
  return path.startsWith("/") && !path.startsWith("//") && !path.startsWith("/\\") ? path : "/";
}

export async function startGoogleFlow(flow: GoogleFlow) {
  (await cookies()).set(FLOW_COOKIE, JSON.stringify(flow), {
    httpOnly: true,
    // Lax still sends it on the top-level redirect back from Google.
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: FLOW_PATH,
    maxAge: FLOW_SECONDS,
  });
}

/** The sign-in in progress, if any, which is forgotten: a callback is used once. */
export async function takeGoogleFlow(): Promise<GoogleFlow | null> {
  const store = await cookies();
  const raw = store.get(FLOW_COOKIE)?.value;
  store.delete({ name: FLOW_COOKIE, path: FLOW_PATH });
  if (!raw) return null;
  try {
    const flow = JSON.parse(raw) as Partial<GoogleFlow>;
    if (typeof flow.state !== "string" || typeof flow.nonce !== "string" || typeof flow.verifier !== "string") return null;
    return { state: flow.state, nonce: flow.nonce, verifier: flow.verifier, next: localPath(flow.next) };
  } catch {
    return null;
  }
}
