import { Card, Input, Select, Space, Table, Tag, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { ChangeEvent } from 'react';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useCursorTableState } from '../hooks/useCursorTableState';
import { useAuditLogs } from '../hooks/useAuditLogs';
import type { AuditLogItem } from '../types';

const { Title, Text } = Typography;

const ACTION_OPTIONS = [
  { value: '', label: 'All actions' },
  { value: 'create', label: 'Create' },
  { value: 'update', label: 'Update' },
  { value: 'delete', label: 'Delete' },
];

const RESOURCE_TYPE_OPTIONS = [
  { value: '', label: 'All resources' },
  { value: 'city', label: 'City' },
  { value: 'poi', label: 'POI' },
  { value: 'story', label: 'Story' },
  { value: 'report', label: 'Report' },
];

const ACTION_FILTER_VALUES = ['create', 'update', 'delete'] as const;
const PAGE_SIZE_OPTIONS = [10, 20, 50] as const;

const ACTION_COLORS: Record<string, string> = {
  create: 'green',
  update: 'blue',
  delete: 'red',
};

const METHOD_COLORS: Record<string, string> = {
  GET: 'default',
  POST: 'green',
  PUT: 'blue',
  DELETE: 'red',
  PATCH: 'orange',
};

function PayloadDisplay({ payload }: { payload: unknown }) {
  if (payload == null) {
    return <Text type="secondary">—</Text>;
  }
  return (
    <pre
      style={{ margin: 0, fontSize: 12, maxHeight: 300, overflow: 'auto', whiteSpace: 'pre-wrap' }}
    >
      {typeof payload === 'string' ? payload : JSON.stringify(payload, null, 2)}
    </pre>
  );
}

function toStartOfDayISOString(value: string) {
  return value ? `${value}T00:00:00Z` : '';
}

function toEndOfDayISOString(value: string) {
  return value ? `${value}T23:59:59Z` : '';
}

export default function AuditLogsPage() {
  const [actorId, setActorId] = useState('');
  const [resourceType, setResourceType] = useState('');
  const [createdFromDate, setCreatedFromDate] = useState('');
  const [createdToDate, setCreatedToDate] = useState('');

  const {
    filter: filterAction,
    page,
    pageSize: perPage,
    cursor,
    setFilter,
    setPageAndSize,
    recordNextCursor,
    resetPagination,
  } = useCursorTableState<(typeof ACTION_FILTER_VALUES)[number]>({
    filterKey: 'action',
    filterValues: ACTION_FILTER_VALUES,
    defaultFilter: '',
    defaultPageSize: 20,
    pageSizeOptions: PAGE_SIZE_OPTIONS,
  });

  const { logs } = useAuditLogs({
    actorId,
    action: filterAction,
    resourceType,
    createdFrom: toStartOfDayISOString(createdFromDate),
    createdTo: toEndOfDayISOString(createdToDate),
    cursor,
    limit: perPage,
  });

  const logData = logs.data?.items ?? [];
  const total = useMemo(() => {
    if (!logs.data) {
      return page * perPage;
    }
    if (logs.data.has_more) {
      return page * perPage + 1;
    }
    return (page - 1) * perPage + logData.length;
  }, [page, perPage, logData.length, logs.data]);

  useEffect(() => {
    if (logs.data?.has_more && logs.data.next_cursor) {
      recordNextCursor(logs.data.next_cursor);
    }
  }, [logs.data, recordNextCursor]);

  useEffect(() => {
    if (page > 1 && !logs.isLoading && logData.length === 0) {
      setPageAndSize(Math.max(1, page - 1), perPage);
    }
  }, [page, logData.length, logs.isLoading, setPageAndSize, perPage]);

  const handleResourceTypeChange = useCallback(
    (value: string) => {
      setResourceType(value);
      resetPagination();
    },
    [resetPagination],
  );

  const handleActorIdChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      setActorId(event.target.value);
      resetPagination();
    },
    [resetPagination],
  );

  const handleCreatedFromChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      setCreatedFromDate(event.target.value);
      resetPagination();
    },
    [resetPagination],
  );

  const handleCreatedToChange = useCallback(
    (event: ChangeEvent<HTMLInputElement>) => {
      setCreatedToDate(event.target.value);
      resetPagination();
    },
    [resetPagination],
  );

  const columns: ColumnsType<AuditLogItem> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: 'Actor',
      dataIndex: 'actor_id',
      width: 140,
      ellipsis: true,
      render: (actorId: string) => actorId || <Text type="secondary">—</Text>,
    },
    {
      title: 'Action',
      dataIndex: 'action',
      width: 90,
      render: (action: string) => <Tag color={ACTION_COLORS[action] ?? 'default'}>{action}</Tag>,
    },
    {
      title: 'Resource',
      key: 'resource',
      width: 150,
      render: (_: unknown, record: AuditLogItem) => (
        <Space size={4}>
          <Tag>{record.resource_type}</Tag>
          <Text type="secondary">#{record.resource_id}</Text>
        </Space>
      ),
    },
    {
      title: 'Method',
      dataIndex: 'http_method',
      width: 80,
      render: (method: string) => <Tag color={METHOD_COLORS[method] ?? 'default'}>{method}</Tag>,
    },
    {
      title: 'Path',
      dataIndex: 'request_path',
      ellipsis: true,
      render: (path: string) => <Text code>{path}</Text>,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      width: 90,
      render: (status: string) => (
        <Tag color={status === 'success' ? 'green' : 'red'}>{status}</Tag>
      ),
    },
    {
      title: 'Date',
      dataIndex: 'created_at',
      width: 160,
      render: (date: string) => new Date(date).toLocaleString(),
    },
  ];

  return (
    <div data-testid="audit-logs-page">
      <Title level={2}>Audit Logs</Title>

      <Card>
        <Space style={{ marginBottom: 16, flexWrap: 'wrap' }}>
          <Input
            value={actorId}
            onChange={handleActorIdChange}
            placeholder="Actor ID"
            style={{ width: 180 }}
            data-testid="actor-id-filter"
          />
          <Text>Action:</Text>
          <Select
            value={filterAction}
            onChange={(value) => setFilter(value)}
            options={ACTION_OPTIONS}
            style={{ width: 160 }}
            data-testid="action-filter"
          />
          <Text>Resource:</Text>
          <Select
            value={resourceType}
            onChange={handleResourceTypeChange}
            options={RESOURCE_TYPE_OPTIONS}
            style={{ width: 160 }}
            data-testid="resource-type-filter"
          />
          <Input
            type="date"
            value={createdFromDate}
            onChange={handleCreatedFromChange}
            style={{ width: 160 }}
            data-testid="created-from-filter"
          />
          <Input
            type="date"
            value={createdToDate}
            onChange={handleCreatedToChange}
            style={{ width: 160 }}
            data-testid="created-to-filter"
          />
        </Space>

        {logs.isLoading && <Text data-testid="audit-logs-loading">Loading audit logs...</Text>}

        <Table<AuditLogItem>
          columns={columns}
          dataSource={logData}
          rowKey="id"
          loading={logs.isLoading}
          size="small"
          scroll={{ x: 900 }}
          data-testid="audit-logs-table"
          expandable={{
            expandedRowRender: (record) => (
              <div style={{ padding: '8px 0' }} data-testid={`audit-detail-${record.id}`}>
                <Space direction="vertical" size={8} style={{ width: '100%' }}>
                  {record.trace_id && (
                    <div>
                      <Text strong>Trace ID: </Text>
                      <Text code>{record.trace_id}</Text>
                    </div>
                  )}
                  <div>
                    <Text strong>Request: </Text>
                    <Text code>
                      {record.http_method} {record.request_path}
                    </Text>
                  </div>
                  <div>
                    <Text strong>Timestamp: </Text>
                    <Text>{new Date(record.created_at).toISOString()}</Text>
                  </div>
                  <div>
                    <Text strong>Payload: </Text>
                    <PayloadDisplay payload={record.payload} />
                  </div>
                </Space>
              </div>
            ),
            rowExpandable: () => true,
          }}
          pagination={{
            current: page,
            pageSize: perPage,
            total,
            showSizeChanger: true,
            pageSizeOptions: ['10', '20', '50'],
            showTotal: (t) => `Total ${t} entries`,
            onChange: (p, ps) => {
              setPageAndSize(p, ps);
            },
          }}
          locale={{
            emptyText: logs.isLoading
              ? 'Loading audit logs...'
              : 'No audit logs match the current filters.',
          }}
        />
      </Card>
    </div>
  );
}
