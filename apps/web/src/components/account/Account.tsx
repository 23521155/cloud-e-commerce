import { PasswordField, SubmitButton } from "@/components/auth/AuthClient";
import { changePassword, updateProfile } from "@/lib/account-actions";

export type Profile = {
  name: string;
  email: string;
  phone: string;
  address: string;
  city: string;
  postal: string;
  country: string;
};

export type SavedNotice = "profile" | "password";

/** Why the ledger page was not saved; the details stay as they were. */
export type ProfileError = "profile-invalid" | "profile-failed";

const PROFILE_ERROR: Record<ProfileError, string> = {
  "profile-invalid": "Not saved: one of the details is too long. Please shorten it and try again.",
  "profile-failed": "Not saved: the shop could not be reached. Please try again in a moment.",
};

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
export function Account({ profile, saved, error }: { profile: Profile; saved?: SavedNotice; error?: ProfileError }) {
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
              {error && (
                <p className="acc-notice m-0" role="alert">
                  {PROFILE_ERROR[error]}
                </p>
              )}
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
        <Notice show={saved === "password"}>New password set.</Notice>
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
