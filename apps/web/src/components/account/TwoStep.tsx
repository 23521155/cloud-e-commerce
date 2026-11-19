"use client";

/* eslint-disable @next/next/no-img-element -- the QR code is a data: URL made per request */
import Link from "next/link";
import { useActionState } from "react";
import { SubmitButton } from "@/components/auth/AuthClient";
import { confirmTotpSetup, startTotpSetup, type TotpSetupState } from "@/lib/account-actions";

/** Turning two-step sign-in on: scan a QR code, then prove it with a first code. */
export function TwoStepSetup() {
  const [started, start] = useActionState<TotpSetupState>(startTotpSetup, {});
  const [confirmed, confirm] = useActionState<TotpSetupState, FormData>(confirmTotpSetup, {});

  if (confirmed.recoveryCodes) {
    return (
      <div className="two-step__steps">
        <p className="m-0">
          Two-step sign-in is on. If you lose your phone, each of these codes signs you in once instead of the app. Keep them somewhere safe: they
          are not shown again.
        </p>
        <ul className="two-step__codes">
          {confirmed.recoveryCodes.map((c) => (
            <li key={c}>
              <code>{c}</code>
            </li>
          ))}
        </ul>
        <Link href="/account?saved=mfa-on#two-step" className="ledger__to-slip under-hover">
          I have kept them
        </Link>
      </div>
    );
  }

  // The confirm form keeps the QR code in its own state once a code was tried.
  const setup = confirmed.qr ? confirmed : started;

  if (!setup.qr) {
    return (
      <form action={start} className="ledger__slip-form">
        {setup.error && (
          <p className="acc-notice m-0" role="alert">
            {setup.error}
          </p>
        )}
        <p className="m-0">Sign-in will also ask for a code from an authenticator app on your phone, such as Google or Microsoft Authenticator.</p>
        <SubmitButton pending="Starting…">Set up two-step sign-in</SubmitButton>
      </form>
    );
  }

  return (
    <form action={confirm} className="two-step">
      <img src={setup.qr} alt="QR code to add Marginalleya to your authenticator app" width={220} height={220} className="two-step__qr" />
      <div className="two-step__steps">
        <p className="m-0">1. In your authenticator app, add an account and scan this code.</p>
        <p className="m-0">
          Cannot scan? Type this key instead: <code className="two-step__secret">{setup.secret}</code>
        </p>
        <p className="m-0">2. Type the six-digit code the app now shows.</p>
        {confirmed.error && (
          <p className="acc-notice m-0" role="alert">
            {confirmed.error}
          </p>
        )}
        <input type="hidden" name="qr" value={setup.qr} />
        <input type="hidden" name="secret" value={setup.secret} />
        <label className="auth__field">
          <span className="co__label small-upper">Code</span>
          <input name="code" className="co__input" inputMode="numeric" autoComplete="one-time-code" pattern="\s*\d{3}\s*\d{3}\s*" maxLength={7} required />
        </label>
        <SubmitButton pending="Checking…">Turn it on</SubmitButton>
      </div>
    </form>
  );
}
