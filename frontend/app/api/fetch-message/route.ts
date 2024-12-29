import { NextResponse } from 'next/server';

// POSTメソッドを処理
export async function POST(request: Request) {
  try {
    // リクエストボディを取得
    const body = await request.json();

    // バックエンドのエンドポイントURLを環境変数から取得
    const backendURL = process.env.BACKEND_URL;

    if (!backendURL) {
      return NextResponse.json(
        { message: 'Backend URL is not configured.' },
        { status: 500 }
      );
    }

    // バックエンドにPOSTリクエストを送信
    const response = await fetch(`${backendURL}/api/fetch-message`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    });

    // バックエンドからのレスポンスがOKでない場合
    if (!response.ok) {
      const errorData = await response.json();
      return NextResponse.json(
        { message: errorData.message || 'Failed to fetch message from backend.' },
        { status: response.status }
      );
    }

    // バックエンドからのデータを取得
    const data = await response.json();

    // フロントエンドにレスポンスを返す
    return NextResponse.json(data, { status: 200 });

  } catch (error) {
    console.error('Error in fetch-message API route:', error);
    return NextResponse.json(
      { message: 'Internal Server Error' },
      { status: 500 }
    );
  }
}
