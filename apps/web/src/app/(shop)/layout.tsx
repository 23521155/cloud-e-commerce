import { SiteFooter } from "@/components/layout/SiteFooter";
import { SiteHeader } from "@/components/layout/SiteHeader";

// Book status and prices change with every order, so render per request: prerendering would freeze
// build-time data and needs the gateway during `next build`, which a Docker build does not have.
export const dynamic = "force-dynamic";

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
