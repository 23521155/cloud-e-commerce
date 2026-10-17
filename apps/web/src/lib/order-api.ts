// Order client for order-service, reached through the API gateway.
// Server-side only: the gateway URL is read on the server and never sent to the browser.
import { forwardedFor, GATEWAY_URL } from "@/lib/gateway";
import { authHeader } from "@/lib/session";

const BASE = `${GATEWAY_URL}/api/orders`;

/** Methods the order service places for real; the others are still mocked by /checkout/pay. */
export type PlaceableMethod = "cod" | "vietqr" | "card";

export const isPlaceable = (id: string): id is PlaceableMethod => id === "cod" || id === "vietqr" || id === "card";

/**
 * A checkout. Prices are not sent: the order service reads them from the catalogue. A signed-in
 * reader's token goes with it, so the order is theirs; without one it is a guest order.
 */
export type NewOrder = {
  slugs: string[];
  paymentMethod: PlaceableMethod;
  email: string;
  phone: string;
  fullName: string;
  address: string;
  city: string;
  postalCode: string | null;
  country: string;
};

/** The order as the order service stored it. */
export type PlacedOrder = {
  code: string;
  status: string;
  paymentMethod: string;
  paymentStatus: string;
  subtotalVnd: number;
  createdAt: string;
  items: { slug: string; title: string; priceVnd: number }[];
  /** VNPay's page for an order awaiting payment; absent once its payment window has closed. */
  paymentUrl?: string;
};

export type PlaceOrderResult =
  | { ok: true; order: PlacedOrder }
  /** Some copies were bought by someone else; nothing was ordered. */
  | { ok: false; reason: "unavailable"; slugs: string[] }
  /** A field was refused; the message names it. */
  | { ok: false; reason: "invalid"; message: string }
  /** The order could not be placed now (service down, or an unexpected answer). */
  | { ok: false; reason: "failed" };

/**
 * POST /api/orders. key is the idempotency key of this checkout: sending the same key again
 * returns the order it already placed instead of placing a second one.
 */
export async function placeOrder(key: string, { slugs, ...order }: NewOrder): Promise<PlaceOrderResult> {
  let res: Response;
  try {
    res = await fetch(BASE, {
      method: "POST",
      headers: { "Content-Type": "application/json", "Idempotency-Key": key, ...(await authHeader()), ...(await forwardedFor()) },
      body: JSON.stringify({ items: slugs.map((slug) => ({ slug })), ...order }),
      cache: "no-store",
    });
  } catch {
    return { ok: false, reason: "failed" };
  }

  // 200 is a repeated request answered with the order it already placed
  if (res.status === 201 || res.status === 200) return { ok: true, order: (await res.json()) as PlacedOrder };

  const body = (await res.json().catch(() => ({}))) as { error?: string; books?: { slug: string }[] };

  if (res.status === 409) return { ok: false, reason: "unavailable", slugs: (body.books ?? []).map((b) => b.slug) };
  if (res.status === 400) return { ok: false, reason: "invalid", message: (body.error ?? "").replace(/^invalid order: /, "") };

  return { ok: false, reason: "failed" };
}

export type VnpayReturnResult =
  /** The order as it is now, and what VNPay told the customer. */
  | { ok: true; order: PlacedOrder; paymentSucceeded: boolean }
  /** The redirect was not signed by VNPay, or names no order. */
  | { ok: false; reason: "invalid" }
  | { ok: false; reason: "failed" };

/**
 * GET /api/orders/payments/vnpay/return. query is the query string VNPay put on the customer's
 * return redirect, forwarded as is: the order service checks its signature.
 */
export async function vnpayReturn(query: string): Promise<VnpayReturnResult> {
  let res: Response;
  try {
    res = await fetch(`${BASE}/payments/vnpay/return?${query}`, { cache: "no-store", headers: await forwardedFor() });
  } catch {
    return { ok: false, reason: "failed" };
  }

  if (res.ok) return { ok: true, ...((await res.json()) as { order: PlacedOrder; paymentSucceeded: boolean }) };
  if (res.status === 400 || res.status === 404) return { ok: false, reason: "invalid" };

  return { ok: false, reason: "failed" };
}

export type MyOrdersResult =
  | { ok: true; orders: PlacedOrder[] }
  /** The token was refused (expired); the reader has to sign in again. */
  | { ok: false; reason: "signed-out" }
  | { ok: false; reason: "failed" };

export type LookupResult =
  | { ok: true; order: PlacedOrder }
  /** No order has this code and email (the service gives one answer for both). */
  | { ok: false; reason: "not-found" }
  | { ok: false; reason: "failed" };

/** POST /api/orders/lookup: a guest's order by its code and the email it was placed with. */
export async function lookupOrder(code: string, email: string): Promise<LookupResult> {
  let res: Response;
  try {
    res = await fetch(`${BASE}/lookup`, {
      method: "POST",
      headers: { "Content-Type": "application/json", ...(await forwardedFor()) },
      body: JSON.stringify({ code, email }),
      cache: "no-store",
    });
  } catch {
    return { ok: false, reason: "failed" };
  }

  if (res.ok) return { ok: true, order: (await res.json()) as PlacedOrder };
  if (res.status === 404 || res.status === 400) return { ok: false, reason: "not-found" };

  return { ok: false, reason: "failed" };
}

/** GET /api/orders: the signed-in reader's orders, newest first. */
export async function listMyOrders(accessToken: string): Promise<MyOrdersResult> {
  let res: Response;
  try {
    res = await fetch(BASE, { cache: "no-store", headers: { Authorization: `Bearer ${accessToken}`, ...(await forwardedFor()) } });
  } catch {
    return { ok: false, reason: "failed" };
  }

  if (res.ok) return { ok: true, orders: ((await res.json()) as { items: PlacedOrder[] }).items };
  if (res.status === 401) return { ok: false, reason: "signed-out" };

  return { ok: false, reason: "failed" };
}
