import { useState, useRef } from 'react';
import './App.css';

function App() {
  const [comando, setComando] = useState('');
  const [salida, setSalida] = useState('');
  const fileInputRef = useRef(null);

  const handleChooseFileClick = () => {
    fileInputRef.current.click();
  };

  const handleFileChange = (event) => {
    const file = event.target.files[0];
    if (!file) return;

    const reader = new FileReader();
    reader.onload = (e) => {
      setComando(e.target.result);
    };
    reader.readAsText(file);
  };

  const handleClear = () => {
    setComando('');
    setSalida('');
  };

  const ejecutarComandos = async () => {
    const response = await fetch('http://localhost:8000/execute', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ comando }),
    });

    const data = await response.json();
    setSalida(prev => prev + `${data.salida}\n`);
    setComando('');
  };

  const handleKeyPress = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      ejecutarComandos();
    }
  };

  return (
    <>
      <section>
        <div className='menuBotones'>
          <input
            type="file"
            ref={fileInputRef}
            onChange={handleFileChange}
            style={{ display: 'none' }}
            accept=".txt,.sh"
          />
          <button onClick={handleChooseFileClick}>Elegir archivo</button>
          <button onClick={ejecutarComandos}>Ejecutar</button>
          <button onClick={handleClear}>Limpiar</button>
          <div className='titulo'><div>ExtreamFS</div></div>
        </div>
        <div className='ioWrapper'>
          <div className='entradaComandos'>
            <h3>Entrada</h3>
            <textarea
              placeholder='Escribe tus comandos aquí o carga un archivo... '
              value={comando}
              onChange={(e) => setComando(e.target.value)}
              onKeyPress={handleKeyPress}
            ></textarea>
          </div>
          <div className='salidaComandos'>
            <h3>Salida</h3>
            <textarea
              placeholder='La salida aparecerá aquí...'
              value={salida}
              readOnly
            ></textarea>
          </div>
        </div>
      </section>
    </>
  );
}

export default App;