"use client";

import React, { useState } from "react";

interface SlackMessage {
  text: string;
  user: string;
  ts: string;
  user_name: string;
  timestamp: string;
}

export default function Home() {
  const [url, setUrl] = useState("");
  const [messages, setMessages] = useState<SlackMessage[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleFetchMessages = async () => {
    if (!url.trim()) {
      setError("Please enter a valid Slack message URL.");
      return;
    }
    setError(null); // Reset error state
    setLoading(true);
    try {
      const response = await fetch("/api/fetch-message", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url }),
      });

      if (!response.ok) {
        throw new Error(`HTTP error! Status: ${response.status}`);
      }

      const data = await response.json();
      setMessages(data.messages || []);
    } catch (err) {
      console.error("Error fetching messages:", err);
      setError("Failed to fetch messages. Please check the URL and try again.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bg-white shadow-lg rounded-lg p-8 max-w-lg w-full">
      <h1 className="text-2xl font-bold text-gray-800 text-center mb-6">
        Slack Message Fetcher
      </h1>
      <div className="space-y-4">
        <input
          type="text"
          placeholder="Enter Slack message URL"
          value={url}
          onChange={(e) => setUrl(e.target.value)}
          className="w-full p-3 border rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-400"
        />
        <button
          onClick={handleFetchMessages}
          disabled={loading}
          className={`w-full ${
            loading
              ? "bg-gray-300 cursor-not-allowed"
              : "bg-blue-500 hover:bg-blue-600"
          } text-white py-2 px-4 rounded-lg transition duration-200`}
        >
          {loading ? "Fetching..." : "取得"}
        </button>
      </div>
      {error && (
        <p className="text-red-500 text-sm mt-2">{error}</p>
      )}
      <div className="mt-6">
        <h2 className="text-lg font-semibold text-gray-700 mb-4">Messages</h2>
        <div className="space-y-4 max-h-80 overflow-y-auto">
          {messages.map((msg, index) => (
            <div key={index} className="bg-gray-50 border p-4 rounded-lg">
              <p className="text-sm text-gray-600">{msg.timestamp}</p>
              <p className="font-bold text-gray-800">{msg.user_name}</p>
              <p className="text-gray-700">{msg.text}</p>
            </div>
          ))}
          {messages.length === 0 && !loading && (
            <p className="text-gray-600">No messages found.</p>
          )}
        </div>
      </div>
    </div>
  );
}
