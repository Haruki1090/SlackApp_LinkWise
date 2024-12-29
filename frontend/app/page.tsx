"use client";

import { useState } from "react";

export default function Home() {
  const [url, setUrl] = useState("");
  const [result, setResult] = useState("");
  const [error, setError] = useState("");

  const handleFetch = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setResult("");

    try {
      const response = await fetch("/api/fetch-message", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url }),
      });

      if (!response.ok) {
        throw new Error("Failed to fetch message");
      }

      const data = await response.json();
      setResult(JSON.stringify(data, null, 2));
    } catch (err: unknown) {
      if (err instanceof Error) {
        setError(err.message || "エラーが発生しました");
      } else {
        setError("エラーが発生しました");
      }
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-r from-blue-50 to-blue-100 flex items-center justify-center">
      <div className="bg-white shadow-md rounded-lg p-8 max-w-md w-full">
        <h1 className="text-2xl font-bold text-gray-800 mb-6 text-center">
          Slack Message Fetcher
        </h1>
        <form onSubmit={handleFetch} className="space-y-4">
          <input
            type="text"
            placeholder="SlackのメッセージURLを入力"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-400"
          />
          <button
            type="submit"
            className="w-full bg-blue-500 text-white py-2 px-4 rounded-lg hover:bg-blue-600 focus:outline-none focus:ring-2 focus:ring-blue-400"
          >
            取得
          </button>
        </form>
        {error && <p className="text-red-500 text-center mt-4">{error}</p>}
        {result && (
          <pre className="bg-gray-100 p-4 mt-4 rounded-lg text-sm overflow-auto">
            {result}
          </pre>
        )}
      </div>
    </div>
  );
}
