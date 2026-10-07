import type { Metadata } from "next";
import Link from "next/link";
import { AdminNav } from "@/components/admin/AdminUI";
import { Logo } from "@/components/ui/Logo";

export const metadata: Metadata = {
  title: "Shop office — Marginalleya",
  robots: { index: false },
};

/** The shop's back office: a walnut sidebar and a parchment desk to work on. No customer nav or footer. */
export default function AdminLayout({ children }: LayoutProps<"/">) {
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
      </aside>

      <main id="main" className="admin__desk">
        {children}
      </main>
    </div>
  );
}
