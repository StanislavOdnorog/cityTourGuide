import {
  AuditOutlined,
  DashboardOutlined,
  EnvironmentOutlined,
  GlobalOutlined,
  LogoutOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Badge, ConfigProvider, App as AntApp, Layout, Menu, Button, Spin, theme } from 'antd';
import { useEffect } from 'react';
import { BrowserRouter, Routes, Route, Navigate, useNavigate, useLocation } from 'react-router-dom';
import { ErrorBoundary } from './components/ErrorBoundary';
import { useNewReportsCount } from './hooks/useReports';
import { registerQueryClient, resetAuth } from './lib/authReset';
import CitiesPage from './pages/CitiesPage';
import DashboardPage from './pages/DashboardPage';
import LoginPage from './pages/LoginPage';
import NotFoundPage from './pages/NotFoundPage';
import POIDetailPage from './pages/POIDetailPage';
import POIMapPage from './pages/POIMapPage';
import AuditLogsPage from './pages/AuditLogsPage';
import ReportsPage from './pages/ReportsPage';
import { useAuthStore } from './store/authStore';

const { Header, Content, Sider } = Layout;

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
});

registerQueryClient(queryClient);

function AppLayout() {
  const navigate = useNavigate();
  const location = useLocation();
  const user = useAuthStore((s) => s.user);
  const { data: newReportsCount = 0 } = useNewReportsCount();

  const menuItems = [
    {
      key: '/',
      icon: <DashboardOutlined />,
      label: <span data-testid="nav-dashboard">Dashboard</span>,
    },
    {
      key: '/cities',
      icon: <GlobalOutlined />,
      label: <span data-testid="nav-cities">Cities</span>,
    },
    {
      key: '/poi-map',
      icon: <EnvironmentOutlined />,
      label: <span data-testid="nav-poi-map">POI Map</span>,
    },
    {
      key: '/reports',
      icon: <WarningOutlined />,
      label: (
        <Badge count={newReportsCount} offset={[16, 0]} size="small">
          <span data-testid="nav-reports">Reports</span>
        </Badge>
      ),
    },
    {
      key: '/audit-logs',
      icon: <AuditOutlined />,
      label: <span data-testid="nav-audit-logs">Audit Logs</span>,
    },
  ];

  const handleLogout = () => {
    resetAuth();
    navigate('/login', { replace: true });
  };

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sider breakpoint="lg" collapsedWidth={80}>
        <div
          style={{ height: 32, margin: 16, color: '#fff', fontWeight: 700, textAlign: 'center' }}
        >
          CSG Admin
        </div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[location.pathname]}
          items={menuItems}
          onClick={({ key }) => navigate(key)}
          data-testid="sidebar-nav"
        />
      </Sider>
      <Layout>
        <Header
          style={{
            padding: '0 24px',
            background: '#fff',
            display: 'flex',
            justifyContent: 'flex-end',
            alignItems: 'center',
          }}
        >
          {user?.email && <span style={{ marginRight: 16, color: '#666' }}>{user.email}</span>}
          <Button icon={<LogoutOutlined />} onClick={handleLogout} data-testid="logout-button">
            Logout
          </Button>
        </Header>
        <Content style={{ margin: 24 }}>
          <Routes>
            <Route path="/" element={<DashboardPage />} />
            <Route path="/cities" element={<CitiesPage />} />
            <Route path="/poi-map" element={<POIMapPage />} />
            <Route path="/pois/:id" element={<POIDetailPage />} />
            <Route path="/reports" element={<ReportsPage />} />
            <Route path="/audit-logs" element={<AuditLogsPage />} />
            <Route path="*" element={<NotFoundPage />} />
          </Routes>
        </Content>
      </Layout>
    </Layout>
  );
}

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const isAuthenticated = useAuthStore((s) => s.isAuthenticated);
  const isHydrated = useAuthStore((s) => s.isHydrated);
  const location = useLocation();

  if (!isHydrated) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '100vh' }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" state={{ from: location.pathname }} replace />;
  }

  return <>{children}</>;
}

function AuthInit({ children }: { children: React.ReactNode }) {
  const hydrateFromStorage = useAuthStore((s) => s.hydrateFromStorage);

  useEffect(() => {
    hydrateFromStorage();
  }, [hydrateFromStorage]);

  return <>{children}</>;
}

export default function App() {
  return (
    <ErrorBoundary>
      <ConfigProvider theme={{ algorithm: theme.defaultAlgorithm }}>
        <AntApp>
          <QueryClientProvider client={queryClient}>
            <BrowserRouter>
              <AuthInit>
                <Routes>
                  <Route path="/login" element={<LoginPage />} />
                  <Route
                    path="/*"
                    element={
                      <ProtectedRoute>
                        <AppLayout />
                      </ProtectedRoute>
                    }
                  />
                </Routes>
              </AuthInit>
            </BrowserRouter>
          </QueryClientProvider>
        </AntApp>
      </ConfigProvider>
    </ErrorBoundary>
  );
}
