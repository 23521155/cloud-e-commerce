"use client";

import { placeOrder } from "@/app/(shop)/checkout/actions";
import { BasketSkeleton } from "@/components/basket/BasketSkeleton";
import { Checkout, type CheckoutPrefill } from "@/components/checkout/Checkout";
import { MotionController } from "@/components/motion/MotionController";
import { DotButton } from "@/components/ui/DotButton";
import { subtotal } from "@/lib/basket";
import { useBasketLines } from "@/lib/basket-store";

/** The checkout page. The basket lives in this browser, so it is read here. */
export function CheckoutView({ prefill }: { prefill?: CheckoutPrefill }) {
  const { lines: all, failed } = useBasketLines();
  // Sold copies are dropped here: they cannot be bought
  const lines = all?.filter((l) => !l.sold) ?? null;

  return (
    <>
      <MotionController />

      <section className="work basket" aria-labelledby="checkout-title">
        <div className="catalogue__head">
          <h1 id="checkout-title" data-reveal className="big-sans m-0 text-[clamp(2.4rem,6vw,7rem)]">
            <span className="line">
              <span>Check</span>
            </span>
            <span className="line">
              <span className="em-italic text-gold-400">out</span>
            </span>
          </h1>
          <p className="m-0 max-w-[34ch] text-[clamp(17px,1.3vw,21px)] leading-relaxed text-parchment-200">
            Where to send the parcel and how you would like to pay.
          </p>
        </div>

        {failed ? (
          <div className="catalogue__empty" role="alert">
            <p className="m-0 font-serif text-[clamp(1.6rem,3vw,2.6rem)] uppercase">The shelves did not answer</p>
            <p className="m-0 mt-3 text-parchment-200">Your basket is safe in this browser. Reload the page in a moment.</p>
          </div>
        ) : lines === null ? (
          <BasketSkeleton />
        ) : lines.length === 0 ? (
          <div className="catalogue__empty">
            <p className="m-0 font-serif text-[clamp(1.6rem,3vw,2.6rem)] uppercase">Nothing to check out</p>
            <p className="m-0 mt-3 text-parchment-200">Your basket has no copies that can still be bought.</p>
            <DotButton href="/catalogue" className="mt-8">
              Browse the shelves
            </DotButton>
          </div>
        ) : (
          <Checkout lines={lines} total={subtotal(lines)} action={placeOrder} prefill={prefill} />
        )}
      </section>
    </>
  );
}
