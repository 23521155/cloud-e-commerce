import type { Metadata } from "next";
import Link from "next/link";
import { redirect } from "next/navigation";
import { AdminNav } from "@/components/admin/AdminUI";
import { Logo } from "@/components/ui/Logo";
import { signOut } from "@/lib/auth-actions";
import { getSession, isStaffWithCode } from "@/lib/session";

// Orders and stock are live data: render per request, never at build time (see the (shop) layout).
export const dynamic = "force-dynamic";

export const metadata: Metadata = {
  title: "Shop office — Marginalleya",
  robots: { index: false },
};

/**
 * The shop's back office: a walnut sidebar and a parchment desk to work on. No customer nav or footer.
 * Staff only, signed in with two-step sign-in. This guards the pages; the data they show must be
 * guarded again where it is read, once it comes from the services.
 */
export default async function AdminLayout({ children }: LayoutProps<"/">) {
  const session = await getSession();
  if (!session) redirect("/admin/login");
  if (!session.roles.includes("staff")) redirect("/admin/login?error=not-staff");
  if (!isStaffWithCode(session)) redirect("/admin/login?error=needs-two-step");

  return (
    <div className="admin">
      <aside className="admin__side">
        <Link href="/admin/overview" className="admin__brand">
          <Logo className="h-8 w-auto text-gold-400" />
          <span>
            Marginalleya
            <em>Shop office</em>
          </span>
        </Link>

        <AdminNav />

        <Link href="/" className="admin__out small-upper under-hover">
          View the shop
        </Link>
        <form action={signOut}>
          <button type="submit" className="admin__out small-upper under-hover">
            Sign out ({session.name})
          </button>
        </form>
      </aside>

      <main id="main" className="admin__desk">
        {children}
      </main>
    </div>
  );
}
