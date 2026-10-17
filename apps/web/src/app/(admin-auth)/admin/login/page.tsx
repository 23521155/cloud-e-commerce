import type { Metadata } from "next";
import Link from "next/link";
import { redirect } from "next/navigation";
import { PasswordField, SubmitButton } from "@/components/auth/AuthClient";
import { Logo } from "@/components/ui/Logo";
import { staffSignIn } from "@/lib/auth-actions";
import { getSession, isStaffWithCode } from "@/lib/session";

// Lives outside the (admin) group so it gets no admin sidebar; the URL is still /admin/login.
export const metadata: Metadata = {
  title: "Staff sign in — Marginalleya",
  robots: { index: false },
};

const ERRORS = {
  wrong: "The email or password is incorrect.",
  "not-staff": "This is not a staff account. Readers sign in on the shop.",
  "needs-two-step": "Staff must use two-step sign-in. Turn it on on your account page, then sign in here again.",
  busy: "Too many attempts. Please wait a minute and try again.",
  failed: "You could not be signed in right now. Please try again.",
} as const;

export default async function AdminLoginPage({ searchParams }: PageProps<"/admin/login">) {
  // Already in: straight to the desk.
  if (isStaffWithCode(await getSession())) redirect("/admin/overview");

  const sp = await searchParams;
  const error = Object.keys(ERRORS).find((k) => k === sp.error) as keyof typeof ERRORS | undefined;

  return (
    <main id="main" className="admin-login">
      <div className="admin-login__inner">
        <Link href="/" className="admin-login__brand" aria-label="Marginalleya — back to the shop">
          <Logo className="h-12 w-auto text-gold-400" />
        </Link>

        <section className="co__card admin-login__card" aria-labelledby="admin-login-title">
          <div className="catalogue__card-head small-upper" aria-hidden="true">
            <span>Shop office</span>
            <span>Staff only</span>
          </div>
          <h1 id="admin-login-title" className="admin-login__title">
            Sign in to the <em>back room</em>
          </h1>
          <form action={staffSignIn} className="auth__form">
            {error && (
              <p role="alert" className="auth__error m-0">
                {ERRORS[error]}
                {error === "needs-two-step" && (
                  <>
                    {" "}
                    <Link href="/account#two-step" className="under-hover">
                      Go to your account
                    </Link>
                  </>
                )}
              </p>
            )}
            <label className="auth__field">
              <span className="co__label small-upper">Staff email</span>
              <input className="co__input" name="email" type="email" autoComplete="username" placeholder="you@marginalleya.com" required />
            </label>
            <PasswordField label="Password" name="password" autoComplete="current-password" />
            <SubmitButton pending="Signing in…">Sign in</SubmitButton>
          </form>
          <p className="admin-login__help">Lost access? Ask the shop owner to reset it.</p>
        </section>

        <Link href="/" className="admin-login__out small-upper under-hover">
          Back to the shop
        </Link>
      </div>
    </main>
  );
}
