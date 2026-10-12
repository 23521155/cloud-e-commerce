"use client";

import { useEffect } from "react";
import { clearBasket } from "@/lib/basket-store";

/** Empties the guest basket once an order has been placed. Renders nothing. */
export function ClearBasket() {
  useEffect(() => clearBasket(), []);
  return null;
}
