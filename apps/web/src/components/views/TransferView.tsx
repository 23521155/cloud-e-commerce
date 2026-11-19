"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { DotButton } from "@/components/ui/DotButton";
import type { TransferInstructions } from "@/lib/order-api";

const price = new Intl.NumberFormat("en-US", { style: "currency", currency: "VND", maximumFractionDigits: 0 });

/** Seconds between two checks of the payment. */
const CHECK_SECONDS = 4;

type State = "waiting" | "paid" | "expired" | "none";

const clock = (ms: number) => {
  const s = Math.max(0, Math.ceil(ms / 1000));
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;
};

/** The bank transfer page of a VietQR order: the QR code, the same details by hand, and a wait for the money. */
export function TransferView({ code, expiresAt, transfer }: { code: string; expiresAt: string; transfer: TransferInstructions }) {
  const router = useRouter();
  const [state, setState] = useState<State>("waiting");
  const [now, setNow] = useState(() => Date.now());

  const deadline = Date.parse(expiresAt);
  const expired = state === "expired" || now >= deadline;

  // The countdown
  useEffect(() => {
    if (expired) return;
    const tick = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(tick);
  }, [expired]);

  // Asks whether the money has arrived, until it has or the time to pay runs out
  useEffect(() => {
    if (state === "paid") return;

    const check = async () => {
      try {
        const res = await fetch("/api/checkout/transfer-status", { cache: "no-store" });
        if (!res.ok) return;
        const body = (await res.json()) as { state: State };
        setState(body.state);
        if (body.state === "paid") router.replace(`/checkout/confirmed?method=vietqr&code=${encodeURIComponent(code)}`);
      } catch {
        // The next check tries again
      }
    };

    const poll = setInterval(check, CHECK_SECONDS * 1000);
    void check();
    return () => clearInterval(poll);
  }, [state, code, router]);

  return (
    <section className="work basket" aria-labelledby="transfer-title">
      <div className="co-pay">
        <div className="co-pay__card">
          <p className="small-upper m-0 text-oxblood-700">Bank transfer (VietQR)</p>
          <h1 id="transfer-title" className="co-pay__title">
            Scan to pay
          </h1>
          <p className="co-pay__amount m-0 tabular-nums">{price.format(transfer.amountVnd)}</p>

          {!expired && (
            // Remote image from the QR service, so a plain <img> until a domain is configured for next/image
            // eslint-disable-next-line @next/next/no-img-element
            <img src={transfer.imageUrl} alt={`VietQR code to pay order ${code}`} width={220} height={220} className="h-auto w-[clamp(180px,40vw,220px)]" />
          )}

          <dl className="m-0 grid w-full grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-left text-[15px]">
            <dt className="text-ink-600">Bank</dt>
            <dd className="m-0 text-right">{transfer.bank}</dd>
            <dt className="text-ink-600">Account</dt>
            <dd className="m-0 text-right tabular-nums">{transfer.accountNo}</dd>
            <dt className="text-ink-600">Name</dt>
            <dd className="m-0 text-right">{transfer.accountName}</dd>
            <dt className="text-ink-600">Transfer text</dt>
            <dd className="m-0 text-right font-bold tabular-nums">{transfer.content}</dd>
          </dl>

          <p className="co-pay__text m-0" role="status">
            {expired
              ? "The time to pay has run out and your copies were let go. Nothing was charged: place the order again."
              : `Open your banking app, scan the code and keep the transfer text as it is: it is how we find your order. This page moves on by itself once the money arrives. Time left: ${clock(deadline - now)}.`}
          </p>
        </div>

        <div className="co-pay__actions">
          <DotButton href="/checkout" variant={expired ? "solid" : undefined}>
            {expired ? "Back to checkout" : "Cancel"}
          </DotButton>
        </div>
      </div>
    </section>
  );
}
