import { useState, useEffect, useRef } from 'react';

interface Message {
  username: string;
  text: string;
  isFile?: boolean;
  fileUrl?: string;
  fileName?: string;
}

export default function Chat() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [input, setInput] = useState('');
  const [username, setUsername] = useState('');
  const [isConnected, setIsConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    const ws = new WebSocket(`ws://${window.location.hostname}:8080/ws`);
    wsRef.current = ws;

    ws.onopen = () => {
      console.log('Connected');
      setIsConnected(true);
    };

    ws.onmessage = (event) => {
      const msg: Message = JSON.parse(event.data);
      setMessages((prev) => [...prev, msg]);
    };

    ws.onclose = () => {
      console.log('Disconnected');
      setIsConnected(false);
    };

    return () => {
      ws.close();
    };
  }, []);

  const sendMessage = (text: string, isFile = false, fileUrl = '', fileName = '') => {
    if (!wsRef.current) return;
    const msg: Message = { username, text, isFile, fileUrl, fileName };
    wsRef.current.send(JSON.stringify(msg));
  };

  const handleSendText = () => {
    if (input.trim() === '') return;
    sendMessage(input);
    setInput('');
  };

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const formData = new FormData();
    formData.append('file', file);

    try {
      const response = await fetch(`http://${window.location.hostname}:8080/upload`, {
        method: 'POST',
        body: formData,
      });
      const downloadUrl = await response.text(); // "/download/filename"
      sendMessage(`Файл: ${file.name}`, true, downloadUrl, file.name);
    } catch (err) {
      console.error('Upload failed', err);
    }
  };

  if (!username) {
    return (
      <div>
        <h2>Enter your name</h2>
        <input
          type="text"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          placeholder="Username"
        />
        <button onClick={() => username && setUsername(username)}>Join</button>
      </div>
    );
  }

  return (
    <div>
      <h2>Chat as {username}</h2>
      <div style={{ height: '300px', overflowY: 'scroll', border: '1px solid black', marginBottom: '10px', padding: '5px' }}>
        {messages.map((msg, idx) => (
          <div key={idx}>
            <strong>{msg.username}:</strong>{' '}
            {msg.isFile ? (
              <a href={`http://${window.location.hostname}:8080${msg.fileUrl}`} target="_blank" rel="noopener noreferrer">
                {msg.text}
              </a>
            ) : (
              msg.text
            )}
          </div>
        ))}
      </div>
      <input
        type="text"
        value={input}
        onChange={(e) => setInput(e.target.value)}
        onKeyDown={(e) => e.key === 'Enter' && handleSendText()}
        placeholder="Message"
      />
      <button onClick={handleSendText}>Send</button>
      <input type="file" onChange={handleFileUpload} />
      <div>Status: {isConnected ? 'Connected' : 'Disconnected'}</div>
    </div>
  );
}