import { expect, test as base, type Page, type Route } from '@playwright/test';

export const authFile = '.auth/admin.json';

const transparentPng = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAusB9VE3SkAAAAAASUVORK5CYII=',
  'base64',
);

export const mockData = {
  loginResponse: {
    data: {
      id: '550e8400-e29b-41d4-a716-446655440000',
      email: 'admin@example.com',
      name: 'Admin User',
      auth_provider: 'email' as const,
      provider_id: null,
      language_pref: 'en',
      is_anonymous: false,
      is_admin: true,
      deleted_at: null,
      deletion_scheduled_at: null,
      created_at: '2025-01-01T00:00:00Z',
      updated_at: '2025-01-01T00:00:00Z',
    },
    tokens: {
      access_token: 'mock-access-token',
      refresh_token: 'mock-refresh-token',
      expires_in: 3600,
    },
  },
  adminStats: {
    data: {
      cities_count: 5,
      pois_count: 120,
      stories_count: 340,
      reports_count: 8,
      new_reports_count: 2,
    },
  },
  cities: [
    {
      id: 1,
      name: 'Tbilisi',
      name_ru: 'Тбилиси',
      country: 'Georgia',
      center_lat: 41.7151,
      center_lng: 44.8271,
      radius_km: 15,
      is_active: true,
      download_size_mb: 24.5,
      created_at: '2025-01-01T00:00:00Z',
      updated_at: '2025-01-01T00:00:00Z',
    },
    {
      id: 2,
      name: 'Prague',
      name_ru: 'Прага',
      country: 'Czech Republic',
      center_lat: 50.0755,
      center_lng: 14.4378,
      radius_km: 12.5,
      is_active: true,
      download_size_mb: 19.2,
      created_at: '2025-02-01T00:00:00Z',
      updated_at: '2025-02-01T00:00:00Z',
    },
  ],
  pois: [
    {
      id: 10,
      city_id: 1,
      name: 'Narikala Fortress',
      name_ru: 'Крепость Нарикала',
      lat: 41.6878,
      lng: 44.8092,
      type: 'monument' as const,
      tags: null,
      address: 'Narikala Hill, Tbilisi',
      interest_score: 90,
      status: 'active' as const,
      created_at: '2025-01-15T00:00:00Z',
      updated_at: '2025-01-15T00:00:00Z',
    },
    {
      id: 11,
      city_id: 1,
      name: 'Peace Bridge',
      name_ru: null,
      lat: 41.748,
      lng: 44.858,
      type: 'bridge' as const,
      tags: null,
      address: 'Old Tbilisi, Tbilisi',
      interest_score: 85,
      status: 'active' as const,
      created_at: '2025-01-16T00:00:00Z',
      updated_at: '2025-01-16T00:00:00Z',
    },
  ],
  stories: [
    {
      id: 100,
      poi_id: 10,
      language: 'en',
      text: 'Narikala has watched over Tbilisi for centuries.',
      audio_url: null,
      duration_sec: 95,
      layer_type: 'general' as const,
      order_index: 0,
      is_inflation: false,
      confidence: 94,
      sources: null,
      status: 'active' as const,
      created_at: '2025-01-20T00:00:00Z',
      updated_at: '2025-01-20T00:00:00Z',
    },
    {
      id: 101,
      poi_id: 10,
      language: 'ru',
      text: 'Крепость Нарикала возвышается над старым городом.',
      audio_url: null,
      duration_sec: 110,
      layer_type: 'atmosphere' as const,
      order_index: 1,
      is_inflation: false,
      confidence: 88,
      sources: null,
      status: 'active' as const,
      created_at: '2025-01-21T00:00:00Z',
      updated_at: '2025-01-21T00:00:00Z',
    },
  ],
  reports: [
    {
      id: 1,
      story_id: 100,
      user_id: '550e8400-e29b-41d4-a716-446655440001',
      type: 'wrong_fact' as const,
      comment: 'The date mentioned is incorrect.',
      user_lat: null,
      user_lng: null,
      status: 'new' as const,
      resolved_at: null,
      created_at: '2025-03-01T12:00:00Z',
      poi_id: 10,
      poi_name: 'Narikala Fortress',
      story_language: 'en',
      story_status: 'active',
    },
    {
      id: 2,
      story_id: 101,
      user_id: '550e8400-e29b-41d4-a716-446655440002',
      type: 'wrong_location' as const,
      comment: 'Map pin is slightly off.',
      user_lat: 41.6938,
      user_lng: 44.8108,
      status: 'new' as const,
      resolved_at: null,
      created_at: '2025-03-02T14:30:00Z',
      poi_id: 11,
      poi_name: 'Peace Bridge',
      story_language: 'en',
      story_status: 'active',
    },
    {
      id: 3,
      story_id: 102,
      user_id: '550e8400-e29b-41d4-a716-446655440003',
      type: 'inappropriate_content' as const,
      comment: 'Resolved earlier.',
      user_lat: null,
      user_lng: null,
      status: 'resolved' as const,
      resolved_at: '2025-03-03T09:00:00Z',
      created_at: '2025-02-28T09:00:00Z',
      poi_id: 10,
      poi_name: 'Narikala Fortress',
      story_language: 'ru',
      story_status: 'disabled',
    },
  ],
  inflationJobs: [
    {
      id: 1,
      poi_id: 10,
      status: 'completed' as const,
      trigger_type: 'admin_manual' as const,
      segments_count: 3,
      max_segments: 3,
      started_at: '2025-03-01T08:00:00Z',
      completed_at: '2025-03-01T08:03:00Z',
      error_log: null,
      created_at: '2025-03-01T08:00:00Z',
    },
  ],
};

type AdminApiOptions = {
  statsStatus?: number;
};

class AdminApiMock {
  constructor(private readonly page: Page) {}

  async install(options: AdminApiOptions = {}) {
    const reports = mockData.reports.map((report) => ({ ...report }));

    await this.page.route('https://*.tile.openstreetmap.org/**', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'image/png',
        body: transparentPng,
      }),
    );

    await this.page.route('**/api/v1/auth/login', async (route) => {
      const body = route.request().postDataJSON() as { email?: string; password?: string };
      if (body.email === 'admin@example.com' && body.password === 'password123') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(mockData.loginResponse),
        });
        return;
      }

      await route.fulfill({
        status: 401,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Invalid credentials' }),
      });
    });

    await this.page.route('**/api/v1/admin/stats', (route) =>
      route.fulfill({
        status: options.statsStatus ?? 200,
        contentType: 'application/json',
        body:
          options.statsStatus && options.statsStatus >= 400
            ? JSON.stringify({ error: 'Server error' })
            : JSON.stringify(mockData.adminStats),
      }),
    );

    await this.page.route('**/api/v1/admin/cities*', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          items: mockData.cities,
          next_cursor: '',
          has_more: false,
        }),
      }),
    );

    await this.page.route('**/api/v1/pois?*', (route) => {
      const url = new URL(route.request().url());
      const cityId = Number(url.searchParams.get('city_id'));
      const type = url.searchParams.get('type');
      const status = url.searchParams.get('status');

      const items = mockData.pois.filter(
        (poi) =>
          poi.city_id === cityId &&
          (!type || poi.type === type) &&
          (!status || poi.status === status),
      );

      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          items,
          next_cursor: '',
          has_more: false,
        }),
      });
    });

    await this.page.route('**/api/v1/pois/*', (route) => {
      const id = Number(route.request().url().split('/').pop());
      const poi = mockData.pois.find((item) => item.id === id);

      return route.fulfill({
        status: poi ? 200 : 404,
        contentType: 'application/json',
        body: JSON.stringify(
          poi ? { data: poi } : { error: 'POI not found' },
        ),
      });
    });

    await this.page.route('**/api/v1/stories?*', (route) => {
      const url = new URL(route.request().url());
      const poiId = Number(url.searchParams.get('poi_id'));

      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          items: mockData.stories.filter((story) => story.poi_id === poiId),
          next_cursor: '',
          has_more: false,
        }),
      });
    });

    await this.page.route('**/api/v1/admin/pois/*/reports', (route) => {
      const poiId = Number(route.request().url().split('/').slice(-2, -1)[0]);
      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          items: reports.filter((report) => report.poi_id === poiId),
          next_cursor: '',
          has_more: false,
        }),
      });
    });

    await this.page.route('**/api/v1/admin/pois/*/inflation-jobs', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          items: mockData.inflationJobs,
          next_cursor: '',
          has_more: false,
        }),
      }),
    );

    await this.page.route('**/api/v1/admin/reports/*/disable-story', (route) => {
      const id = Number(route.request().url().split('/').slice(-2, -1)[0]);
      const target = reports.find((report) => report.id === id);
      if (!target) {
        return route.fulfill({
          status: 404,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Report not found' }),
        });
      }

      target.status = 'resolved';
      target.resolved_at = '2025-03-04T10:00:00Z';

      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            report: {
              id: target.id,
              story_id: target.story_id,
              user_id: target.user_id,
              type: target.type,
              comment: target.comment,
              user_lat: target.user_lat,
              user_lng: target.user_lng,
              status: target.status,
              resolved_at: target.resolved_at,
              created_at: target.created_at,
            },
            story: {
              id: target.story_id,
              poi_id: target.poi_id ?? 10,
              language: target.story_language ?? 'en',
              status: 'disabled',
            },
          },
        }),
      });
    });

    const fulfillReportsList = (route: Route) => {
      const url = new URL(route.request().url());
      const status = url.searchParams.get('status');
      const items = status ? reports.filter((report) => report.status === status) : reports;

      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          items,
          next_cursor: '',
          has_more: false,
        }),
      });
    };

    await this.page.route('**/api/v1/admin/reports', (route) => {
      if (route.request().method() === 'GET') {
        return fulfillReportsList(route);
      }

      return route.fallback();
    });

    await this.page.route('**/api/v1/admin/reports?*', (route) => {
      if (route.request().method() === 'GET') {
        return fulfillReportsList(route);
      }

      return route.fallback();
    });

    await this.page.route('**/api/v1/admin/reports/*', (route) => {
      const request = route.request();
      const url = new URL(request.url());

      if (request.method() === 'GET') {
        return fulfillReportsList(route);
      }

      const id = Number(url.pathname.split('/').pop());
      const target = reports.find((report) => report.id === id);
      if (!target) {
        return route.fulfill({
          status: 404,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Report not found' }),
        });
      }

      const body = request.postDataJSON() as { status?: typeof target.status };
      target.status = body.status ?? target.status;
      target.resolved_at =
        target.status === 'resolved' || target.status === 'dismissed'
          ? '2025-03-04T09:00:00Z'
          : null;

      return route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: {
            id: target.id,
            story_id: target.story_id,
            user_id: target.user_id,
            type: target.type,
            comment: target.comment,
            user_lat: target.user_lat,
            user_lng: target.user_lng,
            status: target.status,
            resolved_at: target.resolved_at,
            created_at: target.created_at,
          },
        }),
      });
    });
  }
}

class LoginPage {
  constructor(private readonly page: Page) {}

  async goto() {
    await this.page.goto('/login');
  }

  async login(email: string, password: string) {
    await this.page.getByTestId('login-email').fill(email);
    await this.page.getByTestId('login-password').fill(password);
    await this.page.getByTestId('login-submit').click();
  }
}

class Sidebar {
  constructor(private readonly page: Page) {}

  async goToReports() {
    await this.page.getByTestId('nav-reports').click();
  }

  async goToPoiMap() {
    await this.page.getByTestId('nav-poi-map').click();
  }
}

class Toasts {
  constructor(private readonly page: Page) {}

  async expectSuccess(text: string) {
    await expect(this.page.locator('.ant-message')).toContainText(text);
  }

  async expectError(text: string) {
    await expect(this.page.locator('.ant-message')).toContainText(text);
  }
}

export const test = base.extend<{
  adminApi: AdminApiMock;
  loginPage: LoginPage;
  sidebar: Sidebar;
  toasts: Toasts;
}>({
  adminApi: async ({ page }, runFixture) => {
    await runFixture(new AdminApiMock(page));
  },
  loginPage: async ({ page }, runFixture) => {
    await runFixture(new LoginPage(page));
  },
  sidebar: async ({ page }, runFixture) => {
    await runFixture(new Sidebar(page));
  },
  toasts: async ({ page }, runFixture) => {
    await runFixture(new Toasts(page));
  },
});

export { expect };
