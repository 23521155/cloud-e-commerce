// Base URL of the API gateway, shared by every backend client; each adds its own
// path (/api/catalog, /api/orders...). Server-side only: GATEWAY_API_URL is read
// on the server and never sent to the browser.
export const GATEWAY_URL = (process.env.GATEWAY_API_URL ?? "http://localhost:8080").replace(/\/$/, "");
