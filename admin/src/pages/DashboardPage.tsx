import {
  AlertOutlined,
  EnvironmentOutlined,
  GlobalOutlined,
  ReadOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import { Alert, Card, Col, Progress, Row, Skeleton, Statistic, Table, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useDashboardStats } from '../hooks';
import type { City } from '../types';

const { Title } = Typography;

const cityColumns: ColumnsType<City> = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
  { title: 'Name', dataIndex: 'name', key: 'name' },
  { title: 'Country', dataIndex: 'country', key: 'country' },
  {
    title: 'Active',
    dataIndex: 'is_active',
    key: 'is_active',
    render: (v: boolean) => (v ? 'Yes' : 'No'),
    width: 80,
  },
  {
    title: 'Radius (km)',
    dataIndex: 'radius_km',
    key: 'radius_km',
    width: 110,
    render: (v: number) => v.toFixed(1),
  },
];

function StatCard({
  testId,
  title,
  value,
  prefix,
  color,
  isLoading,
}: {
  testId: string;
  title: string;
  value: number | string;
  prefix: React.ReactNode;
  color: string;
  isLoading: boolean;
}) {
  return (
    <Card data-testid={testId}>
      {isLoading ? (
        <Skeleton active paragraph={false} />
      ) : (
        <Statistic title={title} value={value} prefix={prefix} valueStyle={{ color }} />
      )}
    </Card>
  );
}

export default function DashboardPage() {
  const { stats, isLoading, isError, error, cities } = useDashboardStats();

  const totalReports = stats?.reportsCount ?? 0;
  const newReports = stats?.newReportsCount ?? 0;
  const resolvedReports = totalReports - newReports;
  const reportResolvedPct = totalReports > 0 ? Math.round((resolvedReports / totalReports) * 100) : 100;

  return (
    <div data-testid="dashboard-page">
      <Title level={2}>Dashboard</Title>

      {isError && (
        <Alert
          type="error"
          message="Failed to load dashboard"
          description={error?.message ?? 'An unexpected error occurred.'}
          showIcon
          style={{ marginBottom: 16 }}
        />
      )}

      <Row gutter={[16, 16]}>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            testId="stat-cities"
            title="Cities"
            value={stats?.citiesCount ?? '-'}
            prefix={<GlobalOutlined />}
            color="#1677ff"
            isLoading={isLoading}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            testId="stat-pois"
            title="Points of Interest"
            value={stats?.poisCount ?? '-'}
            prefix={<EnvironmentOutlined />}
            color="#52c41a"
            isLoading={isLoading}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            testId="stat-stories"
            title="Stories"
            value={stats?.storiesCount ?? '-'}
            prefix={<ReadOutlined />}
            color="#722ed1"
            isLoading={isLoading}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            testId="stat-reports"
            title="Reports"
            value={stats?.reportsCount ?? '-'}
            prefix={<WarningOutlined />}
            color={stats && stats.reportsCount > 0 ? '#fa541c' : '#8c8c8c'}
            isLoading={isLoading}
          />
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <StatCard
            testId="stat-new-reports"
            title="New Reports"
            value={stats?.newReportsCount ?? '-'}
            prefix={<AlertOutlined />}
            color={stats && stats.newReportsCount > 0 ? '#fa8c16' : '#8c8c8c'}
            isLoading={isLoading}
          />
        </Col>
      </Row>

      {!isLoading && stats && (
        <Row gutter={[16, 16]} style={{ marginTop: 24 }}>
          <Col xs={24} sm={12}>
            <Card title="Report Resolution" data-testid="report-resolution-card">
              <Progress
                percent={reportResolvedPct}
                format={() => `${resolvedReports} / ${totalReports} resolved`}
                status={reportResolvedPct === 100 ? 'success' : 'active'}
              />
            </Card>
          </Col>
        </Row>
      )}

      <Title level={4} style={{ marginTop: 32 }}>
        Cities
      </Title>
      <Table<City>
        columns={cityColumns}
        dataSource={cities}
        rowKey="id"
        loading={isLoading}
        pagination={false}
        size="small"
        locale={{ emptyText: isLoading ? ' ' : 'No cities found' }}
        data-testid="cities-table"
      />
    </div>
  );
}
