import {
  EnvironmentOutlined,
  GlobalOutlined,
  ReadOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import { Card, Col, Row, Spin, Statistic, Table, Typography } from 'antd';
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

export default function DashboardPage() {
  const { stats, isLoading, cities } = useDashboardStats();

  return (
    <div>
      <Title level={2}>Dashboard</Title>

      <Spin spinning={isLoading}>
        <Row gutter={[16, 16]}>
          <Col xs={24} sm={12} lg={6}>
            <Card>
              <Statistic
                title="Cities"
                value={stats.citiesCount}
                prefix={<GlobalOutlined />}
                valueStyle={{ color: '#1677ff' }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card>
              <Statistic
                title="Points of Interest"
                value={stats.poisCount}
                prefix={<EnvironmentOutlined />}
                valueStyle={{ color: '#52c41a' }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card>
              <Statistic
                title="Stories"
                value={stats.storiesCount}
                prefix={<ReadOutlined />}
                valueStyle={{ color: '#722ed1' }}
              />
            </Card>
          </Col>
          <Col xs={24} sm={12} lg={6}>
            <Card>
              <Statistic
                title="Reports"
                value={stats.reportsCount}
                prefix={<WarningOutlined />}
                valueStyle={{ color: stats.reportsCount > 0 ? '#fa541c' : '#8c8c8c' }}
              />
            </Card>
          </Col>
        </Row>
      </Spin>

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
      />
    </div>
  );
}
