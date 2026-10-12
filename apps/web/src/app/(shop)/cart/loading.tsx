import { BasketSkeleton } from "@/components/basket/BasketSkeleton";

/** Shown while the server renders the basket page; CartView keeps the same card until the basket is read. */
export default function Loading() {
  return (
    <section className="work basket" aria-label="Basket">
      <BasketSkeleton />
    </section>
  );
}
