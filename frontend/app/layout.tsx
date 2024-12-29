import "./globals.css";

export const metadata = {
  title: "Slack Message Fetcher",
  description: "Fetch Slack thread messages and display them beautifully",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body className="bg-gray-50 text-gray-900 font-sans">
        <header className="bg-white border-b border-gray-200 p-4 shadow-sm">
          <h1 className="text-3xl font-bold" style={{ color: "#a22041" }}>
            Slack Message Fetcher
          </h1>
        </header>
        <main className="flex justify-center items-center min-h-screen p-6">
          <div className="w-full max-w-4xl bg-white shadow-lg rounded-lg p-8">
            {children}
          </div>
        </main>
        <footer className="text-center text-gray-500 text-sm py-4">
          © 2024 Slack Message Fetcher. All Rights Reserved.
        </footer>
      </body>
    </html>
  );
}
