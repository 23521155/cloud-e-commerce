import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { MotionController } from "@/components/motion/MotionController";
import { Sell } from "@/components/sell/Sell";

export const metadata: Metadata = {
  title: "Sell your books — Marginalleya",
  description: "Describe a book you would like to sell and the shop will reply with an offer.",
};

/** Mock offer request: nothing is stored or sent yet. Details and photos stay in the POST body, never the URL. */
async function requestOffer() {
  "use server";
  redirect("/sell?sent=1");
}

export default async function SellPage({ searchParams }: PageProps<"/sell">) {
  const sp = await searchParams;

  return (
    <>
      <MotionController />

      <section className="work basket" aria-labelledby="sell-title">
        <div className="catalogue__head">
          <h1 id="sell-title" data-reveal className="big-sans m-0 text-[clamp(2.4rem,6vw,7rem)]">
            <span className="line">
              <span>Sell us</span>
            </span>
            <span className="line">
              <span className="em-italic text-gold-400">your books</span>
            </span>
          </h1>
          <p className="m-0 max-w-[34ch] text-[clamp(17px,1.3vw,21px)] leading-relaxed text-parchment-200">
            Describe the copy honestly, faults first. We read it and reply with an offer.
          </p>
        </div>

        <Sell action={requestOffer} sent={sp.sent === "1"} />
      </section>
    </>
  );
}
