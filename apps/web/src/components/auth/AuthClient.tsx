"use client";

import Link from "next/link";
import { useActionState, useState } from "react";
import { useFormStatus } from "react-dom";
import { register, signIn, verifyCode, type AuthFormState } from "@/lib/auth-actions";

/** Text input on the index card; required unless told otherwise. */
export function TextField({ label, ...input }: { label: string } & React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <label className="auth__field">
      <span className="co__label small-upper">{label}</span>
      <input className="co__input" required {...input} />
    </label>
  );
}

/** Primary submit: rough rust brush button with a pending label while the server action runs. */
export function SubmitButton({ children, pending: pendingLabel }: { children: React.ReactNode; pending: string }) {
  const { pending } = useFormStatus();
  return (
    <button type="submit" className="dot-btn dot-btn--solid auth__submit" disabled={pending} aria-disabled={pending}>
      {pending ? pendingLabel : children}
      <span className="dot-btn__dot" aria-hidden="true">
        ❧
      </span>
    </button>
  );
}

type PasswordProps = {
  label: string;
  name: string;
  autoComplete: "current-password" | "new-password";
  hint?: string;
  /** Name of another password input in the same form this one must equal */
  match?: string;
  minLength?: number;
  aside?: React.ReactNode;
};

/** Password input with a show/hide toggle; optionally checks it equals another field. */
export function PasswordField({ label, name, autoComplete, hint, match, minLength, aside }: PasswordProps) {
  const [shown, setShown] = useState(false);

  const checkMatch = (input: HTMLInputElement) => {
    if (!match) return;
    const other = input.form?.elements.namedItem(match) as HTMLInputElement | null;
    input.setCustomValidity(other && other.value !== input.value ? "Passwords do not match." : "");
  };

  return (
    <label className="auth__field">
      <span className="auth__label-row">
        <span className="co__label small-upper">{label}</span>
        {aside}
      </span>
      <span className="auth__password">
        <input
          name={name}
          type={shown ? "text" : "password"}
          className="co__input"
          autoComplete={autoComplete}
          minLength={minLength}
          required
          onInput={(e) => checkMatch(e.currentTarget)}
          onBlur={(e) => checkMatch(e.currentTarget)}
        />
        <button type="button" className="auth__reveal small-upper" onClick={() => setShown((s) => !s)} aria-pressed={shown} aria-label={shown ? `Hide ${label.toLowerCase()}` : `Show ${label.toLowerCase()}`}>
          {shown ? "Hide" : "Show"}
        </button>
      </span>
      {hint && <span className="co__hint">{hint}</span>}
    </label>
  );
}

function FormError({ error }: { error?: string }) {
  if (!error) return null;
  return (
    <p role="alert" className="auth__error m-0">
      {error}
    </p>
  );
}

/**
 * Sign-in form: a refused attempt shows why and keeps the email. next is where to go once signed in;
 * notice is a line above the form (after registering, or after a password reset).
 */
export function SignInForm({ notice, next }: { notice?: string; next?: string }) {
  const [state, formAction] = useActionState<AuthFormState, FormData>(signIn, {});
  return (
    <form action={formAction} className="auth__form">
      {notice && !state.error && (
        <p role="status" className="auth__sent m-0">
          {notice}
        </p>
      )}
      <FormError error={state.error} />
      {next && <input type="hidden" name="next" value={next} />}
      <TextField label="Email" name="email" type="email" autoComplete="email" placeholder="you@example.com" defaultValue={state.email} />
      <PasswordField
        label="Password"
        name="password"
        autoComplete="current-password"
        aside={
          <Link href="/forgot-password" className="auth__aside under-hover">
            Forgot?
          </Link>
        }
      />
      <SubmitButton pending="Signing in…">Sign in</SubmitButton>
    </form>
  );
}

/** Register form: a refused attempt shows why and keeps the name and email, never the passwords. */
export function RegisterForm() {
  const [state, formAction] = useActionState<AuthFormState, FormData>(register, {});
  return (
    <form action={formAction} className="auth__form">
      <FormError error={state.error} />
      <TextField label="Name" name="name" autoComplete="name" defaultValue={state.name} />
      <TextField label="Email" name="email" type="email" autoComplete="email" placeholder="you@example.com" defaultValue={state.email} />
      <PasswordField label="Password" name="password" autoComplete="new-password" minLength={8} hint="At least 8 characters." />
      <PasswordField label="Confirm password" name="confirm" autoComplete="new-password" match="password" />
      <SubmitButton pending="Creating account…">Create account</SubmitButton>
    </form>
  );
}

/** The code step of a sign-in: six digits from the reader's authenticator app. */
export function CodeForm({ next }: { next?: string }) {
  const [state, formAction] = useActionState<AuthFormState, FormData>(verifyCode, {});
  return (
    <form action={formAction} className="auth__form">
      <FormError error={state.error} />
      {next && <input type="hidden" name="next" value={next} />}
      <TextField label="Code from your authenticator app" name="code" autoComplete="one-time-code" maxLength={16} placeholder="123456" autoFocus />
      <p className="co__hint m-0">Lost your phone? Type one of your recovery codes instead.</p>
      <SubmitButton pending="Checking…">Sign in</SubmitButton>
    </form>
  );
}
