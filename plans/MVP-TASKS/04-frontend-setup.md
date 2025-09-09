# Task 04: React Frontend Setup

## Objective
Set up a minimal React frontend with dark mode, authentication, and basic navigation for the MVP.

## Prerequisites
- Studio API Service (Task 03) running on port 8010
- Node.js and npm installed
- `/home/uneid/iter3/memmieai/memmie-studio` directory exists

## Task Steps

### Step 1: Initialize React App
```bash
cd /home/uneid/iter3/memmieai/memmie-studio
npx create-react-app web --template typescript
cd web
```

### Step 2: Install Dependencies
```bash
npm install axios react-router-dom @types/react-router-dom
npm install --save-dev @types/node
```

### Step 3: Create API Client
Create file: `/home/uneid/iter3/memmieai/memmie-studio/web/src/api/client.ts`

```typescript
import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_URL || 'http://localhost:8010/api/v1';

const apiClient = axios.create({
  baseURL: API_BASE_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Add auth token to requests
apiClient.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('auth_token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Handle auth errors
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('auth_token');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

export interface Provider {
  id: string;
  provider_id: string;
  name: string;
  description: string;
  template: any;
}

export interface Document {
  id: string;
  provider_id: string;
  title: string;
  preview: string;
  metadata: any;
  created_at: string;
  updated_at: string;
}

export interface CreateDocumentRequest {
  provider_id: string;
  content: string;
  process_content: boolean;
  metadata?: any;
}

export interface DocumentResponse {
  id: string;
  provider_id: string;
  original: string;
  processed?: string;
  processed_id?: string;
  metadata: any;
  created_at: string;
}

export const api = {
  // Auth (using Auth service directly for now)
  login: async (email: string, password: string) => {
    const response = await axios.post('http://localhost:8001/api/v1/auth/login', {
      email,
      password,
    });
    return response.data;
  },

  register: async (email: string, password: string, username: string) => {
    const response = await axios.post('http://localhost:8001/api/v1/auth/register', {
      email,
      password,
      username,
    });
    return response.data;
  },

  // Providers
  getProviders: async (): Promise<Provider[]> => {
    const response = await apiClient.get('/providers');
    return response.data.providers;
  },

  // Documents
  createDocument: async (data: CreateDocumentRequest): Promise<DocumentResponse> => {
    const response = await apiClient.post('/documents', data);
    return response.data;
  },

  listDocuments: async (providerId?: string): Promise<Document[]> => {
    const params = providerId ? { provider_id: providerId } : {};
    const response = await apiClient.get('/documents', { params });
    return response.data.documents;
  },
};
```

### Step 4: Create Auth Context
Create file: `/home/uneid/iter3/memmieai/memmie-studio/web/src/contexts/AuthContext.tsx`

```typescript
import React, { createContext, useContext, useState, useEffect } from 'react';
import { api } from '../api/client';

interface AuthContextType {
  isAuthenticated: boolean;
  user: any | null;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, username: string) => Promise<void>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within AuthProvider');
  }
  return context;
};

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [user, setUser] = useState<any | null>(null);

  useEffect(() => {
    const token = localStorage.getItem('auth_token');
    if (token) {
      setIsAuthenticated(true);
      // TODO: Validate token and get user info
    }
  }, []);

  const login = async (email: string, password: string) => {
    const response = await api.login(email, password);
    localStorage.setItem('auth_token', response.token);
    setIsAuthenticated(true);
    setUser(response.user);
  };

  const register = async (email: string, password: string, username: string) => {
    const response = await api.register(email, password, username);
    localStorage.setItem('auth_token', response.token);
    setIsAuthenticated(true);
    setUser(response.user);
  };

  const logout = () => {
    localStorage.removeItem('auth_token');
    setIsAuthenticated(false);
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ isAuthenticated, user, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  );
};
```

### Step 5: Create Login Component
Create file: `/home/uneid/iter3/memmieai/memmie-studio/web/src/components/Login.tsx`

```typescript
import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import './Login.css';

export const Login: React.FC = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const { login } = useAuth();
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await login(email, password);
      navigate('/');
    } catch (err: any) {
      setError(err.response?.data?.error || 'Login failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="login-container">
      <div className="login-box">
        <h1>Memmie Studio</h1>
        <h2>Sign In</h2>
        <form onSubmit={handleSubmit}>
          <input
            type="email"
            placeholder="Email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            disabled={loading}
          />
          <input
            type="password"
            placeholder="Password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            disabled={loading}
          />
          {error && <div className="error">{error}</div>}
          <button type="submit" disabled={loading}>
            {loading ? 'Signing in...' : 'Sign In'}
          </button>
        </form>
        <p>
          Don't have an account? <Link to="/register">Sign Up</Link>
        </p>
      </div>
    </div>
  );
};
```

Create file: `/home/uneid/iter3/memmieai/memmie-studio/web/src/components/Login.css`

```css
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #0a0a0a;
}

.login-box {
  background: #1a1a1a;
  padding: 40px;
  border-radius: 12px;
  width: 100%;
  max-width: 400px;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
}

.login-box h1 {
  color: #ffffff;
  text-align: center;
  margin-bottom: 10px;
  font-size: 28px;
}

.login-box h2 {
  color: #888;
  text-align: center;
  margin-bottom: 30px;
  font-size: 18px;
  font-weight: normal;
}

.login-box input {
  width: 100%;
  padding: 12px;
  margin-bottom: 15px;
  background: #0a0a0a;
  border: 1px solid #333;
  border-radius: 6px;
  color: #ffffff;
  font-size: 16px;
}

.login-box input:focus {
  outline: none;
  border-color: #4a9eff;
}

.login-box button {
  width: 100%;
  padding: 12px;
  background: #4a9eff;
  color: white;
  border: none;
  border-radius: 6px;
  font-size: 16px;
  cursor: pointer;
  transition: background 0.2s;
}

.login-box button:hover:not(:disabled) {
  background: #3a8eef;
}

.login-box button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.login-box .error {
  color: #ff4444;
  text-align: center;
  margin-bottom: 15px;
  font-size: 14px;
}

.login-box p {
  text-align: center;
  color: #888;
  margin-top: 20px;
}

.login-box a {
  color: #4a9eff;
  text-decoration: none;
}

.login-box a:hover {
  text-decoration: underline;
}
```

### Step 6: Create Dashboard Component
Create file: `/home/uneid/iter3/memmieai/memmie-studio/web/src/components/Dashboard.tsx`

```typescript
import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { api, Provider, Document } from '../api/client';
import './Dashboard.css';

export const Dashboard: React.FC = () => {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [documents, setDocuments] = useState<Document[]>([]);
  const [selectedProvider, setSelectedProvider] = useState<string>('');
  const [loading, setLoading] = useState(true);
  const { logout } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    loadData();
  }, [selectedProvider]);

  const loadData = async () => {
    try {
      setLoading(true);
      const [providersData, documentsData] = await Promise.all([
        api.getProviders(),
        api.listDocuments(selectedProvider),
      ]);
      setProviders(providersData);
      setDocuments(documentsData);
    } catch (error) {
      console.error('Failed to load data:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleNewDocument = (providerId: string) => {
    navigate(`/create/${providerId}`);
  };

  const handleLogout = () => {
    logout();
    navigate('/login');
  };

  return (
    <div className="dashboard">
      <header className="dashboard-header">
        <h1>Memmie Studio</h1>
        <button onClick={handleLogout} className="logout-btn">
          Sign Out
        </button>
      </header>

      <div className="dashboard-content">
        <aside className="sidebar">
          <h2>Providers</h2>
          <div className="provider-list">
            <button
              className={`provider-item ${selectedProvider === '' ? 'active' : ''}`}
              onClick={() => setSelectedProvider('')}
            >
              All Documents
            </button>
            {providers.map((provider) => (
              <button
                key={provider.provider_id}
                className={`provider-item ${selectedProvider === provider.provider_id ? 'active' : ''}`}
                onClick={() => setSelectedProvider(provider.provider_id)}
              >
                {provider.name}
              </button>
            ))}
          </div>
        </aside>

        <main className="main-content">
          <div className="content-header">
            <h2>{selectedProvider ? providers.find(p => p.provider_id === selectedProvider)?.name : 'All Documents'}</h2>
            {selectedProvider && (
              <button
                className="new-doc-btn"
                onClick={() => handleNewDocument(selectedProvider)}
              >
                + New Document
              </button>
            )}
          </div>

          {loading ? (
            <div className="loading">Loading...</div>
          ) : documents.length === 0 ? (
            <div className="empty-state">
              <p>No documents yet</p>
              {selectedProvider && (
                <button
                  className="new-doc-btn"
                  onClick={() => handleNewDocument(selectedProvider)}
                >
                  Create your first document
                </button>
              )}
            </div>
          ) : (
            <div className="document-grid">
              {documents.map((doc) => (
                <div key={doc.id} className="document-card">
                  <h3>{doc.title}</h3>
                  <p>{doc.preview}</p>
                  <div className="document-meta">
                    <span>{new Date(doc.created_at).toLocaleDateString()}</span>
                    <span>{providers.find(p => p.provider_id === doc.provider_id)?.name}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </main>
      </div>
    </div>
  );
};
```

Create file: `/home/uneid/iter3/memmieai/memmie-studio/web/src/components/Dashboard.css`

```css
.dashboard {
  min-height: 100vh;
  background: #0a0a0a;
  color: #ffffff;
}

.dashboard-header {
  background: #1a1a1a;
  padding: 15px 30px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-bottom: 1px solid #333;
}

.dashboard-header h1 {
  font-size: 24px;
  margin: 0;
}

.logout-btn {
  padding: 8px 16px;
  background: transparent;
  color: #888;
  border: 1px solid #333;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
}

.logout-btn:hover {
  color: #fff;
  border-color: #666;
}

.dashboard-content {
  display: flex;
  height: calc(100vh - 65px);
}

.sidebar {
  width: 250px;
  background: #1a1a1a;
  padding: 20px;
  border-right: 1px solid #333;
  overflow-y: auto;
}

.sidebar h2 {
  font-size: 14px;
  text-transform: uppercase;
  color: #666;
  margin-bottom: 15px;
}

.provider-list {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.provider-item {
  padding: 10px 15px;
  background: transparent;
  color: #888;
  border: none;
  border-radius: 6px;
  text-align: left;
  cursor: pointer;
  transition: all 0.2s;
}

.provider-item:hover {
  background: #2a2a2a;
  color: #fff;
}

.provider-item.active {
  background: #2a2a2a;
  color: #4a9eff;
}

.main-content {
  flex: 1;
  padding: 30px;
  overflow-y: auto;
}

.content-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 30px;
}

.content-header h2 {
  font-size: 28px;
  margin: 0;
}

.new-doc-btn {
  padding: 10px 20px;
  background: #4a9eff;
  color: white;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.2s;
}

.new-doc-btn:hover {
  background: #3a8eef;
}

.loading {
  text-align: center;
  color: #666;
  padding: 50px;
}

.empty-state {
  text-align: center;
  padding: 50px;
  color: #666;
}

.empty-state p {
  margin-bottom: 20px;
}

.document-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 20px;
}

.document-card {
  background: #1a1a1a;
  padding: 20px;
  border-radius: 8px;
  border: 1px solid #333;
  cursor: pointer;
  transition: all 0.2s;
}

.document-card:hover {
  border-color: #4a9eff;
  transform: translateY(-2px);
}

.document-card h3 {
  margin: 0 0 10px 0;
  font-size: 18px;
  color: #fff;
}

.document-card p {
  color: #888;
  margin: 0 0 15px 0;
  line-height: 1.5;
}

.document-meta {
  display: flex;
  justify-content: space-between;
  font-size: 12px;
  color: #666;
}
```

### Step 7: Update App.tsx
Replace file: `/home/uneid/iter3/memmieai/memmie-studio/web/src/App.tsx`

```typescript
import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './contexts/AuthContext';
import { Login } from './components/Login';
import { Dashboard } from './components/Dashboard';
import './App.css';

const PrivateRoute: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { isAuthenticated } = useAuth();
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" />;
};

function App() {
  return (
    <Router>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route
            path="/"
            element={
              <PrivateRoute>
                <Dashboard />
              </PrivateRoute>
            }
          />
        </Routes>
      </AuthProvider>
    </Router>
  );
}

export default App;
```

### Step 8: Update App.css
Replace file: `/home/uneid/iter3/memmieai/memmie-studio/web/src/App.css`

```css
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Oxygen',
    'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue',
    sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  background: #0a0a0a;
  color: #ffffff;
}

code {
  font-family: source-code-pro, Menlo, Monaco, Consolas, 'Courier New',
    monospace;
}

#root {
  min-height: 100vh;
}
```

### Step 9: Update index.css
Replace file: `/home/uneid/iter3/memmieai/memmie-studio/web/src/index.css`

```css
body {
  margin: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Oxygen',
    'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue',
    sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
  background: #0a0a0a;
  color: #ffffff;
}
```

### Step 10: Build and Test

```bash
# Terminal 1: Development mode with hot reload
cd /home/uneid/iter3/memmieai/memmie-studio/web
npm start
# Opens at http://localhost:3000

# Terminal 2: Build for production
cd /home/uneid/iter3/memmieai/memmie-studio/web
npm run build
# Creates optimized build in web/build/

# Terminal 3: Test with Studio API serving the built files
cd /home/uneid/iter3/memmieai/memmie-studio
go run cmd/server/main.go
# Access at http://localhost:8010
```

## Expected Output
- Development server runs on port 3000
- Login page with dark theme
- Dashboard showing providers (Book Writer, Pitch Creator)
- Document list view
- Responsive layout
- Authentication flow working

## Success Criteria
✅ React app compiles without errors
✅ Dark mode theme applied throughout
✅ Login/logout functionality works
✅ Can view list of providers
✅ Can view documents (when available)
✅ Navigation between views works
✅ API client connects to Studio service
✅ Production build completes successfully

## Notes
- Uses TypeScript for type safety
- Simple dark theme with #0a0a0a background
- Authentication tokens stored in localStorage
- API calls go through Studio service (port 8010)
- Next tasks will add document creation interfaces