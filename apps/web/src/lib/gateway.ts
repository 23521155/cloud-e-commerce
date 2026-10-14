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