import { CheckCircleOutlined, CloseCircleOutlined, StopOutlined } from '@ant-design/icons';
import { App, Button, Card, Select, Space, Table, Tag, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useReports } from '../hooks/useReports';
import type { Report, ReportStatus, ReportType } from '../types';

const { Title, Text } = Typography;

const REPORT_STATUS_COLORS: Record<ReportStatus, string> = {
  new: 'red',
  reviewed: 'blue',
  resolved: 'green',
  dismissed: 'default',
};

const REPORT_TYPE_COLORS: Record<ReportType, string> = {
  wrong_location: 'orange',
  wrong_fact: 'volcano',
  inappropriate_content: 'red',
};

const STATUS_OPTIONS = [
  { value: '', label: 'All statuses' },
  { value: 'new', label: 'New' },
  { value: 'reviewed', label: 'Reviewed' },
  { value: 'resolved', label: 'Resolved' },
  { value: 'dismissed', label: 'Dismissed' },
];

export default function ReportsPage() {
  const navigate = useNavigate();
  const { message } = App.useApp();

  const [filterStatus, setFilterStatus] = useState<ReportStatus | ''>('');
  const [page, setPage] = useState(1);
  const [perPage, setPerPage] = useState(20);

  const { reports, updateStatus, disableStory } = useReports({
    status: filterStatus,
    page,
    perPage,
  });

  const reportData = reports.data?.data ?? [];
  const total = reports.data?.meta.total ?? 0;

  const handleUpdateStatus = (reportId: number, newStatus: ReportStatus) => {
    updateStatus.mutate(
      { reportId, newStatus },
      {
        onSuccess: () => message.success(`Report ${newStatus}`),
        onError: () => message.error('Failed to update report status'),
      },
    );
  };

  const handleDisableStory = (report: Report) => {
    disableStory.mutate(report.story_id, {
      onSuccess: () => {
        message.success(`Story #${report.story_id} disabled`);
        // Also resolve the report
        updateStatus.mutate(
          { reportId: report.id, newStatus: 'resolved' },
          {
            onSuccess: () => message.success(`Report #${report.id} resolved`),
            onError: () => message.error('Failed to resolve report'),
          },
        );
      },
      onError: () => message.error('Failed to disable story'),
    });
  };

  const columns: ColumnsType<Report> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
      sorter: (a, b) => a.id - b.id,
    },
    {
      title: 'Story ID',
      dataIndex: 'story_id',
      width: 90,
      render: (storyId: number) => (
        <Button type="link" size="small" onClick={() => navigate(`/pois/${storyId}`)}>
          #{storyId}
        </Button>
      ),
    },
    {
      title: 'Type',
      dataIndex: 'type',
      width: 170,
      filters: [
        { text: 'Wrong Location', value: 'wrong_location' },
        { text: 'Wrong Fact', value: 'wrong_fact' },
        { text: 'Inappropriate', value: 'inappropriate_content' },
      ],
      onFilter: (value, record) => record.type === value,
      render: (type: ReportType) => (
        <Tag color={REPORT_TYPE_COLORS[type]}>{type.replace(/_/g, ' ')}</Tag>
      ),
    },
    {
      title: 'Comment',
      dataIndex: 'comment',
      ellipsis: true,
      render: (comment: string | null) =>
        comment ? (
          <Tooltip title={comment}>
            <Text>{comment}</Text>
          </Tooltip>
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
    {
      title: 'Status',
      dataIndex: 'status',
      width: 110,
      render: (status: ReportStatus) => (
        <Tag color={REPORT_STATUS_COLORS[status]}>{status}</Tag>
      ),
    },
    {
      title: 'Date',
      dataIndex: 'created_at',
      width: 140,
      sorter: (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
      defaultSortOrder: 'descend',
      render: (date: string) => new Date(date).toLocaleDateString(),
    },
    {
      title: 'Actions',
      width: 240,
      render: (_: unknown, record: Report) => {
        if (record.status === 'resolved' || record.status === 'dismissed') {
          return <Text type="secondary">Closed</Text>;
        }

        return (
          <Space size="small">
            <Tooltip title="Dismiss report">
              <Button
                size="small"
                icon={<CloseCircleOutlined />}
                loading={updateStatus.isPending}
                onClick={() => handleUpdateStatus(record.id, 'dismissed')}
              >
                Dismiss
              </Button>
            </Tooltip>
            <Tooltip title="Disable the reported story and resolve">
              <Button
                size="small"
                danger
                icon={<StopOutlined />}
                loading={disableStory.isPending}
                onClick={() => handleDisableStory(record)}
              >
                Disable Story
              </Button>
            </Tooltip>
            <Tooltip title="Mark as resolved">
              <Button
                size="small"
                type="primary"
                icon={<CheckCircleOutlined />}
                loading={updateStatus.isPending}
                onClick={() => handleUpdateStatus(record.id, 'resolved')}
              >
                Resolve
              </Button>
            </Tooltip>
          </Space>
        );
      },
    },
  ];

  return (
    <div>
      <Title level={2}>Reports</Title>

      <Card>
        <Space style={{ marginBottom: 16 }}>
          <Text>Status:</Text>
          <Select
            value={filterStatus}
            onChange={(value) => {
              setFilterStatus(value);
              setPage(1);
            }}
            options={STATUS_OPTIONS}
            style={{ width: 160 }}
          />
        </Space>

        <Table<Report>
          columns={columns}
          dataSource={reportData}
          rowKey="id"
          loading={reports.isLoading}
          size="small"
          scroll={{ x: 900 }}
          pagination={{
            current: page,
            pageSize: perPage,
            total,
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50'],
            showTotal: (t) => `Total ${t} reports`,
            onChange: (p, ps) => {
              setPage(p);
              setPerPage(ps);
            },
          }}
        />
      </Card>
    </div>
  );
}
