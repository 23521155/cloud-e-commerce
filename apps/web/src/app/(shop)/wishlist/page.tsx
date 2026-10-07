import type { Metadata } from "next";
import { MotionController } from "@/components/motion/MotionController";
import { Wishlist } from "@/components/wishlist/Wishlist";
import { getWishlist } from "@/lib/wishlist";

export const metadata: Metadata = {
  title: "Wishlist — Marginalleya",
  description: "Copies you have saved to come back to.",
};

const copies = (n: number) => `${n} ${n === 1 ? "copy" : "copies"}`;

export default function WishlistPage() {
  const items = getWishlist();
  const available = items.filter((i) => !i.sold).length;

  return (
    <>
      <MotionController />

      <section className="work catalogue" aria-labelledby="wishlist-title">
        <div className="catalogue__head">
          <h1 id="wishlist-title" data-reveal className="big-sans m-0 text-[clamp(2.4rem,6vw,7rem)]">
            <span className="line">
              <span>Your</span>
            </span>
            <span className="line">
              <span className="em-italic text-gold-400">wishlist</span>
            </span>
          </h1>
          <p className="m-0 max-w-[34ch] text-[clamp(17px,1.3vw,21px)] leading-relaxed text-parchment-200">
            {items.length === 0
              ? "Copies you save will wait here."
              : `${copies(items.length)} saved, ${available} still on the shelf. Each is the only one, so it can sell before you come back.`}
          </p>
        </div>

        <Wishlist items={items} />
      </section>
    </>
  );
}
