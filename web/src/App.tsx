import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider } from './contexts/AuthContext';
import { ProtectedRoute } from './components/ProtectedRoute';
import { Layout } from './components/Layout';
import { Login } from './pages/Login';
import { ChangePassword } from './pages/ChangePassword';
import { Dashboard } from './pages/Dashboard';
import { JobsPage } from './pages/JobsPage';
import { AnalyticsPage } from './pages/AnalyticsPage';
import { DocumentsPage } from './pages/DocumentsPage';
import { SkillsPage } from './pages/SkillsPage';
import { EmployeesPage } from './pages/EmployeesPage';
import { VenuesPage } from './pages/VenuesPage';
import { appBasePath } from './lib/app-paths';

function App() {
  return (
    <BrowserRouter basename={appBasePath || undefined}>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/profile" element={<Navigate to="/" replace />} />
          <Route path="/profile/settings" element={<Navigate to="/" replace />} />
          <Route path="/change-password" element={
            <ProtectedRoute bypassForcePasswordChange><ChangePassword /></ProtectedRoute>
          } />

          <Route path="/" element={<ProtectedRoute><Layout><Dashboard /></Layout></ProtectedRoute>} />
          <Route path="/jobs" element={<ProtectedRoute><Layout><JobsPage /></Layout></ProtectedRoute>} />
          <Route path="/jobs/new" element={<ProtectedRoute><Layout><JobsPage /></Layout></ProtectedRoute>} />
          <Route path="/jobs/:id" element={<ProtectedRoute><Layout><JobsPage /></Layout></ProtectedRoute>} />
          <Route path="/jobs/:id/edit" element={<ProtectedRoute><Layout><JobsPage /></Layout></ProtectedRoute>} />

          <Route path="/analytics" element={<ProtectedRoute><Layout><AnalyticsPage /></Layout></ProtectedRoute>} />
          <Route path="/documents" element={<ProtectedRoute><Layout><DocumentsPage /></Layout></ProtectedRoute>} />
          <Route path="/employees" element={<ProtectedRoute><Layout><EmployeesPage /></Layout></ProtectedRoute>} />
          <Route path="/venues" element={<ProtectedRoute><Layout><VenuesPage /></Layout></ProtectedRoute>} />
          <Route path="/admin/skills" element={<ProtectedRoute><Layout><SkillsPage /></Layout></ProtectedRoute>} />
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  );
}

export default App;
