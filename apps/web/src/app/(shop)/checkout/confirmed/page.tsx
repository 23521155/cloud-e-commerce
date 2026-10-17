import type { Metadata } from "next";
import { ClearBasket } from "@/components/basket/ClearBasket";
import { DotButton } from "@/components/ui/DotButton";
import { getPaymentMethod } from "@/lib/payment";
import { getSession } from "@/lib/session";

export const metadata: Metadata = {
  title: "Order received — Marginalleya",
};

const first = (v: string | string[] | undefined) => (Array.isArray(v) ? v[0] : v);

/** Shape of an order code from the order service; anything else in the URL is ignored. */
const ORDER_CODE = /^MA-\d{6}$/;

export default async function ConfirmedPage({ searchParams }: PageProps<"/checkout/confirmed">) {
  const params = await searchParams;
  const method = getPaymentMethod(first(params.method));
  // Real orders (cash on delivery, or paid on VNPay) arrive with their code; the mocked methods have none
  const code = first(params.code);
  const placed = code !== undefined && ORDER_CODE.test(code);
  // A signed-in reader's order is on their orders page; a guest keeps the code and email instead
  const signedIn = (await getSession()) !== null;

  return (
    <section className="work basket" aria-labelledby="confirmed-title">
      <div className="co-done">
        <span className="co-done__stamp" aria-hidden="true">
          {method?.flow === "offline" ? "Received" : "Paid"}
        </span>
        <h1 id="confirmed-title" className="big-sans m-0 mt-10 text-[clamp(2.2rem,5vw,5.5rem)]">
          Thank you, <span className="em-italic text-gold-400">reader</span>
        </h1>
        {method && (
          <p className="small-upper m-0 mt-6 text-gold-300">
            {method.flow === "offline" ? "Pay the courier on delivery" : `Paid with ${method.name}`}
          </p>
        )}
        {placed ? (
          <>
            {/* The copies are ordered, so they leave the basket */}
            <ClearBasket />
            <p className="m-0 mt-8 font-serif text-[clamp(1.6rem,3vw,2.6rem)] tabular-nums">Order {code}</p>
            <p className="m-0 mt-4 max-w-[44ch] text-[clamp(17px,1.3vw,21px)] leading-relaxed text-parchment-200">
              {signedIn
                ? "It is on your orders page now, where you can follow it until it reaches you."
                : "Keep this number with the email you gave: you will need both to ask about your order."}
            </p>
          </>
        ) : (
          <p className="m-0 mt-4 max-w-[44ch] text-[clamp(17px,1.3vw,21px)] leading-relaxed text-parchment-200">
            This is a demo checkout: no order was placed and nothing was charged.
          </p>
        )}

        <div className="mt-10 flex flex-wrap justify-center gap-x-10 gap-y-6">
          <DotButton href="/catalogue" variant="solid">
            Back to the shelves
          </DotButton>
          {placed ? (
            <DotButton href="/orders">{signedIn ? "View your orders" : "Find your order"}</DotButton>
          ) : (
            <DotButton href="/cart">View basket</DotButton>
          )}
        </div>
      </div>
    </section>
  );
}
