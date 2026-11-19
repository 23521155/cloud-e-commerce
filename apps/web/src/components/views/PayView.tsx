"use client";

import { SampleQr } from "@/components/checkout/PaymentDetail";
import { DotButton } from "@/components/ui/DotButton";
import { subtotal } from "@/lib/basket";
import { useBasketLines } from "@/lib/basket-store";
import type { PaymentMethod } from "@/lib/payment";

const price = new Intl.NumberFormat("en-US", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

export type PayStep = { provider: string; title: string; text: string; qr: boolean };

/** The mock provider page. The amount is the basket's, which lives in this browser. */
export function PayView({ method, step }: { method: PaymentMethod; step: PayStep }) {
  const { lines } = useBasketLines();
  const total = lines && subtotal(lines.filter((l) => !l.sold));

  return (
    <section className="work basket" aria-labelledby="pay-title">
      <div className="co-pay">
        <p className="co-pay__demo small-upper m-0">Demo payment page · stands in for {step.provider}</p>

        <div className="co-pay__card">
          <p className="small-upper m-0 text-oxblood-700">{method.name}</p>
          <h1 id="pay-title" className="co-pay__title">
            {step.title}
          </h1>
          <p className="co-pay__amount m-0 tabular-nums">{total === null ? "—" : price.format(total)}</p>

          {step.qr && total !== null && (
            <div className="co-pay__qr">
              <SampleQr seed={total + method.id.length} />
            </div>
          )}
          <p className="co-pay__text m-0">{step.text}</p>
        </div>

        <div className="co-pay__actions">
          <DotButton href={`/checkout/confirmed?method=${method.id}`} variant="solid">
            Simulate payment
          </DotButton>
          <DotButton href="/checkout">Cancel</DotButton>
        </div>
      </div>
    </section>
  );
}
