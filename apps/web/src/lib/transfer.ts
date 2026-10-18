import { cookies } from "next/headers";
import type { PlacedOrder, TransferInstructions } from "@/lib/order-api";

// The order a reader is paying by bank transfer, kept in an httpOnly cookie. The order service gives
// the instructions once, when the order is placed; the cookie lets /checkout/transfer show them and
// look the order up (code + email) to see when it is paid. It is only written by this server, so a
// link cannot make the page show someone else's QR code.
export const TRANSFER_COOKIE = "pending-transfer";

export type PendingTransfer = {
  code: string;
  email: string;
  /** When the copies are released: the last moment a transfer still counts. */
  expiresAt: string;
  transfer: TransferInstructions;
};

/** Keeps a new transfer. Only Server Functions and Route Handlers can set cookies. */
export async function savePendingTransfer(order: PlacedOrder, email: string) {
  if (!order.transfer || !order.expiresAt) return;

  const pending: PendingTransfer = { code: order.code, email, expiresAt: order.expiresAt, transfer: order.transfer };

  (await cookies()).set(TRANSFER_COOKIE, JSON.stringify(pending), {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    expires: new Date(order.expiresAt),
  });
}

/** The transfer being paid, or null when there is none or the cookie is not ours. */
export async function readPendingTransfer(): Promise<PendingTransfer | null> {
  const value = (await cookies()).get(TRANSFER_COOKIE)?.value;
  if (!value) return null;

  try {
    const p = JSON.parse(value) as PendingTransfer;
    if (typeof p.code !== "string" || typeof p.email !== "string" || typeof p.expiresAt !== "string" || !p.transfer) return null;
    return p;
  } catch {
    return null;
  }
}

export async function clearPendingTransfer() {
  (await cookies()).delete(TRANSFER_COOKIE);
}
