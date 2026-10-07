import type { Metadata } from "next";
import Link from "next/link";
import { redirect } from "next/navigation";
import { PasswordField, SubmitButton } from "@/components/auth/AuthClient";
import { Logo } from "@/components/ui/Logo";

// Lives outside the (admin) group so it gets no admin sidebar; the URL is still /admin/login.
export const metadata: Metadata = {
  title: "Staff sign in — Marginalleya",
  robots: { index: false },
};

/** Mock staff sign-in: no accounts or sessions yet. Credentials stay in the POST body and are dropped. */
async function staffSignIn() {
  "use server";
  redirect("/admin/overview");
}

export default function AdminLoginPage() {
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
