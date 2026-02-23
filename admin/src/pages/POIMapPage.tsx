import { Button, Card, Col, Descriptions, Row, Select, Spin, Tag, Typography } from 'antd';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import markerIcon2x from 'leaflet/dist/images/marker-icon-2x.png';
import markerIcon from 'leaflet/dist/images/marker-icon.png';
import markerShadow from 'leaflet/dist/images/marker-shadow.png';
import { useMemo, useState } from 'react';
import { MapContainer, Marker, Popup, TileLayer } from 'react-leaflet';
import MarkerClusterGroup from 'react-leaflet-cluster';
import { Link } from 'react-router-dom';
import { useCities, usePOIs } from '../hooks';
import type { POI, POIStatus, POIType } from '../types';

const { Title } = Typography;

// Fix default marker icons (leaflet webpack/vite issue)
L.Icon.Default.mergeOptions({
  iconUrl: markerIcon,
  iconRetinaUrl: markerIcon2x,
  shadowUrl: markerShadow,
});

const POI_TYPE_COLORS: Record<POIType, string> = {
  building: '#1677ff',
  street: '#13c2c2',
  park: '#52c41a',
  monument: '#fa8c16',
  church: '#722ed1',
  bridge: '#eb2f96',
  square: '#faad14',
  museum: '#2f54eb',
  district: '#8c8c8c',
  other: '#595959',
};

const STATUS_COLORS: Record<POIStatus, string> = {
  active: 'green',
  disabled: 'red',
  pending_review: 'orange',
};

function createColorIcon(color: string) {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="25" height="41" viewBox="0 0 25 41">
    <path d="M12.5 0C5.6 0 0 5.6 0 12.5c0 2.4.7 4.7 1.9 6.6L12.5 41l10.6-21.9c1.2-1.9 1.9-4.2 1.9-6.6C25 5.6 19.4 0 12.5 0z" fill="${color}" stroke="#fff" stroke-width="1"/>
    <circle cx="12.5" cy="12.5" r="5" fill="#fff"/>
  </svg>`;
  return L.divIcon({
    html: svg,
    className: '',
    iconSize: [25, 41],
    iconAnchor: [12, 41],
    popupAnchor: [1, -34],
  });
}

const poiTypeOptions = [
  { label: 'All types', value: '' },
  { label: 'Building', value: 'building' },
  { label: 'Street', value: 'street' },
  { label: 'Park', value: 'park' },
  { label: 'Monument', value: 'monument' },
  { label: 'Church', value: 'church' },
  { label: 'Bridge', value: 'bridge' },
  { label: 'Square', value: 'square' },
  { label: 'Museum', value: 'museum' },
  { label: 'District', value: 'district' },
  { label: 'Other', value: 'other' },
];

const poiStatusOptions = [
  { label: 'All statuses', value: '' },
  { label: 'Active', value: 'active' },
  { label: 'Disabled', value: 'disabled' },
  { label: 'Pending Review', value: 'pending_review' },
];

function POIPopup({ poi }: { poi: POI }) {
  return (
    <div style={{ width: 260 }}>
      <Descriptions column={1} size="small">
        <Descriptions.Item label="Name">{poi.name}</Descriptions.Item>
        {poi.name_ru && <Descriptions.Item label="Name (RU)">{poi.name_ru}</Descriptions.Item>}
        <Descriptions.Item label="Type">
          <Tag color={POI_TYPE_COLORS[poi.type]}>{poi.type}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label="Status">
          <Tag color={STATUS_COLORS[poi.status]}>{poi.status}</Tag>
        </Descriptions.Item>
        <Descriptions.Item label="Interest Score">{poi.interest_score}</Descriptions.Item>
        {poi.address && <Descriptions.Item label="Address">{poi.address}</Descriptions.Item>}
        <Descriptions.Item label="Coordinates">
          {poi.lat.toFixed(5)}, {poi.lng.toFixed(5)}
        </Descriptions.Item>
      </Descriptions>
      <div style={{ marginTop: 8, textAlign: 'right' }}>
        <Link to={`/pois/${poi.id}`}>
          <Button type="primary" size="small">
            View Details
          </Button>
        </Link>
      </div>
    </div>
  );
}

export default function POIMapPage() {
  const { data: cities = [], isLoading: citiesLoading } = useCities();
  const [userSelectedCityId, setUserSelectedCityId] = useState<number | null>(null);
  const [filterType, setFilterType] = useState<POIType | ''>('');
  const [filterStatus, setFilterStatus] = useState<POIStatus | ''>('');

  // Auto-select first city: use user selection if set, otherwise first available city
  const selectedCityId = userSelectedCityId ?? (cities.length > 0 ? cities[0].id : null);

  const {
    data: pois = [],
    isLoading: poisLoading,
    isFetching,
  } = usePOIs({
    cityId: selectedCityId,
    status: filterStatus || undefined,
    type: filterType || undefined,
  });

  const selectedCity = cities.find((c) => c.id === selectedCityId);
  const mapCenter: [number, number] = selectedCity
    ? [selectedCity.center_lat, selectedCity.center_lng]
    : [41.7151, 44.8271]; // Default: Tbilisi

  const cityOptions = cities.map((c) => ({
    label: `${c.name} (${c.country})`,
    value: c.id,
  }));

  const iconCache = useMemo(() => {
    const cache: Partial<Record<POIType, L.DivIcon>> = {};
    for (const [type, color] of Object.entries(POI_TYPE_COLORS)) {
      cache[type as POIType] = createColorIcon(color);
    }
    return cache;
  }, []);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 112px)' }}>
      <Row align="middle" justify="space-between" style={{ marginBottom: 16 }}>
        <Col>
          <Title level={2} style={{ margin: 0 }}>
            POI Map
          </Title>
        </Col>
        <Col>
          {pois.length > 0 && (
            <Tag style={{ fontSize: 14, padding: '4px 12px' }}>
              {pois.length} POI{pois.length !== 1 ? 's' : ''}
            </Tag>
          )}
        </Col>
      </Row>

      <Card size="small" style={{ marginBottom: 12 }}>
        <Row gutter={12}>
          <Col xs={24} sm={8}>
            <Select
              placeholder="Select city"
              style={{ width: '100%' }}
              options={cityOptions}
              value={selectedCityId}
              onChange={(v) => setUserSelectedCityId(v)}
              loading={citiesLoading}
            />
          </Col>
          <Col xs={12} sm={8}>
            <Select
              style={{ width: '100%' }}
              options={poiTypeOptions}
              value={filterType}
              onChange={(v) => setFilterType(v)}
            />
          </Col>
          <Col xs={12} sm={8}>
            <Select
              style={{ width: '100%' }}
              options={poiStatusOptions}
              value={filterStatus}
              onChange={(v) => setFilterStatus(v)}
            />
          </Col>
        </Row>
      </Card>

      <Spin spinning={poisLoading || isFetching} style={{ flex: 1 }}>
        <div style={{ flex: 1, minHeight: 400 }}>
          <MapContainer
            key={`${mapCenter[0]}-${mapCenter[1]}`}
            center={mapCenter}
            zoom={13}
            style={{ height: '100%', width: '100%', minHeight: 400, borderRadius: 8 }}
          >
            <TileLayer
              attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
              url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
            />
            <MarkerClusterGroup chunkedLoading>
              {pois.map((poi) => (
                <Marker
                  key={poi.id}
                  position={[poi.lat, poi.lng]}
                  icon={iconCache[poi.type] ?? iconCache.other}
                >
                  <Popup>
                    <POIPopup poi={poi} />
                  </Popup>
                </Marker>
              ))}
            </MarkerClusterGroup>
          </MapContainer>
        </div>
      </Spin>
    </div>
  );
}
