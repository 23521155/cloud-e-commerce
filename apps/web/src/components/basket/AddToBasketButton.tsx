"use client";

import { useState } from "react";
import { DotButton } from "@/components/ui/DotButton";
import { MAX_BASKET } from "@/lib/basket";
import { addToBasket, useBasket } from "@/lib/basket-store";

/**
 * Puts one copy in the guest basket (this browser's localStorage). Once it is there the button
 * becomes a link to the basket. Looks like DotButton; `variant` matches its look.
 */
export function AddToBasketButton({ slug, title, variant = "solid", className = "" }: { slug: string; title: string; variant?: "outline" | "solid"; className?: string }) {
  const basket = useBasket();
  const [full, setFull] = useState(false);

  if (basket?.includes(slug)) {
    return (
      <DotButton href="/cart" variant={variant} className={className}>
        In your basket
      </DotButton>
    );
  }

  return (
    <>
      <button
        type="button"
        className={`dot-btn ${variant === "solid" ? "dot-btn--solid" : ""} cursor-pointer border-0 bg-transparent ${className}`.trim()}
        aria-label={`Add ${title} to basket`}
        onClick={() => setFull(addToBasket(slug) === "full")}
      >
        Add to basket
        <span className="dot-btn__dot" aria-hidden="true">
          ❧
        </span>
      </button>
      {full && (
        <p role="status" className="m-0 basis-full text-parchment-200">
          Your basket holds {MAX_BASKET} copies at most. Check out or remove one first.
        </p>
      )}
    </>
  );
}
