import { useState } from 'react';
import {
  Table,
  Button,
  Tag,
  Input,
  Select,
  Space,
  Drawer,
  Form,
  InputNumber,
  Switch,
  Modal,
  App,
} from 'antd';
import { PlusOutlined, EditOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import { MapContainer, TileLayer, Circle, useMap } from 'react-leaflet';
import type { ColumnsType } from 'antd/es/table';
import { useCities } from '../hooks';
import { useCityManagement } from '../hooks';
import type { City } from '../types';
import 'leaflet/dist/leaflet.css';

function MapUpdater({ lat, lng, zoom }: { lat: number; lng: number; zoom: number }) {
  const map = useMap();
  map.setView([lat, lng], zoom);
  return null;
}

export default function CitiesPage() {
  const { message } = App.useApp();
  const { data: cities = [], isLoading } = useCities();
  const { createCity, updateCity, toUpdateRequest } = useCityManagement();

  const [searchText, setSearchText] = useState('');
  const [activeFilter, setActiveFilter] = useState<string | undefined>(undefined);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editingCity, setEditingCity] = useState<City | null>(null);
  const [form] = Form.useForm();

  const filteredCities = cities.filter((city) => {
    const matchesSearch =
      !searchText || city.name.toLowerCase().includes(searchText.toLowerCase());
    const matchesActive =
      activeFilter === undefined ||
      (activeFilter === 'active' ? city.is_active : !city.is_active);
    return matchesSearch && matchesActive;
  });

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
            onError: (err) => message.error(err.message),
          },
        );
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
            onError: (err) => message.error(err.message),
          },
        );
      } else {
        createCity.mutate(body, {
          onSuccess: () => {
            message.success('City created');
            setDrawerOpen(false);
          },
          onError: (err) => message.error(err.message),
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
      sorter: (a, b) => a.name.localeCompare(b.name),
    },
    {
      title: 'Country',
      dataIndex: 'country',
      sorter: (a, b) => a.country.localeCompare(b.country),
    },
    {
      title: 'Center',
      render: (_, city) => `${city.center_lat.toFixed(4)}, ${city.center_lng.toFixed(4)}`,
    },
    {
      title: 'Radius (km)',
      dataIndex: 'radius_km',
      sorter: (a, b) => a.radius_km - b.radius_km,
    },
    {
      title: 'Status',
      dataIndex: 'is_active',
      render: (active: boolean) => (
        <Tag color={active ? 'green' : 'red'}>{active ? 'Active' : 'Inactive'}</Tag>
      ),
    },
    {
      title: 'Download (MB)',
      dataIndex: 'download_size_mb',
      sorter: (a, b) => a.download_size_mb - b.download_size_mb,
    },
    {
      title: 'Created',
      dataIndex: 'created_at',
      render: (date: string) => new Date(date).toLocaleDateString(),
      sorter: (a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime(),
    },
    {
      title: 'Actions',
      render: (_, city) => (
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
        </Space>
      ),
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
            placeholder="Search cities..."
            allowClear
            style={{ width: 250 }}
            onSearch={setSearchText}
            onChange={(e) => !e.target.value && setSearchText('')}
          />
          <Select
            placeholder="Status"
            allowClear
            style={{ width: 120 }}
            value={activeFilter}
            onChange={setActiveFilter}
            options={[
              { value: 'active', label: 'Active' },
              { value: 'inactive', label: 'Inactive' },
            ]}
          />
        </Space>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreateDrawer}>
          Create City
        </Button>
      </div>

      <Table
        columns={columns}
        dataSource={filteredCities}
        rowKey="id"
        loading={isLoading}
        pagination={{ pageSize: 20, showSizeChanger: true, pageSizeOptions: ['10', '20', '50'] }}
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
    </div>
  );
}
