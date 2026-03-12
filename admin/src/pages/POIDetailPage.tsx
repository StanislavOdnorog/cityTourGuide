import {
  ArrowLeftOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import {
  App,
  Badge,
  Button,
  Card,
  Col,
  Descriptions,
  Empty,
  Row,
  Space,
  Spin,
  Switch,
  Table,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { usePOIDetail } from '../hooks';
import type { InflationJob, POIStatus, Report, Story, StoryStatus } from '../types';

const { Title, Text, Paragraph } = Typography;

const STATUS_COLORS: Record<POIStatus, string> = {
  active: 'green',
  disabled: 'red',
  pending_review: 'orange',
};

const STORY_STATUS_COLORS: Record<StoryStatus, string> = {
  active: 'green',
  disabled: 'red',
  reported: 'volcano',
  pending_review: 'orange',
};

const LAYER_TYPE_COLORS: Record<string, string> = {
  atmosphere: 'blue',
  human_story: 'purple',
  hidden_detail: 'cyan',
  time_shift: 'magenta',
  general: 'default',
};

function AudioPreview({ url }: { url: string }) {
  const audioRef = useRef<HTMLAudioElement>(null);
  const [playing, setPlaying] = useState(false);

  const toggle = () => {
    if (!audioRef.current) return;
    if (playing) {
      audioRef.current.pause();
      setPlaying(false);
    } else {
      audioRef.current.play();
      setPlaying(true);
    }
  };

  return (
    <>
      <audio
        ref={audioRef}
        src={url}
        onEnded={() => setPlaying(false)}
        onPause={() => setPlaying(false)}
      />
      <Button
        type="text"
        size="small"
        icon={playing ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
        onClick={toggle}
      >
        {playing ? 'Pause' : 'Play'}
      </Button>
    </>
  );
}

export default function POIDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { message } = App.useApp();
  const poiId = id ? parseInt(id, 10) : null;

  const {
    poi: { data: poi, isLoading: poiLoading },
    stories: { data: stories = [], isLoading: storiesLoading },
    reports: { data: reports = [], isLoading: reportsLoading },
    inflationJobs: { data: jobs = [], isLoading: jobsLoading },
    updatePOI,
    toggleStoryStatus,
    triggerInflation,
  } = usePOIDetail(poiId);

  if (poiLoading) {
    return (
      <div style={{ textAlign: 'center', padding: 48 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (!poi) {
    return <Empty description="POI not found" />;
  }

  const handleTogglePOI = (checked: boolean) => {
    const newStatus: POIStatus = checked ? 'active' : 'disabled';
    updatePOI.mutate({ status: newStatus } as Partial<typeof poi>, {
      onSuccess: () => message.success(`POI ${checked ? 'enabled' : 'disabled'}`),
      onError: () => message.error('Failed to update POI status'),
    });
  };

  const handleToggleStory = (storyId: number, currentStatus: StoryStatus) => {
    toggleStoryStatus.mutate(
      { storyId, currentStatus },
      {
        onSuccess: (updated) =>
          message.success(`Story ${updated.status === 'active' ? 'enabled' : 'disabled'}`),
        onError: () => message.error('Failed to update story status'),
      },
    );
  };

  const handleTriggerInflation = () => {
    triggerInflation.mutate(undefined, {
      onSuccess: () => message.success('Inflation job created'),
      onError: (err: unknown) => {
        const errorMsg =
          err && typeof err === 'object' && 'response' in err
            ? ((err as { response?: { data?: { error?: string } } }).response?.data?.error ??
              'Failed to trigger inflation')
            : 'Failed to trigger inflation';
        message.error(errorMsg);
      },
    });
  };

  const storyColumns: ColumnsType<Story> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: 'Language',
      dataIndex: 'language',
      width: 80,
      render: (lang: string) => <Tag>{lang.toUpperCase()}</Tag>,
    },
    {
      title: 'Layer Type',
      dataIndex: 'layer_type',
      width: 120,
      render: (type: string) => <Tag color={LAYER_TYPE_COLORS[type] ?? 'default'}>{type}</Tag>,
    },
    {
      title: 'Text',
      dataIndex: 'text',
      ellipsis: true,
      render: (text: string) => (
        <Paragraph ellipsis={{ rows: 2, expandable: 'collapsible' }} style={{ margin: 0 }}>
          {text}
        </Paragraph>
      ),
    },
    {
      title: 'Audio',
      dataIndex: 'audio_url',
      width: 100,
      render: (url: string | null) =>
        url ? <AudioPreview url={url} /> : <Text type="secondary">No audio</Text>,
    },
    {
      title: 'Duration',
      dataIndex: 'duration_sec',
      width: 80,
      render: (sec: number | null) => (sec ? `${sec}s` : '-'),
    },
    {
      title: 'Confidence',
      dataIndex: 'confidence',
      width: 90,
      render: (val: number) => <Text>{val}%</Text>,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      width: 120,
      render: (status: StoryStatus) => (
        <Tag color={STORY_STATUS_COLORS[status]}>{status}</Tag>
      ),
    },
    {
      title: 'Inflation',
      dataIndex: 'is_inflation',
      width: 80,
      render: (val: boolean) => (val ? <Tag color="geekblue">AI</Tag> : null),
    },
    {
      title: 'Enable',
      width: 80,
      render: (_: unknown, record: Story) => (
        <Switch
          size="small"
          checked={record.status === 'active'}
          loading={toggleStoryStatus.isPending}
          onChange={() => handleToggleStory(record.id, record.status)}
        />
      ),
    },
  ];

  const reportColumns: ColumnsType<Report> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: 'Story ID',
      dataIndex: 'story_id',
      width: 80,
    },
    {
      title: 'Type',
      dataIndex: 'type',
      width: 150,
      render: (type: string) => <Tag color="red">{type.replace(/_/g, ' ')}</Tag>,
    },
    {
      title: 'Comment',
      dataIndex: 'comment',
      ellipsis: true,
      render: (comment: string | null) => comment ?? <Text type="secondary">-</Text>,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      width: 100,
      render: (status: string) => {
        const colors: Record<string, string> = {
          new: 'red',
          reviewed: 'blue',
          resolved: 'green',
          dismissed: 'default',
        };
        return <Tag color={colors[status] ?? 'default'}>{status}</Tag>;
      },
    },
    {
      title: 'Date',
      dataIndex: 'created_at',
      width: 140,
      render: (date: string) => new Date(date).toLocaleDateString(),
    },
  ];

  const jobColumns: ColumnsType<InflationJob> = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 60,
    },
    {
      title: 'Status',
      dataIndex: 'status',
      width: 100,
      render: (status: string) => {
        const colors: Record<string, string> = {
          pending: 'orange',
          running: 'blue',
          completed: 'green',
          failed: 'red',
        };
        return <Tag color={colors[status] ?? 'default'}>{status}</Tag>;
      },
    },
    {
      title: 'Trigger',
      dataIndex: 'trigger_type',
      width: 120,
      render: (type: string) => type.replace(/_/g, ' '),
    },
    {
      title: 'Segments',
      width: 100,
      render: (_: unknown, record: InflationJob) =>
        `${record.segments_count} / ${record.max_segments}`,
    },
    {
      title: 'Created',
      dataIndex: 'created_at',
      width: 140,
      render: (date: string) => new Date(date).toLocaleDateString(),
    },
    {
      title: 'Error',
      dataIndex: 'error_log',
      ellipsis: true,
      render: (log: string | null) =>
        log ? (
          <Text type="danger" ellipsis>
            {log}
          </Text>
        ) : (
          '-'
        ),
    },
  ];

  const newReportsCount = reports.filter((r) => r.status === 'new').length;

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(-1)}>
          Back
        </Button>
      </Space>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={16}>
          <Card
            title={
              <Space>
                <Title level={4} style={{ margin: 0 }}>
                  {poi.name}
                </Title>
                <Tag color={STATUS_COLORS[poi.status]}>{poi.status}</Tag>
              </Space>
            }
            extra={
              <Space>
                <Text type="secondary">Enable:</Text>
                <Switch
                  checked={poi.status === 'active'}
                  loading={updatePOI.isPending}
                  onChange={handleTogglePOI}
                />
              </Space>
            }
          >
            <Descriptions column={{ xs: 1, sm: 2 }} size="small">
              {poi.name_ru && (
                <Descriptions.Item label="Name (RU)">{poi.name_ru}</Descriptions.Item>
              )}
              <Descriptions.Item label="Type">
                <Tag>{poi.type}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="Coordinates">
                {poi.lat.toFixed(5)}, {poi.lng.toFixed(5)}
              </Descriptions.Item>
              <Descriptions.Item label="Interest Score">{poi.interest_score}</Descriptions.Item>
              {poi.address && (
                <Descriptions.Item label="Address">{poi.address}</Descriptions.Item>
              )}
              <Descriptions.Item label="City ID">{poi.city_id}</Descriptions.Item>
              <Descriptions.Item label="Created">
                {new Date(poi.created_at).toLocaleDateString()}
              </Descriptions.Item>
            </Descriptions>
          </Card>
        </Col>

        <Col xs={24} lg={8}>
          <Card title="Actions">
            <Space direction="vertical" style={{ width: '100%' }}>
              <Button
                type="primary"
                icon={<ThunderboltOutlined />}
                loading={triggerInflation.isPending}
                onClick={handleTriggerInflation}
                block
              >
                Trigger Inflation
              </Button>
              <Text type="secondary">
                Generate additional story segments via AI. Max 3 per POI.
              </Text>
            </Space>
          </Card>
        </Col>
      </Row>

      <Card style={{ marginTop: 16 }}>
        <Tabs
          items={[
            {
              key: 'stories',
              label: `Stories (${stories.length})`,
              children: (
                <Table<Story>
                  columns={storyColumns}
                  dataSource={stories}
                  rowKey="id"
                  loading={storiesLoading}
                  size="small"
                  pagination={{ pageSize: 10 }}
                  scroll={{ x: 900 }}
                />
              ),
            },
            {
              key: 'reports',
              label: (
                <Badge count={newReportsCount} offset={[10, 0]} size="small">
                  Reports ({reports.length})
                </Badge>
              ),
              children: (
                <Table<Report>
                  columns={reportColumns}
                  dataSource={reports}
                  rowKey="id"
                  loading={reportsLoading}
                  size="small"
                  pagination={{ pageSize: 10 }}
                  locale={{ emptyText: 'No reports for this POI' }}
                />
              ),
            },
            {
              key: 'inflation',
              label: `Inflation Jobs (${jobs.length})`,
              children: (
                <Table<InflationJob>
                  columns={jobColumns}
                  dataSource={jobs}
                  rowKey="id"
                  loading={jobsLoading}
                  size="small"
                  pagination={{ pageSize: 10 }}
                  locale={{ emptyText: 'No inflation jobs' }}
                />
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
}
