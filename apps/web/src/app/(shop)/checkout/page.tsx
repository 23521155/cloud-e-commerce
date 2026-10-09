import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { Checkout } from "@/components/checkout/Checkout";
import { MotionController } from "@/components/motion/MotionController";
import { DotButton } from "@/components/ui/DotButton";
import { getBasket, subtotal } from "@/lib/basket";
import { getPaymentMethod } from "@/lib/payment";

export const metadata: Metadata = {
  title: "Checkout — Marginalleya",
  description: "Delivery and payment for the copies in your basket.",
};

/** Mock order: no payment, no persistence yet. Form data stays in the POST body, never the URL. */
async function placeOrder(formData: FormData) {
  "use server";
  // Only the method id travels to the next page; contact and address details never enter the URL
  const method = getPaymentMethod(formData.get("payment"));
  if (!method) redirect("/checkout");
  // Gateway methods hand off to the provider's page (mocked by /checkout/pay); the rest finish here
  redirect(method.flow === "gateway" ? `/checkout/pay?method=${method.id}` : `/checkout/confirmed?method=${method.id}`);
}

export default async function CheckoutPage() {
  // Sold copies are dropped here: they cannot be bought
  const lines = (await getBasket()).filter((l) => !l.sold);
  const total = subtotal(lines);

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

        {lines.length === 0 ? (
          <div className="catalogue__empty">
            <p className="m-0 font-serif text-[clamp(1.6rem,3vw,2.6rem)] uppercase">Nothing to check out</p>
            <p className="m-0 mt-3 text-parchment-200">Your basket has no copies that can still be bought.</p>
            <DotButton href="/catalogue" className="mt-8">
              Browse the shelves
            </DotButton>
          </div>
        ) : (
          <Checkout lines={lines} total={total} action={placeOrder} />
        )}
      </section>
    </>
  );
}
