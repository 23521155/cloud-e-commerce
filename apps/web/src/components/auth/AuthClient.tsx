"use client";

import { useState } from "react";
import { useFormStatus } from "react-dom";

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
