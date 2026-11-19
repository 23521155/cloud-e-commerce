import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { DotButton } from "@/components/ui/DotButton";
import { vnpayReturn } from "@/lib/order-api";

export const metadata: Metadata = {
  title: "Payment — Marginalleya",
};

/** Seconds between two checks while the payment is being confirmed. */
const RECHECK_SECONDS = 3;

/**
 * Where VNPay sends the reader back after paying. The query string is VNPay's signed result; the
 * order service checks it and answers with the order as it is now, so nothing here trusts the URL.
 */
export default async function VnpayReturnPage({ searchParams }: PageProps<"/checkout/vnpay-return">) {
  const query = new URLSearchParams();
  for (const [name, value] of Object.entries(await searchParams)) {
    for (const one of [value].flat()) if (one !== undefined) query.append(name, one);
  }

  const result = await vnpayReturn(query.toString());

  // Paid: the usual thank-you page, which also empties the basket
  if (result.ok && result.order.paymentStatus === "PAID") {
    redirect(`/checkout/confirmed?method=${result.order.paymentMethod}&code=${encodeURIComponent(result.order.code)}`);
  }

  // VNPay took the money but its call to the order service has not landed yet
  const confirming = result.ok && result.paymentSucceeded && result.order.status === "AWAITING_PAYMENT";
  // The order service could not be asked; the same signed URL can be checked again
  const unreachable = !result.ok && result.reason === "failed";

  return (
    <section className="work basket" aria-labelledby="pay-result-title">
      {(confirming || unreachable) && <meta httpEquiv="refresh" content={String(RECHECK_SECONDS)} />}

      <div className="co-done">
        <span className="co-done__stamp" aria-hidden="true">
          {confirming || unreachable ? "Checking" : "Not paid"}
        </span>

        <h1 id="pay-result-title" className="big-sans m-0 mt-10 text-[clamp(2.2rem,5vw,5.5rem)]">
          {confirming || unreachable ? (
            <>
              One <span className="em-italic text-gold-400">moment</span>
            </>
          ) : (
            <>
              Payment <span className="em-italic text-gold-400">not completed</span>
            </>
          )}
        </h1>

        {result.ok && <p className="m-0 mt-8 font-serif text-[clamp(1.6rem,3vw,2.6rem)] tabular-nums">Order {result.order.code}</p>}

        <p className="m-0 mt-4 max-w-[44ch] text-[clamp(17px,1.3vw,21px)] leading-relaxed text-parchment-200" role="status">
          {confirming
            ? "VNPay has your payment and is telling us about it. This page checks again by itself in a few seconds."
            : unreachable
              ? "We could not check your payment just now. This page tries again by itself in a few seconds."
              : result.ok
                ? "Nothing was charged and this order is cancelled. Your copies are still in your basket, so you can try again or choose another way to pay."
                : "We could not read this payment result. If money left your account, keep your bank’s receipt and contact us with it."}
        </p>

        {!confirming && !unreachable && (
          <div className="mt-10 flex flex-wrap justify-center gap-x-10 gap-y-6">
            <DotButton href="/checkout" variant="solid">
              Back to checkout
            </DotButton>
            <DotButton href="/cart">View basket</DotButton>
          </div>
        )}
      </div>
    </section>
  );
}
