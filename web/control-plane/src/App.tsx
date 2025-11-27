import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ApiProvider } from './hooks/useApi';
import { Layout } from './components/Layout';
import { Dashboard, Topology, Routing, Data } from './pages';
import './App.css';

function App() {
  return (
    <ApiProvider>
      <BrowserRouter>
        <Routes>
          <Route path="/admin" element={<Layout />}>
            <Route index element={<Dashboard />} />
            <Route path="topology" element={<Topology />} />
            <Route path="routing" element={<Routing />} />
            <Route path="data" element={<Data />} />
          </Route>
          {/* Redirect root to admin */}
          <Route path="/" element={<Navigate to="/admin" replace />} />
          {/* Catch-all redirect */}
          <Route path="*" element={<Navigate to="/admin" replace />} />
        </Routes>
      </BrowserRouter>
    </ApiProvider>
  );
}

export default App;
