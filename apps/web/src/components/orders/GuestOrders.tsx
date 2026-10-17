import Link from "next/link";
import { SubmitButton, TextField } from "@/components/auth/AuthClient";
import { Orders } from "@/components/orders/Orders";
import { findOrder, forgetOrder, lookedUpOrder } from "@/lib/order-actions";
import { getGuestOrder } from "@/lib/orders";

const ERROR: Record<string, string> = {
  "not-found": "No order matches that number and email. Check both against your receipt.",
  failed: "Your order could not be looked up right now. Please try again in a moment.",
};

/**
 * The orders page for a reader who is not signed in: a form asks for an order's number and email,
 * then that order's card shows in its place.
 */
export async function GuestOrders({ error }: { error?: string }) {
  const lookup = await lookedUpOrder();
  const order = lookup && (await getGuestOrder(lookup.code, lookup.email));

  if (order) {
    return (
      <section className="work basket" aria-labelledby="orders-title">
        <div className="catalogue__head">
          <h1 id="orders-title" data-reveal className="big-sans m-0 text-[clamp(2.4rem,6vw,7rem)]">
            <span className="line">
              <span>Your</span>
            </span>
            <span className="line">
              <span className="em-italic text-gold-400">order</span>
            </span>
          </h1>
          <form action={forgetOrder}>
            <button type="submit" className="small-upper under-hover text-parchment-200">
              Look up another order
            </button>
          </form>
        </div>
        <Orders orders={[order]} />
      </section>
    );
  }

  const message = error ? ERROR[error] : undefined;

  return (
    <section className="work basket auth" aria-labelledby="orders-title">
      <div className="auth__layout">
        <div className="auth__art">
          <h1 id="orders-title" data-reveal className="big-sans m-0 text-[clamp(2.4rem,5.4vw,5.6rem)]">
            <span className="line">
              <span>Find your</span>
            </span>
            <span className="line">
              <span className="em-italic text-gold-400">order</span>
            </span>
          </h1>
          <p className="m-0 mt-5 max-w-[30ch] text-[clamp(17px,1.3vw,21px)] leading-relaxed text-parchment-200">
            Ordered without an account? The order number is on the page you saw after checkout.
          </p>
        </div>

        <section className="co__card auth__card" aria-label="Order slip">
          <div className="catalogue__card-head small-upper" aria-hidden="true">
            <span>Order slip</span>
            <span>Marginalleya</span>
          </div>
          <div className="auth__body">
            <form action={findOrder} className="auth__form">
              {message && (
                <p role="alert" className="auth__error m-0">
                  {message}
                </p>
              )}
              <TextField label="Order number" name="code" autoComplete="off" placeholder="MA-048291" maxLength={9} />
              <TextField label="Email" name="email" type="email" autoComplete="email" placeholder="you@example.com" />
              <SubmitButton pending="Looking…">Find my order</SubmitButton>
            </form>
            <p className="auth__switch">
              Have an account?{" "}
              <Link href="/sign-in?next=/orders" className="under-hover ml-1">
                Sign in to see every order
              </Link>
            </p>
          </div>
        </section>
      </div>
    </section>
  );
}
