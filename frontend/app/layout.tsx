import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "KestrelOps — Kubernetes Helm Safety Gate",
  description: "Scan Kubernetes manifests and Helm charts for security, deprecated APIs, reliability issues and CVEs before you deploy.",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className="dark">
      <body className="bg-[#080c14] text-slate-100 antialiased">{children}</body>
    </html>
  );
}
