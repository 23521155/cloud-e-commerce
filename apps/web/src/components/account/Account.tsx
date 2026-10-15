import { PasswordField, SubmitButton } from "@/components/auth/AuthClient";
import { changePassword, resendVerification, updateProfile } from "@/lib/account-actions";

export type Profile = {
  name: string;
  email: string;
  phone: string;
  address: string;
  city: string;
  postal: string;
  country: string;
};

export type SavedNotice = "profile" | "password" | "verify-sent";

/** Why the ledger page or the password slip was not saved; nothing changed. */
export type AccountError = keyof typeof ACCOUNT_ERROR;

const ACCOUNT_ERROR = {
  "profile-invalid": "Not saved: one of the details is too long. Please shorten it and try again.",
  "profile-failed": "Not saved: the shop could not be reached. Please try again in a moment.",
  "password-wrong": "Not changed: the current password is not right.",
  "password-invalid": "Not changed: the new password must be 8 to 128 characters.",
  "password-mismatch": "Not changed: the new password and the repeat do not match.",
  "password-failed": "Not changed: the shop could not be reached. Please try again in a moment.",
  "verify-too-soon": "A link went out a moment ago. Check your inbox, or ask again in a minute.",
  "verify-failed": "The link could not be sent right now. Please try again in a moment.",
} as const;

export const isAccountError = (value: unknown): value is AccountError => typeof value === "string" && value in ACCOUNT_ERROR;

function ErrorNotice({ error, prefix }: { error?: AccountError; prefix: "profile" | "password" | "verify" }) {
  if (!error?.startsWith(prefix)) return null;
  return (
    <p className="acc-notice m-0" role="alert">
      {ACCOUNT_ERROR[error]}
    </p>
  );
}

const COUNTRIES = ["Vietnam", "United States", "United Kingdom", "Australia", "Canada", "Japan", "Singapore", "Other country"];

function Notice({ show, children }: { show: boolean; children: React.ReactNode }) {
  if (!show) return null;
  return (
    <p className="acc-notice m-0" role="status">
      {children}
    </p>
  );
}

/** An entry on a ruled ledger line: small caption above, the value written in on the rule. */
function Entry({ label, name, className = "", ...input }: { label: string; name: string; className?: string } & React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <label className={`ledger__entry ${className}`}>
      <span className="ledger__caption">{label}</span>
      <input name={name} className="ledger__input" {...input} />
    </label>
  );
}

/** Account settings as an open ledger: details across two pages, the password on a loose slip below. */
export function Account({ profile, emailVerified, saved, error }: { profile: Profile; emailVerified: boolean; saved?: SavedNotice; error?: AccountError }) {
  return (
    <div className="ledger">
      <form action={updateProfile} className="ledger__form">
        {/* Wide screens draw the pages with the open-book photo, whose headings are printed in; the h2s stay for screen readers */}
        <div className="ledger__book">
          <div className="ledger__page ledger__page--left">
            <h2 className="ledger__title">
              Particulars <em>of the reader</em>
            </h2>
            <Entry label="Full name" name="name" autoComplete="name" required maxLength={100} defaultValue={profile.name} />
            {/* Changing the email needs the new address confirmed, which the shop cannot do yet */}
            <Entry label="Email" name="email" type="email" autoComplete="email" readOnly defaultValue={profile.email} />
            {!emailVerified && (
              <p className="ledger__margin">
                Not confirmed yet: follow the link we emailed you.{" "}
                {/* A second submit button of the same form: the ledger page cannot hold a nested form */}
                <button type="submit" formAction={resendVerification} formNoValidate className="ledger__to-slip under-hover">
                  Send the link again
                </button>
              </p>
            )}
            <Notice show={saved === "verify-sent"}>A new link is on its way to {profile.email}.</Notice>
            <ErrorNotice error={error} prefix="verify" />
            <Entry label="Phone" name="phone" type="tel" autoComplete="tel" maxLength={20} defaultValue={profile.phone} placeholder="only the courier sees it" />
            <p className="ledger__margin">Kept by the shop, never shown to other readers.</p>
            {/* The password slip sits below the book, out of the first screen; point to it from the page */}
            <a href="#password-slip" className="ledger__to-slip under-hover">
              Changing your password? It is on the loose slip below ↓
            </a>
          </div>
          <div className="ledger__page ledger__page--right">
            <h2 className="ledger__title">
              Where parcels <em>are sent</em>
            </h2>
            <Entry label="Street address" name="address" autoComplete="street-address" maxLength={300} defaultValue={profile.address} />
            <div className="ledger__pair">
              <Entry label="City" name="city" autoComplete="address-level2" maxLength={100} defaultValue={profile.city} />
              <Entry label="Post code" name="postal" autoComplete="postal-code" maxLength={20} defaultValue={profile.postal} />
            </div>
            <label className="ledger__entry">
              <span className="ledger__caption">Country</span>
              <select name="country" className="ledger__input co__select" autoComplete="country-name" defaultValue={profile.country}>
                {COUNTRIES.map((c) => (
                  <option key={c}>{c}</option>
                ))}
              </select>
            </label>
            <div className="ledger__foot">
              <SubmitButton pending="Saving…">Save the page</SubmitButton>
              <Notice show={saved === "profile"}>Saved.</Notice>
              <ErrorNotice error={error} prefix="profile" />
            </div>
          </div>
        </div>
      </form>

      <section className="ledger__slip" aria-labelledby="ledger-pass">
        {/* Anchor at the slip's middle: scrolling to it centres the slip on screen */}
        <span id="password-slip" className="ledger__slip-anchor" aria-hidden="true" />
        <h2 id="ledger-pass" className="ledger__slip-head">
          A loose slip: <em>your password</em>
        </h2>
        <Notice show={saved === "password"}>New password set. Any other device you were signed in on has been signed out.</Notice>
        <ErrorNotice error={error} prefix="password" />
        <form action={changePassword} className="ledger__slip-form">
          <PasswordField label="Current" name="current" autoComplete="current-password" />
          <PasswordField label="New" name="password" autoComplete="new-password" minLength={8} hint="At least 8 characters." />
          <PasswordField label="Again" name="confirm" autoComplete="new-password" match="password" />
          <SubmitButton pending="Changing…">Change password</SubmitButton>
        </form>
      </section>
    </div>
  );
}
