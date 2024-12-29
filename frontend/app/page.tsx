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

  const handleFetchMessages = async () => {
    if (!url) return alert("Please enter a valid Slack message URL.");
    setLoading(true);
    try {
      const response = await fetch(`${process.env.NEXT_PUBLIC_BACKEND_URL}/api/fetch-message`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ url }),
      });
      const data = await response.json();
      setMessages(data.messages || []);
    } catch (error) {
      console.error("Error fetching messages:", error);
      alert("Failed to fetch messages. Please try again.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <div className="mb-6">
        <h2 className="text-xl font-bold mb-4" style={{ color: "#a22041" }}>
          Enter Slack Message URL
        </h2>
        <div className="flex gap-2">
          <input
            type="text"
            placeholder="https://slack.com/..."
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            className="flex-1 p-3 border rounded-lg focus:outline-none focus:ring-2 focus:ring-pink-500"
          />
          <button
            onClick={handleFetchMessages}
            disabled={loading}
            className={`px-4 py-2 text-white rounded-lg ${
              loading
                ? "bg-gray-400 cursor-not-allowed"
                : "bg-[#a22041] hover:bg-[#891938]"
            }`}
          >
            {loading ? "Fetching..." : "Fetch"}
          </button>
        </div>
      </div>
      <div>
        <h3 className="text-lg font-semibold mb-3" style={{ color: "#a22041" }}>
          Messages
        </h3>
        <div className="space-y-4 max-h-96 overflow-y-auto">
          {messages.map((msg, index) => (
            <div
              key={index}
              className="bg-gray-100 p-4 rounded-lg border border-gray-200"
            >
              <p className="text-sm text-gray-600">{msg.timestamp}</p>
              <p className="font-semibold" style={{ color: "#a22041" }}>
                {msg.user_name}
              </p>
              <p className="text-gray-800">{msg.text}</p>
            </div>
          ))}
          {messages.length === 0 && (
            <p className="text-gray-600">No messages found.</p>
          )}
        </div>
      </div>
    </div>
  );
}
