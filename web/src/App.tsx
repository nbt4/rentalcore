import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { AuthProvider } from './contexts/AuthContext';
import { ProtectedRoute } from './components/ProtectedRoute';
import { RoleGuard } from './components/RoleGuard';
import { Layout } from './components/Layout';
import { Login } from './pages/Login';
import { ChangePassword } from './pages/ChangePassword';
import { Dashboard } from './pages/Dashboard';
import { JobsPage } from './pages/JobsPage';
import { CustomersPage } from './pages/CustomersPage';
import { AnalyticsPage } from './pages/AnalyticsPage';
import { DocumentsPage } from './pages/DocumentsPage';
import { AdminPage } from './pages/AdminPage';
import { ProfilePage } from './pages/ProfilePage';

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/change-password" element={
            <ProtectedRoute bypassForcePasswordChange><ChangePassword /></ProtectedRoute>
          } />

          <Route path="/" element={<ProtectedRoute><Layout><Dashboard /></Layout></ProtectedRoute>} />
          <Route path="/jobs" element={<ProtectedRoute><Layout><JobsPage /></Layout></ProtectedRoute>} />
          <Route path="/jobs/:id" element={<ProtectedRoute><Layout><JobsPage /></Layout></ProtectedRoute>} />
          <Route path="/jobs/:id/edit" element={<ProtectedRoute><Layout><JobsPage /></Layout></ProtectedRoute>} />
          <Route path="/customers" element={<ProtectedRoute><Layout><CustomersPage /></Layout></ProtectedRoute>} />
          <Route path="/customers/:id" element={<ProtectedRoute><Layout><CustomersPage /></Layout></ProtectedRoute>} />
          <Route path="/customers/:id/edit" element={<ProtectedRoute><Layout><CustomersPage /></Layout></ProtectedRoute>} />
          <Route path="/analytics" element={<ProtectedRoute><Layout><AnalyticsPage /></Layout></ProtectedRoute>} />
          <Route path="/documents" element={<ProtectedRoute><Layout><DocumentsPage /></Layout></ProtectedRoute>} />
          <Route path="/profile" element={<ProtectedRoute><Layout><ProfilePage /></Layout></ProtectedRoute>} />
          <Route
            path="/admin"
            element={
              <ProtectedRoute>
                <Layout>
                  <RoleGuard requiredRoles={['admin', 'manager']}>
                    <AdminPage />
                  </RoleGuard>
                </Layout>
              </ProtectedRoute>
            }
          />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}

export default App;
