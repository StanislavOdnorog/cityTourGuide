import { CheckCircleOutlined, CloseCircleOutlined, StopOutlined } from '@ant-design/icons';
import { App, Button, Card, Select, Space, Table, Tag, Tooltip, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useEffect, useMemo } from 'react';
import { Link } from 'react-router-dom';
import { useCursorTableState } from '../hooks/useCursorTableState';
import { useReports } from '../hooks/useReports';
import type { AdminReportListItem, ReportStatus, ReportType } from '../types';

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

const FILTER_VALUES = ['new', 'reviewed', 'resolved', 'dismissed'] as const;
const PAGE_SIZE_OPTIONS = [10, 20, 50] as const;

export default function ReportsPage() {
  const { message } = App.useApp();

  const {
    filter: filterStatus,
    page,
    pageSize: perPage,
    cursor,
    setFilter,
    setPageAndSize,
    recordNextCursor,
  } = useCursorTableState<ReportStatus>({
    filterKey: 'status',
    filterValues: FILTER_VALUES,
    defaultFilter: '',
    defaultPageSize: 20,
    pageSizeOptions: PAGE_SIZE_OPTIONS,
  });

  const { reports, updateStatus, disableStory } = useReports({
    status: filterStatus,
    cursor,
    limit: perPage,
  });

  const reportData = reports.data?.items ?? [];
  const total = useMemo(() => {
    if (!reports.data) {
      return page * perPage;
    }

    if (reports.data.has_more) {
      return page * perPage + 1;
    }

    return (page - 1) * perPage + reportData.length;
  }, [page, perPage, reportData.length, reports.data]);

  useEffect(() => {
    if (reports.data?.has_more && reports.data.next_cursor) {
      recordNextCursor(reports.data.next_cursor);
    }
  }, [reports.data, recordNextCursor]);

  useEffect(() => {
    if (page > 1 && !reports.isLoading && reportData.length === 0) {
      setPageAndSize(Math.max(1, page - 1), perPage);
    }
  }, [page, reportData.length, reports.isLoading, setPageAndSize, perPage]);

  const handleUpdateStatus = (reportId: number, newStatus: ReportStatus) => {
    updateStatus.mutate(
      { reportId, newStatus },
      {
        onSuccess: () => message.success(`Report ${newStatus}`),
      },
    );
  };

  const handleDisableStory = (report: AdminReportListItem) => {
    disableStory.mutate(report.id, {
      onSuccess: () => {
        message.success('Moderation completed');
      },
    });
  };

  const columns: ColumnsType<AdminReportListItem> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
      render: (id: number) => <span data-testid={`report-row-${id}`}>{id}</span>,
    },
    {
      title: 'POI',
      key: 'poi',
      width: 180,
      render: (_: unknown, record: AdminReportListItem) => {
        if (record.poi_id != null && record.poi_name) {
          return (
            <Link to={`/pois/${record.poi_id}?storyId=${record.story_id}`}>
              {record.poi_name}
            </Link>
          );
        }
        return <Text type="secondary">Unknown POI</Text>;
      },
    },
    {
      title: 'Story',
      key: 'story_info',
      width: 130,
      render: (_: unknown, record: AdminReportListItem) => (
        <Space direction="vertical" size={0}>
          <Text type="secondary">#{record.story_id}</Text>
          {record.story_language && (
            <Tag>{record.story_language.toUpperCase()}</Tag>
          )}
          {record.story_status && record.story_status !== 'active' && (
            <Tag color={record.story_status === 'disabled' ? 'red' : 'orange'}>
              {record.story_status}
            </Tag>
          )}
        </Space>
      ),
    },
    {
      title: 'Type',
      dataIndex: 'type',
      width: 170,
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
      render: (date: string) => new Date(date).toLocaleDateString(),
    },
    {
      title: 'Actions',
      width: 240,
      render: (_: unknown, record: AdminReportListItem) => {
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
    <div data-testid="reports-page">
      <Title level={2}>Reports</Title>

      <Card>
        <Space style={{ marginBottom: 16 }}>
          <Text>Status:</Text>
          <Select
            value={filterStatus}
            onChange={(value) => setFilter(value)}
            options={STATUS_OPTIONS}
            style={{ width: 160 }}
            data-testid="reports-status-filter"
          />
        </Space>

        <Table<AdminReportListItem>
          columns={columns}
          dataSource={reportData}
          rowKey="id"
          loading={reports.isLoading}
          size="small"
          scroll={{ x: 900 }}
          data-testid="reports-table"
          pagination={{
            current: page,
            pageSize: perPage,
            total,
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50'],
            showTotal: (t) => `Total ${t} reports`,
            onChange: (p, ps) => {
              setPageAndSize(p, ps);
            },
          }}
        />
      </Card>
    </div>
  );
}
