import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { MotionController } from "@/components/motion/MotionController";
import { Sell } from "@/components/sell/Sell";
import { MAX_PHOTO_BYTES, MAX_PHOTOS, photosOf, submitSellRequest } from "@/lib/sell-api";

export const metadata: Metadata = {
  title: "Sell your books — Marginalleya",
  description: "Describe a book you would like to sell and the shop will reply with an offer.",
};

const ERRORS: Record<string, string> = {
  "too-many": `Send at most ${MAX_PHOTOS} photos.`,
  "too-large": `Each photo must be under ${MAX_PHOTO_BYTES / 1024 / 1024} MB.`,
  "not-image": "Photos must be JPEG, PNG, WebP or GIF pictures.",
  invalid: "Something in the form is not right. Check the required fields and try again.",
  busy: "Too many offers from here just now. Please wait a minute and try again.",
  failed: "The offer could not be sent. Please try again in a moment.",
};

async function requestOffer(formData: FormData) {
  "use server";
  const photos = photosOf(formData);
  if (photos.length > MAX_PHOTOS) redirect("/sell?error=too-many");
  if (photos.some((p) => p.size > MAX_PHOTO_BYTES)) redirect("/sell?error=too-large");

  const result = await submitSellRequest(formData);
  if (result.ok) redirect("/sell?sent=1");
  if (result.reason === "invalid" && result.detail?.includes("image")) redirect("/sell?error=not-image");
  redirect(`/sell?error=${result.reason}`);
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

        <Sell action={requestOffer} sent={sp.sent === "1"} error={typeof sp.error === "string" ? ERRORS[sp.error] : undefined} />
      </section>
    </>
  );
}
