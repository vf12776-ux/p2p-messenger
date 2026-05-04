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
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const ws = new WebSocket(`${protocol}//${window.location.host}/ws`);
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

    return () => ws.close();
  }, []);

  const sendMessage = (text: string, isFile = false, fileUrl = '', fileName = '') => {
    if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;
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
      const response = await fetch('/upload', { method: 'POST', body: formData });
      const downloadUrl = await response.text();
      sendMessage(`Файл: ${file.name}`, true, downloadUrl, file.name);
    } catch (err) {
      console.error('Upload failed', err);
      alert('Ошибка загрузки файла');
    }
  };

  if (!username) {
    return (
      <div style={{ padding: '20px' }}>
        <h2>Введите ваше имя</h2>
        <input
          type="text"
          value={username}
          onChange={(e) => setUsername(e.target.value)}
          placeholder="Имя"
          style={{ marginRight: '10px' }}
        />
        <button onClick={() => username && setUsername(username)}>Войти</button>
      </div>
    );
  }

  return (
    <div style={{ padding: '20px' }}>
      <h2>Чат: {username}</h2>
      <div
        style={{
          height: '300px',
          overflowY: 'scroll',
          border: '1px solid #ccc',
          marginBottom: '10px',
          padding: '5px',
        }}
      >
        {messages.map((msg, idx) => (
          <div key={idx}>
            <strong>{msg.username}:</strong>{' '}
            {msg.isFile ? (
              <a href={msg.fileUrl} target="_blank" rel="noopener noreferrer">
                {msg.text}
              </a>
            ) : (
              msg.text
            )}
          </div>
        ))}
      </div>
      <div style={{ display: 'flex', gap: '10px', alignItems: 'center' }}>
        <input
          type="text"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleSendText()}
          placeholder="Сообщение"
          style={{ flex: 1 }}
        />
        <button onClick={handleSendText}>Отправить</button>
        <input type="file" onChange={handleFileUpload} />
      </div>
      <div style={{ marginTop: '10px' }}>Статус: {isConnected ? 'Подключено' : 'Отключено'}</div>
    </div>
  );
}