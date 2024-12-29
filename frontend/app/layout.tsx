export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <head>
        <title>Slack Message Fetcher</title>
      </head>
      <body className="bg-gradient-to-r from-blue-100 to-blue-300 min-h-screen flex items-center justify-center">
        {children}
      </body>
    </html>
  );
}
