import type { Metadata } from "next";
import { notFound } from "next/navigation";
import { BookView } from "@/components/views/BookView";
import { getBook, relatedBooks } from "@/lib/catalogue";

export async function generateMetadata({ params }: PageProps<"/books/[slug]">): Promise<Metadata> {
  const book = await getBook((await params).slug);
  if (!book) return { title: "Book not found — Marginalleya" };
  return {
    title: `${book.title} — Marginalleya`,
    description: `${book.title} by ${book.author}, ${book.year}. ${book.edition}, ${book.condition}.`,
  };
}

export default async function BookPage({ params }: PageProps<"/books/[slug]">) {
  const { slug } = await params;
  const [book, related] = await Promise.all([getBook(slug), relatedBooks(slug)]);
  if (!book) notFound();

  return <BookView book={book} related={related} />;
}
