import { SiteFooter } from "@/components/layout/SiteFooter";
import { SiteHeader } from "@/components/layout/SiteHeader";

// Render per request (never at build time: the gateway is not reachable during a Docker build),
// while fetches that set their own revalidate (the catalogue reads) stay cached between visitors.
export const revalidate = 0;


/** Customer-facing pages: side nav, content, footer. */
export default function ShopLayout({ children }: LayoutProps<"/">) {
  return (
    <>
      <SiteHeader />
      <main id="main" className="flex-1">
        {children}
      </main>
      <SiteFooter />
    </>
  );
}
