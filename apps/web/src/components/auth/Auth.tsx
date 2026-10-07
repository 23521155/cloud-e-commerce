import Link from "next/link";
import { PasswordField, SubmitButton } from "@/components/auth/AuthClient";
import { BookCover } from "@/components/ui/BookArt";
import { continueWithGoogle, register, requestReset, signIn } from "@/lib/auth-actions";

export type AuthMode = "sign-in" | "register" | "forgot";

function TextField({ label, ...input }: { label: string } & React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <label className="auth__field">
      <span className="co__label small-upper">{label}</span>
      <input className="co__input" required {...input} />
    </label>
  );
}

function GoogleButton({ label }: { label: string }) {
  return (
    <form action={continueWithGoogle}>
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

function SignInForm() {
  return (
    <>
      <form action={signIn} className="auth__form">
        <TextField label="Email" name="email" type="email" autoComplete="email" placeholder="you@example.com" />
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
      <Divider />
      <GoogleButton label="Continue with Google" />
    </>
  );
}

function RegisterForm() {
  return (
      <form action={register} className="auth__form">
        <TextField label="Name" name="name" autoComplete="name" />
        <TextField label="Email" name="email" type="email" autoComplete="email" placeholder="you@example.com" />
        <PasswordField label="Password" name="password" autoComplete="new-password" minLength={8} hint="At least 8 characters." />
        <PasswordField label="Confirm password" name="confirm" autoComplete="new-password" match="password" />
        <SubmitButton pending="Creating account…">Create account</SubmitButton>
      </form>
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

const COPY: Record<AuthMode, { card: string; title: [string, string]; intro: string }> = {
  "sign-in": { card: "Reader's card", title: ["Welcome", "back"], intro: "Sign in to see your basket and orders." },
  register: { card: "New reader", title: ["Open an", "account"], intro: "Keep a basket across devices and follow your orders." },
  forgot: { card: "Lost card", title: ["Reset your", "password"], intro: "Enter the email you signed up with and we will send a reset link." },
};

const SWITCH: Record<AuthMode, { text: string; link: string; href: string }> = {
  "sign-in": { text: "New here?", link: "Create an account", href: "/register" },
  register: { text: "Already have an account?", link: "Sign in", href: "/sign-in" },
  forgot: { text: "Remembered it?", link: "Back to sign in", href: "/sign-in" },
};

const SHELF = [
  { title: "Pride and Prejudice", author: "Jane Austen", color: "#24382e" },
  { title: "Meditations", author: "Marcus Aurelius", color: "#5c1a16" },
  { title: "Jamaica Inn", author: "Daphne du Maurier", color: "#64331e" },
];

/** Every auth screen: heading and a few books on the left, the form on an index card on the right. */
export function AuthPage({ mode, sent = false }: { mode: AuthMode; sent?: boolean }) {
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
            {mode === "sign-in" && <SignInForm />}
            {mode === "register" && <RegisterForm />}
            {mode === "forgot" && <ForgotForm sent={sent} />}
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
