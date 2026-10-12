"use client";

import { createContext, startTransition, useActionState, useContext, useEffect, useRef, useState } from "react";
import { useFormStatus } from "react-dom";
import type { CheckoutState } from "@/app/(shop)/checkout/actions";

/** True while CheckoutForm is placing the order. */
const PlacingContext = createContext(false);

/**
 * The order form. Submitted from onSubmit rather than through the form's action: React resets
 * a form once its action has run, and an order that was refused must keep what the reader typed.
 * The action stays on the form so it still posts before the page has hydrated.
 */
export function CheckoutForm({
  action,
  className,
  children,
}: {
  action: (state: CheckoutState, formData: FormData) => Promise<CheckoutState>;
  className?: string;
  children: React.ReactNode;
}) {
  const [state, formAction, pending] = useActionState(action, {});
  // One idempotency key per checkout: placing the order twice returns the first order
  const key = useRef<string | null>(null);
  const alert = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (state.error) alert.current?.scrollIntoView({ behavior: "smooth", block: "center" });
  }, [state]);

  const onSubmit = (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const data = new FormData(e.currentTarget);
    key.current ??= crypto.randomUUID();
    data.set("idempotencyKey", key.current);
    startTransition(() => formAction(data));
  };

  return (
    <form action={formAction} onSubmit={onSubmit} className={className}>
      {state.error && (
        <div ref={alert} role="alert" className="mx-auto w-full max-w-[760px] border border-dashed border-oxblood-700 px-6 py-5 text-parchment-200">
          <p className="m-0">{state.error}</p>
          {state.unavailable && (
            <ul className="m-0 mt-3 pl-5 italic">
              {state.unavailable.map((title) => (
                <li key={title}>{title}</li>
              ))}
            </ul>
          )}
        </div>
      )}
      <PlacingContext value={pending}>{children}</PlacingContext>
    </form>
  );
}

const groupDigits = (v: string, max: number) =>
  v
    .replace(/\D/g, "")
    .slice(0, max)
    .replace(/(\d{4})(?=\d)/g, "$1 ");

/**
 * Card form. Stand-in for the gateway's hosted card fields (an iframe in production, so card data
 * never reaches this server). The inputs have no `name`, so nothing typed here is submitted.
 * They are disabled unless "Card" is the chosen method, so hidden fields never block the order.
 */
export function CardFields() {
  const root = useRef<HTMLDivElement>(null);
  const [active, setActive] = useState(false);
  const [number, setNumber] = useState("");
  const [expiry, setExpiry] = useState("");

  useEffect(() => {
    const form = root.current?.closest("form");
    if (!form) return;
    const sync = () => setActive(new FormData(form).get("payment") === "card");
    sync();
    form.addEventListener("change", sync);
    return () => form.removeEventListener("change", sync);
  }, []);

  const onExpiry = (v: string) => {
    const d = v.replace(/\D/g, "").slice(0, 4);
    setExpiry(d.length > 2 ? `${d.slice(0, 2)} / ${d.slice(2)}` : d);
  };

  return (
    <div ref={root} className="co-cardform">
      <label className="co__field co-cardform__number">
        <span className="co__label small-upper">Card number</span>
        <input
          className="co__input tabular-nums"
          inputMode="numeric"
          autoComplete="cc-number"
          placeholder="1234 5678 9012 3456"
          pattern="[0-9 ]{14,23}"
          title="12 to 19 digits"
          value={number}
          onChange={(e) => setNumber(groupDigits(e.target.value, 19))}
          required
          disabled={!active}
        />
      </label>
      <label className="co__field co-cardform__name">
        <span className="co__label small-upper">Name on card</span>
        <input className="co__input" autoComplete="cc-name" required disabled={!active} />
      </label>
      <label className="co__field">
        <span className="co__label small-upper">Expiry</span>
        <input
          className="co__input tabular-nums"
          inputMode="numeric"
          autoComplete="cc-exp"
          placeholder="MM / YY"
          pattern="(0[1-9]|1[0-2]) / [0-9]{2}"
          title="Month and year, e.g. 08 / 28"
          value={expiry}
          onChange={(e) => onExpiry(e.target.value)}
          required
          disabled={!active}
        />
      </label>
      <label className="co__field">
        <span className="co__label small-upper">CVC</span>
        <input className="co__input tabular-nums" inputMode="numeric" autoComplete="cc-csc" placeholder="123" pattern="[0-9]{3,4}" title="3 or 4 digits on the back" maxLength={4} required disabled={!active} />
      </label>
      <p className="co-cardform__note">Demo fields: what you type here is never sent anywhere. The live site uses the payment provider’s secure fields in this spot.</p>
    </div>
  );
}

/** Submit button for the order form; shows progress while the server action runs. */
export function PlaceOrderButton({ className = "" }: { className?: string }) {
  // useFormStatus covers the post before hydration; CheckoutForm submits without it afterwards
  const placing = useContext(PlacingContext);
  const pending = useFormStatus().pending || placing;
  return (
    <button type="submit" className={`dot-btn dot-btn--solid co__submit ${className}`.trim()} disabled={pending} aria-disabled={pending}>
      {pending ? "Placing order…" : "Place order"}
      <span className="dot-btn__dot" aria-hidden="true">
        ❧
      </span>
    </button>
  );
}

type Step = { key: string; label: string; content: React.ReactNode };

/**
 * One fieldset at a time inside a single form. Hidden steps stay in the DOM so their values
 * are submitted; "Continue" validates only the visible step.
 */
export function Stepper({ steps, last }: { steps: Step[]; last: React.ReactNode }) {
  const [current, setCurrent] = useState(0);
  const panels = useRef<(HTMLDivElement | null)[]>([]);

  const go = (to: number) => {
    setCurrent(to);
    requestAnimationFrame(() => panels.current[to]?.querySelector<HTMLElement>("input, select")?.focus());
  };

  const next = () => {
    const fields = panels.current[current]?.querySelectorAll<HTMLInputElement | HTMLSelectElement>("input, select") ?? [];
    for (const f of fields) {
      if (!f.checkValidity()) {
        f.reportValidity();
        return;
      }
    }
    go(current + 1);
  };

  return (
    <div className="co-steps">
      <ol className="co-steps__rail">
        {steps.map((s, i) => (
          <li key={s.key}>
            <button
              type="button"
              className={`co-steps__tab ${i === current ? "is-current" : ""} ${i < current ? "is-done" : ""}`}
              aria-current={i === current ? "step" : undefined}
              disabled={i > current}
              onClick={() => go(i)}
            >
              <span className="co-steps__num">{i + 1}</span>
              {s.label}
            </button>
          </li>
        ))}
      </ol>

      {steps.map((s, i) => (
        <div key={s.key} ref={(el) => void (panels.current[i] = el)} hidden={i !== current}>
          {s.content}
        </div>
      ))}

      <div className="co-steps__nav">
        {current > 0 ? (
          <button type="button" className="small-upper under-hover co-steps__back" onClick={() => go(current - 1)}>
            Back
          </button>
        ) : (
          <span />
        )}
        {current < steps.length - 1 ? (
          <button type="button" className="dot-btn co-steps__next" onClick={next}>
            Continue
            <span className="dot-btn__dot" aria-hidden="true">
              ❧
            </span>
          </button>
        ) : (
          last
        )}
      </div>
    </div>
  );
}
