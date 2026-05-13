import React, { useState } from 'react';
import Chat from './Chat';

const App: React.FC = () => {
  const [username, setUsername] = useState<string | null>(null);
  const [inputName, setInputName] = useState('');

  const handleJoin = (e: React.FormEvent) => {
    e.preventDefault();
    if (inputName.trim()) {
      setUsername(inputName.trim());
    }
  };

  if (!username) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh' }}>
        <form onSubmit={handleJoin} style={{ textAlign: 'center' }}>
          <h2>Введите ваше имя</h2>
          <input
            type="text"
            value={inputName}
            onChange={(e) => setInputName(e.target.value)}
            autoFocus
            style={{ padding: '10px', fontSize: '16px', marginRight: '10px' }}
          />
          <button type="submit" style={{ padding: '10px 20px', fontSize: '16px' }}>Войти</button>
        </form>
      </div>
    );
  }

  return <Chat username={username} />;
};

export default App;