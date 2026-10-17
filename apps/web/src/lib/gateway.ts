import { headers } from "next/headers";


// Base URL of the API gateway, shared by every backend client; each adds its own
// path (/api/catalog, /api/orders...). Server-side only: GATEWAY_API_URL is read
// on the server and never sent to the browser.
export const GATEWAY_URL = (process.env.GATEWAY_API_URL ?? "http://localhost:8080").replace(/\/$/, "");

/**
 * Passes the visitor's X-Forwarded-For on to the gateway, so it rate-limits and logs the visitor's
 * IP rather than this server's. The gateway only believes it when this server is a trusted proxy.
 */
export async function forwardedFor(): Promise<Record<string, string>> {
    const value = (await headers()).get("x-forwarded-for");
    return value ? { "X-Forwarded-For": value } : {};    
}
/**
 * Who is browsing, for calls that start or renew a session: the visitor's IP as above, and their
 * User-Agent, from which the gateway names the device each session is kept for. Takes the request
 * headers, so Proxy (which has the request, not next/headers) can use it too.
 */
export function visitorHeadersFrom(request: Headers): Record<string, string> {
  const out: Record<string, string> = {};
  const forwarded = request.get("x-forwarded-for");
  const agent = request.get("user-agent");
  if (forwarded) out["X-Forwarded-For"] = forwarded;
  if (agent) out["User-Agent"] = agent;
  return out;
}

export async function visitorHeaders(): Promise<Record<string, string>> {
  return visitorHeadersFrom(await headers());
}
