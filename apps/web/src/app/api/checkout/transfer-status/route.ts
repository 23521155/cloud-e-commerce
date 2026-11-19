import { lookupOrder } from "@/lib/order-api";
import { clearPendingTransfer, readPendingTransfer } from "@/lib/transfer";

/**
 * GET /api/checkout/transfer-status: has the order being paid by bank transfer been paid yet? The
 * order comes from the pending-transfer cookie, so the page sends nothing. The order service only
 * shows an order to its customer once it is paid, so finding it is the answer.
 *
 * state: "waiting" | "paid" | "expired" | "none" (no transfer is being paid).
 */
export async function GET() {
  const pending = await readPendingTransfer();
  if (!pending) return Response.json({ state: "none" });

  const result = await lookupOrder(pending.code, pending.email);

  if (result.ok && result.order.paymentStatus === "PAID") {
    await clearPendingTransfer();
    return Response.json({ state: "paid", code: pending.code });
  }

  // A lookup that failed is asked again by the page, so it is "waiting" too
  const expired = Date.parse(pending.expiresAt) <= Date.now();
  return Response.json({ state: expired ? "expired" : "waiting" });
}
