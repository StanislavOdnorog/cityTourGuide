import { useEffect, useMemo, useState } from 'react';
import {
  Table,
  Button,
  Tag,
  Space,
  Select,
  Drawer,
  Form,
  Input,
  InputNumber,
  Switch,
  Modal,
  App,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  UndoOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons';
import { MapContainer, TileLayer, Circle, useMap } from 'react-leaflet';
import type { ColumnsType } from 'antd/es/table';
import { useCities } from '../hooks';
import { useCityManagement } from '../hooks';
import { useCursorTableState } from '../hooks/useCursorTableState';
import type { City } from '../types';
import 'leaflet/dist/leaflet.css';

const PAGE_SIZE_OPTIONS = [10, 20, 50] as const;

function MapUpdater({ lat, lng, zoom }: { lat: number; lng: number; zoom: number }) {
  const map = useMap();
  map.setView([lat, lng], zoom);
  return null;
}

export default function CitiesPage() {
  const { message } = App.useApp();

  const {
    page,
    pageSize: perPage,
    cursor,
    setPageAndSize,
    recordNextCursor,
    resetPagination,
  } = useCursorTableState({
    filterKey: '_unused',
    filterValues: [] as const,
    defaultFilter: '',
    defaultPageSize: 20,
    pageSizeOptions: PAGE_SIZE_OPTIONS,
  });

  const { data: citiesData, isLoading } = useCities({
    cursor,
    limit: perPage,
    includeDeleted: true,
  });
  const { createCity, updateCity, deleteCity, restoreCity, toUpdateRequest } = useCityManagement();

  const cityItems = citiesData?.items ?? [];

  const total = useMemo(() => {
    if (!citiesData) return page * perPage;
    if (citiesData.has_more) return page * perPage + 1;
    return (page - 1) * perPage + cityItems.length;
  }, [page, perPage, cityItems.length, citiesData]);

  useEffect(() => {
    if (citiesData?.has_more && citiesData.next_cursor) {
      recordNextCursor(citiesData.next_cursor);
    }
  }, [citiesData, recordNextCursor]);

  useEffect(() => {
    if (page > 1 && !isLoading && cityItems.length === 0) {
      setPageAndSize(Math.max(1, page - 1), perPage);
    }
  }, [page, cityItems.length, isLoading, setPageAndSize, perPage]);

  const [searchText, setSearchText] = useState('');
  const [visibilityFilter, setVisibilityFilter] = useState<'all' | 'visible' | 'deleted'>('all');
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editingCity, setEditingCity] = useState<City | null>(null);
  const [form] = Form.useForm();

  const isDeleted = (city: City) => !!city.deleted_at;

  // Client-side search scoped to the loaded page only
  const displayedCities = useMemo(() => {
    return cityItems.filter((city) => {
      const matchesSearch =
        !searchText || city.name.toLowerCase().includes(searchText.toLowerCase());
      if (!matchesSearch) {
        return false;
      }

      if (visibilityFilter === 'deleted') {
        return isDeleted(city);
      }
      if (visibilityFilter === 'visible') {
        return !isDeleted(city);
      }
      return true;
    });
  }, [cityItems, searchText, visibilityFilter]);

  const openCreateDrawer = () => {
    setEditingCity(null);
    form.resetFields();
    form.setFieldsValue({
      is_active: true,
      radius_km: 10,
      download_size_mb: 0,
      center_lat: 0,
      center_lng: 0,
    });
    setDrawerOpen(true);
  };

  const openEditDrawer = (city: City) => {
    setEditingCity(city);
    form.setFieldsValue({
      name: city.name,
      name_ru: city.name_ru,
      country: city.country,
      center_lat: city.center_lat,
      center_lng: city.center_lng,
      radius_km: city.radius_km,
      is_active: city.is_active,
      download_size_mb: city.download_size_mb,
    });
    setDrawerOpen(true);
  };

  const handleToggleActive = (city: City) => {
    const newStatus = !city.is_active;
    const action = newStatus ? 'activate' : 'deactivate';

    Modal.confirm({
      title: `${newStatus ? 'Activate' : 'Deactivate'} "${city.name}"?`,
      icon: <ExclamationCircleOutlined />,
      content: !newStatus
        ? 'Deactivating this city will hide all POIs within it from the mobile app.'
        : `This will make "${city.name}" visible in the mobile app again.`,
      okText: `Yes, ${action}`,
      okType: newStatus ? 'primary' : 'danger',
      onOk: () => {
        const body = { ...toUpdateRequest(city), is_active: newStatus };
        updateCity.mutate(
          { id: city.id, body },
          {
            onSuccess: () => message.success(`City ${action}d successfully`),
          },
        );
      },
    });
  };

  const handleDelete = (city: City) => {
    Modal.confirm({
      title: `Delete "${city.name}"?`,
      icon: <ExclamationCircleOutlined />,
      content: 'The city will be soft-deleted and hidden from the mobile app. You can restore it later.',
      okText: 'Yes, delete',
      okType: 'danger',
      onOk: () => {
        deleteCity.mutate(city.id, {
          onSuccess: () => message.success(`City "${city.name}" deleted`),
        });
      },
    });
  };

  const handleRestore = (city: City) => {
    Modal.confirm({
      title: `Restore "${city.name}"?`,
      icon: <ExclamationCircleOutlined />,
      content: 'This will restore the city. It will become visible again based on its active status.',
      okText: 'Yes, restore',
      onOk: () => {
        restoreCity.mutate(city.id, {
          onSuccess: () => message.success(`City "${city.name}" restored`),
        });
      },
    });
  };

  const handleSubmit = async () => {
    try {
      const values = await form.validateFields();
      const body = {
        name: values.name,
        name_ru: values.name_ru || undefined,
        country: values.country,
        center_lat: values.center_lat,
        center_lng: values.center_lng,
        radius_km: values.radius_km,
        is_active: values.is_active ?? true,
        download_size_mb: values.download_size_mb ?? 0,
      };

      if (editingCity) {
        updateCity.mutate(
          { id: editingCity.id, body },
          {
            onSuccess: () => {
              message.success('City updated');
              setDrawerOpen(false);
            },
          },
        );
      } else {
        createCity.mutate(body, {
          onSuccess: () => {
            message.success('City created');
            setDrawerOpen(false);
            resetPagination();
          },
        });
      }
    } catch {
      // form validation failed
    }
  };

  const columns: ColumnsType<City> = [
    {
      title: 'Name',
      dataIndex: 'name',
      render: (name: string, city) => (
        <span style={isDeleted(city) ? { opacity: 0.5 } : undefined}>{name}</span>
      ),
    },
    {
      title: 'Country',
      dataIndex: 'country',
    },
    {
      title: 'Center',
      render: (_, city) => `${city.center_lat.toFixed(4)}, ${city.center_lng.toFixed(4)}`,
    },
    {
      title: 'Radius (km)',
      dataIndex: 'radius_km',
    },
    {
      title: 'Status',
      render: (_, city) => (
        <Space size={4}>
          {isDeleted(city) ? (
            <Tag color="default">Deleted</Tag>
          ) : (
            <Tag color={city.is_active ? 'green' : 'red'}>
              {city.is_active ? 'Active' : 'Inactive'}
            </Tag>
          )}
        </Space>
      ),
    },
    {
      title: 'Download (MB)',
      dataIndex: 'download_size_mb',
    },
    {
      title: 'Created',
      dataIndex: 'created_at',
      render: (date: string) => new Date(date).toLocaleDateString(),
    },
    {
      title: 'Actions',
      render: (_, city) => {
        if (isDeleted(city)) {
          return (
            <Button
              size="small"
              icon={<UndoOutlined />}
              onClick={() => handleRestore(city)}
              loading={restoreCity.isPending}
            >
              Restore
            </Button>
          );
        }
        return (
          <Space>
            <Button size="small" icon={<EditOutlined />} onClick={() => openEditDrawer(city)}>
              Edit
            </Button>
            <Button
              size="small"
              danger={city.is_active}
              onClick={() => handleToggleActive(city)}
            >
              {city.is_active ? 'Deactivate' : 'Activate'}
            </Button>
            <Button
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => handleDelete(city)}
              loading={deleteCity.isPending}
            >
              Delete
            </Button>
          </Space>
        );
      },
    },
  ];

  const formLat = Form.useWatch('center_lat', form);
  const formLng = Form.useWatch('center_lng', form);
  const formRadius = Form.useWatch('radius_km', form);
  const mapLat = typeof formLat === 'number' && formLat >= -90 && formLat <= 90 ? formLat : 0;
  const mapLng = typeof formLng === 'number' && formLng >= -180 && formLng <= 180 ? formLng : 0;
  const mapRadius = typeof formRadius === 'number' && formRadius > 0 ? formRadius : 10;

  return (
    <div>
      <div style={{ marginBottom: 16, display: 'flex', justifyContent: 'space-between' }}>
        <Space>
          <Input.Search
            placeholder="Filter this page..."
            allowClear
            style={{ width: 250 }}
            onSearch={setSearchText}
            onChange={(e) => !e.target.value && setSearchText('')}
          />
          <Select
            value={visibilityFilter}
            style={{ width: 150 }}
            onChange={(value: 'all' | 'visible' | 'deleted') => setVisibilityFilter(value)}
            options={[
              { value: 'all', label: 'All rows' },
              { value: 'visible', label: 'Visible only' },
              { value: 'deleted', label: 'Deleted only' },
            ]}
          />
        </Space>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreateDrawer}>
          Create City
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={displayedCities}
        rowKey="id"
        loading={isLoading}
        rowClassName={(city) => (isDeleted(city) ? 'ant-table-row-deleted' : '')}
        pagination={{
          current: page,
          pageSize: perPage,
          total,
          showSizeChanger: true,
          pageSizeOptions: ['10', '20', '50'],
          onChange: (p, ps) => setPageAndSize(p, ps),
        }}
      />

      <Drawer
        title={editingCity ? `Edit: ${editingCity.name}` : 'Create City'}
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        width={520}
        extra={
          <Button
            type="primary"
            onClick={handleSubmit}
            loading={createCity.isPending || updateCity.isPending}
          >
            {editingCity ? 'Save' : 'Create'}
          </Button>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true, message: 'Name is required' }]}>
            <Input />
          </Form.Item>
          <Form.Item name="name_ru" label="Name (Russian)">
            <Input />
          </Form.Item>
          <Form.Item name="country" label="Country" rules={[{ required: true, message: 'Country is required' }]}>
            <Input />
          </Form.Item>
          <Space style={{ width: '100%' }} size="middle">
            <Form.Item
              name="center_lat"
              label="Latitude"
              rules={[
                { required: true, message: 'Required' },
                { type: 'number', min: -90, max: 90, message: 'Must be -90 to 90' },
              ]}
              style={{ flex: 1 }}
            >
              <InputNumber style={{ width: '100%' }} step={0.0001} />
            </Form.Item>
            <Form.Item
              name="center_lng"
              label="Longitude"
              rules={[
                { required: true, message: 'Required' },
                { type: 'number', min: -180, max: 180, message: 'Must be -180 to 180' },
              ]}
              style={{ flex: 1 }}
            >
              <InputNumber style={{ width: '100%' }} step={0.0001} />
            </Form.Item>
          </Space>
          <Form.Item
            name="radius_km"
            label="Radius (km)"
            rules={[
              { required: true, message: 'Required' },
              { type: 'number', min: 0.1, max: 100, message: 'Must be 0.1 to 100 km' },
            ]}
          >
            <InputNumber style={{ width: '100%' }} step={0.5} />
          </Form.Item>
          <Form.Item
            name="download_size_mb"
            label="Download Size (MB)"
            rules={[
              { required: true, message: 'Required' },
              { type: 'number', min: 0, message: 'Must be non-negative' },
            ]}
          >
            <InputNumber style={{ width: '100%' }} step={0.1} />
          </Form.Item>
          <Form.Item name="is_active" label="Active" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>

        <div style={{ marginTop: 16 }}>
          <div style={{ marginBottom: 8, fontWeight: 500 }}>Map Preview</div>
          <div style={{ height: 300, borderRadius: 8, overflow: 'hidden' }}>
            <MapContainer
              center={[mapLat, mapLng]}
              zoom={11}
              style={{ height: '100%', width: '100%' }}
            >
              <TileLayer
                attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
                url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
              />
              <Circle
                center={[mapLat, mapLng]}
                radius={mapRadius * 1000}
                pathOptions={{ color: '#1677ff', fillOpacity: 0.15 }}
              />
              <MapUpdater lat={mapLat} lng={mapLng} zoom={11} />
            </MapContainer>
          </div>
        </div>
      </Drawer>
      <style>{`
        .ant-table-row-deleted > td {
          opacity: 0.65;
        }
      `}</style>
    </div>
  );
}
