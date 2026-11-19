import { SiteFooter } from "@/components/layout/SiteFooter";
import { SiteHeader } from "@/components/layout/SiteHeader";
import { getSession } from "@/lib/session";

// Render per request (never at build time: the gateway is not reachable during a Docker build),
// while fetches that set their own revalidate (the catalogue reads) stay cached between visitors.
export const revalidate = 0;


/** Customer-facing pages: side nav, content, footer. */
export default async function ShopLayout({ children }: LayoutProps<"/">) {
  // Read from the session cookie, so showing who is signed in costs no call to the user service.
  const session = await getSession();
  return (
    <>
      <SiteHeader reader={session ? { name: session.name } : null} />
      <main id="main" className="flex-1">
        {children}
      </main>
      <SiteFooter />
    </>
  );
}
