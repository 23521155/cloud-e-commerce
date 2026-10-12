import { BasketSkeleton } from "@/components/basket/BasketSkeleton";

/** Shown while the server renders the checkout; CheckoutView keeps the same card until the basket is read. */
export default function Loading() {
  return (
    <section className="work basket" aria-label="Checkout">
      <BasketSkeleton />
    </section>
  );
}
