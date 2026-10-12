// Order client for order-service, reached through the API gateway.
// Server-side only: the gateway URL is read on the server and never sent to the browser.
import { GATEWAY_URL } from "@/lib/gateway";

const BASE = `${GATEWAY_URL}/api/orders`;

/** A guest cash-on-delivery checkout. Prices are not sent: the order service reads them from the catalogue. */
export type NewOrder = {
  slugs: string[];
  paymentMethod: "cod";
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
      headers: { "Content-Type": "application/json", "Idempotency-Key": key },
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
