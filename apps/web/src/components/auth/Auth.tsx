import Link from "next/link";
import { CodeForm, PasswordField, RegisterForm, SignInForm, SubmitButton, TextField } from "@/components/auth/AuthClient";
import { BookCover } from "@/components/ui/BookArt";
import { verifyEmail } from "@/lib/account-actions";
import { requestReset, resetPassword } from "@/lib/auth-actions";

export type AuthMode = "sign-in" | "register" | "forgot" | "verify" | "reset" | "code";

/** The reset page's state: the token from the link, and why the last try was refused. */
export type ResetState = { token?: string; error?: "invalid-link" | "weak" | "mismatch" | "failed" };

const RESET_ERROR: Record<NonNullable<ResetState["error"]>, string> = {
  "invalid-link": "This link has already been used or has expired. Ask for a new one below.",
  weak: "The new password must be 8 to 128 characters.",
  mismatch: "The new password and the repeat do not match.",
  failed: "The password could not be changed right now. Please try again in a moment.",
};

/** The confirmation page's state: the token from the link, or how confirming went. */
export type VerifyState = { token?: string; done?: boolean; error?: "invalid" | "failed" };

/** Sends the reader to Google (GET /auth/google); a new reader gets an account on the way back. */
function GoogleButton({ label, next }: { label: string; next?: string }) {
  return (
    <form action="/auth/google" method="get">
      {next && <input type="hidden" name="next" value={next} />}
      <button type="submit" className="auth__google">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <path fill="#4285F4" d="M23.5 12.3c0-.8-.1-1.6-.2-2.3H12v4.4h6.5a5.6 5.6 0 0 1-2.4 3.6v3h3.9c2.2-2.1 3.5-5.1 3.5-8.7z" />
          <path fill="#34A853" d="M12 24c3.2 0 6-1.1 7.9-2.9l-3.9-3c-1.1.7-2.4 1.2-4 1.2-3.1 0-5.7-2.1-6.6-4.9h-4v3.1A12 12 0 0 0 12 24z" />
          <path fill="#FBBC05" d="M5.4 14.4a7.2 7.2 0 0 1 0-4.7V6.6h-4a12 12 0 0 0 0 10.8z" />
          <path fill="#EA4335" d="M12 4.8c1.7 0 3.3.6 4.5 1.8l3.4-3.4A12 12 0 0 0 1.4 6.6l4 3.1C6.3 6.9 8.9 4.8 12 4.8z" />
        </svg>
        {label}
      </button>
    </form>
  );
}

function Divider() {
  return (
    <p className="auth__or small-upper" aria-hidden="true">
      <span>or</span>
    </p>
  );
}

function SignIn({ notice, next }: { notice?: string; next?: string }) {
  return (
    <>
      <SignInForm notice={notice} next={next} />
      <Divider />
      <GoogleButton label="Continue with Google" next={next} />
    </>
  );
}

function Register() {
  return (
    <>
      <RegisterForm />
      <Divider />
      <GoogleButton label="Sign up with Google" />
    </>
  );
}

function ForgotForm({ sent }: { sent: boolean }) {
  if (sent) {
    return (
      <div className="auth__sent" role="status">
        <p className="m-0">If an account exists for that address, a link to reset the password is on its way.</p>
        <p className="m-0 mt-3 italic">Nothing arrived? Check the spam folder, or try again in a few minutes.</p>
      </div>
    );
  }
  return (
    <form action={requestReset} className="auth__form">
      <TextField label="Email" name="email" type="email" autoComplete="email" placeholder="you@example.com" />
      <SubmitButton pending="Sending…">Send reset link</SubmitButton>
    </form>
  );
}

function ResetForm({ token, error }: ResetState) {
  const refused = error && (
    <p role="alert" className="auth__error m-0">
      {RESET_ERROR[error]}
    </p>
  );
  if (!token) {
    return (
      <div className="auth__form">
        {refused ?? (
          <p role="alert" className="auth__error m-0">
            This page needs the link from the email. Please open it from there.
          </p>
        )}
        <Link href="/forgot-password" className="auth__aside under-hover">
          Send me a new link
        </Link>
      </div>
    );
  }
  return (
    <form action={resetPassword} className="auth__form">
      {refused}
      <input type="hidden" name="token" value={token} />
      <PasswordField label="New password" name="password" autoComplete="new-password" minLength={8} hint="At least 8 characters." />
      <PasswordField label="Confirm password" name="confirm" autoComplete="new-password" match="password" />
      <SubmitButton pending="Saving…">Set the new password</SubmitButton>
    </form>
  );
}

function VerifyForm({ token, done, error }: VerifyState) {
  if (done) {
    return (
      <div className="auth__sent" role="status">
        <p className="m-0">Thank you: your email address is confirmed.</p>
      </div>
    );
  }
  if (error) {
    return (
      <p role="alert" className="auth__error m-0">
        {error === "invalid"
          ? "This link has already been used or has expired. You can ask for a new one on your account page."
          : "The address could not be confirmed right now. Please open the link again in a moment."}
      </p>
    );
  }
  if (!token) {
    return (
      <p role="alert" className="auth__error m-0">
        This page needs the link from the email. Please open it from there.
      </p>
    );
  }
  return (
    <form action={verifyEmail} className="auth__form">
      <input type="hidden" name="token" value={token} />
      <SubmitButton pending="Confirming…">Confirm my email</SubmitButton>
    </form>
  );
}

const COPY: Record<AuthMode, { card: string; title: [string, string]; intro: string }> = {
  "sign-in": { card: "Reader's card", title: ["Welcome", "back"], intro: "Sign in to see your basket and orders." },
  register: { card: "New reader", title: ["Open an", "account"], intro: "Keep a basket across devices and follow your orders." },
  forgot: { card: "Lost card", title: ["Reset your", "password"], intro: "Enter the email you signed up with and we will send a reset link." },
  verify: { card: "Reader's card", title: ["Confirm your", "email"], intro: "One click, and the shop knows this address is yours." },
  reset: { card: "Lost card", title: ["Choose a new", "password"], intro: "Every device you were signed in on will be signed out." },
  code: { card: "Reader's card", title: ["One more", "step"], intro: "Open your authenticator app and type the code it shows for Marginalleya." },
};

const SWITCH: Record<AuthMode, { text: string; link: string; href: string }> = {
  "sign-in": { text: "New here?", link: "Create an account", href: "/register" },
  register: { text: "Already have an account?", link: "Sign in", href: "/sign-in" },
  forgot: { text: "Remembered it?", link: "Back to sign in", href: "/sign-in" },
  verify: { text: "Done here?", link: "Go to your account", href: "/account" },
  reset: { text: "Remembered it?", link: "Back to sign in", href: "/sign-in" },
  code: { text: "Not you?", link: "Start again", href: "/sign-in" },
};

const SHELF = [
  { title: "Pride and Prejudice", author: "Jane Austen", color: "#24382e" },
  { title: "Meditations", author: "Marcus Aurelius", color: "#5c1a16" },
  { title: "Jamaica Inn", author: "Daphne du Maurier", color: "#64331e" },
];

/** Every auth screen: heading and a few books on the left, the form on an index card on the right. */
export function AuthPage({
  mode,
  sent = false,
  notice,
  next,
  verify,
  reset,
}: {
  mode: AuthMode;
  sent?: boolean;
  /** A line above the sign-in form. */
  notice?: string;
  next?: string;
  verify?: VerifyState;
  reset?: ResetState;
}) {
  const copy = COPY[mode];
  const sw = SWITCH[mode];
  return (
    <section className="work basket auth" aria-labelledby="auth-title">
      <div className="auth__layout">
        <div className="auth__art">
          <h1 id="auth-title" data-reveal className="big-sans m-0 text-[clamp(2.4rem,5.4vw,5.6rem)]">
            <span className="line">
              <span>{copy.title[0]}</span>
            </span>
            <span className="line">
              <span className="em-italic text-gold-400">{copy.title[1]}</span>
            </span>
          </h1>
          <p className="m-0 mt-5 max-w-[30ch] text-[clamp(17px,1.3vw,21px)] leading-relaxed text-parchment-200">{copy.intro}</p>
          <div className="auth__books" aria-hidden="true">
            {SHELF.map((b, i) => (
              <div key={b.title} className="auth__book" style={{ "--i": i } as React.CSSProperties}>
                <BookCover {...b} />
              </div>
            ))}
          </div>
        </div>

        <section className="co__card auth__card" aria-label={copy.card}>
          <div className="catalogue__card-head small-upper" aria-hidden="true">
            <span>{copy.card}</span>
            <span>Marginalleya</span>
          </div>
          <div className="auth__body">
            {mode === "sign-in" && <SignIn notice={notice} next={next} />}
            {mode === "register" && <Register />}
            {mode === "forgot" && <ForgotForm sent={sent} />}
            {mode === "verify" && <VerifyForm {...verify} />}
            {mode === "reset" && <ResetForm {...reset} />}
            {mode === "code" && <CodeForm next={next} />}
            <p className="auth__switch">
              {sw.text}{" "}
              <Link href={sw.href} className="under-hover ml-1">
                {sw.link}
              </Link>
            </p>
          </div>
        </section>
      </div>
    </section>
  );
}
