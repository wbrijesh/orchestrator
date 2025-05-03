import type { Metadata } from "next";
import { Inter } from "next/font/google";
import "./globals.css";
import { UserProvider } from "@/context/UserContext";

// Load Inter font with all weights
const inter = Inter({
  subsets: ["latin"],
  display: "swap",
  weight: ["400", "500", "600", "700"],
});

export const metadata: Metadata = {
  title: "Orchestrator",
  description: "Orchestrator application with authentication",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="light">
      <body className={inter.className}>
        <UserProvider>
          {children}
        </UserProvider>
      </body>
    </html>
  );
}
